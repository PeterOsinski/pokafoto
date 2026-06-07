package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (c *UploadCtl) HandleUpload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	if err := r.ParseMultipartForm(c.Cfg.MaxFileSize()); err != nil {
		writeError(w, http.StatusBadRequest, "UPLOAD_ERROR", fmt.Sprintf("Failed to parse upload: %v", err))
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "NO_FILES", "No files provided")
		return
	}

	var incomingTotal int64
	for _, fh := range files {
		incomingTotal += fh.Size
	}
	used, _ := c.UserStore.GetUsedSpace(userID)
	user, _ := c.UserStore.FindByID(userID)
	if user != nil && user.SpaceQuota != nil && used+incomingTotal > *user.SpaceQuota {
		writeError(w, http.StatusRequestEntityTooLarge, "QUOTA_EXCEEDED",
			fmt.Sprintf("Upload would exceed space quota (%d used + %d incoming > %d limit)", used, incomingTotal, *user.SpaceQuota))
		return
	}

	folderID := folderIDFromForm(r)
	if folderID != nil {
		if !c.CheckFolderAccess(*folderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock to upload")
			return
		}
	}

	batchID := uuid.New().String()
	jobs := make([]map[string]interface{}, 0, len(files))

	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "cannot_open",
			})
			continue
		}

		tempDir := c.Cfg.StoragePath("tmp")
		if err := c.FS.MkdirAll(tempDir, 0755); err != nil {
			file.Close()
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "temp_file_error",
			})
			continue
		}
		tempFile, err := c.FS.CreateTemp(tempDir, "drive-upload-*")
		if err != nil {
			file.Close()
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "temp_file_error",
			})
			continue
		}

		if _, err := io.Copy(tempFile, file); err != nil {
			file.Close()
			tempFile.Close()
			c.FS.Remove(tempFile.Name())
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "write_error",
			})
			continue
		}
		file.Close()
		tempFile.Close()

		skipDedup := r.FormValue("skip_name_size_dedup") == "true"

		job := &model.UploadJob{
			BatchID:           batchID,
			UserID:            userID,
			Filename:          fh.Filename,
			SizeBytes:         fh.Size,
			TempPath:          tempFile.Name(),
			FolderID:          folderIDFromForm(r),
			SkipNameSizeDedup: skipDedup,
			Status:            model.JobStatusQueued,
		}

		if err := c.UploadJobStore.Create(job); err != nil {
			c.FS.Remove(tempFile.Name())
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "db_error",
			})
			continue
		}

		jobs = append(jobs, map[string]interface{}{
			"job_id":   job.ID,
			"filename": fh.Filename,
			"status":   "queued",
		})
	}

	if len(files) > 0 {
		c.WorkerPool.NotifyJobsAvailable()
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"batch_id": batchID,
		"jobs":     jobs,
	})
}

func folderIDFromForm(r *http.Request) *string {
	folderID := r.FormValue("folder_id")
	if folderID == "" {
		return nil
	}
	return &folderID
}

func (c *UploadCtl) HandleUploadStatus(w http.ResponseWriter, r *http.Request) {
	batchID := r.PathValue("batchID")

	allJobs, err := c.UploadJobStore.ListByBatch(batchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get upload status")
		return
	}

	jobs := make([]map[string]interface{}, 0, len(allJobs))
	completed := 0
	failed := 0
	for _, j := range allJobs {
		if j.Status == model.JobStatusCompleted || j.Status == model.JobStatusSkipped {
			completed++
		}
		if j.Status == model.JobStatusFailed {
			failed++
		}
		jobMap := map[string]interface{}{
			"job_id":   j.ID,
			"filename": j.Filename,
			"status":   j.Status,
		}
		if j.FileID != nil && *j.FileID != "" {
			jobMap["file_id"] = *j.FileID
		}
		if j.Reason != nil && *j.Reason != "" {
			jobMap["reason"] = *j.Reason
		}
		if j.Error != nil && *j.Error != "" {
			jobMap["error"] = *j.Error
		}
		jobs = append(jobs, jobMap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"batch_id":  batchID,
		"total":     len(allJobs),
		"completed": completed,
		"failed":    failed,
		"jobs":      jobs,
	})
}

func (c *UploadCtl) HandleUploadActiveJobs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	jobs, err := c.UploadJobStore.ListActiveByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list active jobs")
		return
	}

	result := make([]map[string]interface{}, 0, len(jobs))
	for _, j := range jobs {
		jobMap := map[string]interface{}{
			"job_id":   j.ID,
			"batch_id": j.BatchID,
			"filename": j.Filename,
			"status":   j.Status,
			"progress": j.Progress,
		}
		if j.FileID != nil && *j.FileID != "" {
			jobMap["file_id"] = *j.FileID
		}
		if j.Stage != nil {
			jobMap["stage"] = *j.Stage
		}
		if j.FolderID != nil {
			jobMap["folder_id"] = *j.FolderID
		}
		if j.Error != nil {
			jobMap["error"] = *j.Error
		}
		if j.Reason != nil {
			jobMap["reason"] = *j.Reason
		}
		result = append(result, jobMap)
	}

	if result == nil {
		result = make([]map[string]interface{}, 0)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs": result,
	})
}

func (c *UploadCtl) HandleUploadCheck(w http.ResponseWriter, r *http.Request) {
	var input []struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid request body")
		return
	}

	if len(input) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"duplicates": []interface{}{}})
		return
	}

	userID := getUserID(r)

	records := make([]store.FileRecord, 0, len(input))
	for _, item := range input {
		records = append(records, store.FileRecord{
			OriginalName: item.Filename,
			SizeBytes:    item.Size,
		})
	}

	existing, err := c.FileStore.FindByNameAndSizeBatch(userID, records)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DEDUP_ERROR", "Failed to check duplicates")
		return
	}

	duplicates := make([]map[string]interface{}, 0, len(existing))
	for _, f := range existing {
		duplicates = append(duplicates, map[string]interface{}{
			"filename": f.OriginalName,
			"file_id":  f.ID,
			"size":     f.SizeBytes,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"duplicates": duplicates,
	})
}

func (c *UploadCtl) HandleUploadWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userID := getUserID(r)
	batchID := r.URL.Query().Get("batch")
	wsID := uuid.New().String()

	var ch chan *model.UploadJob

	if batchID != "" {
		ch = c.WorkerPool.Subscribe(batchID, wsID)
		defer c.WorkerPool.Unsubscribe(batchID, wsID)
		c.sendBatchSnapshot(conn, batchID, userID)
	} else {
		ch = c.WorkerPool.SubscribeUser(userID, wsID)
		defer c.WorkerPool.UnsubscribeUser(userID, wsID)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case job, ok := <-ch:
			if !ok {
				return
			}
			if job.UserID != userID {
				continue
			}
			data, _ := json.Marshal(workerJobToMsg(job))
			conn.WriteMessage(websocket.TextMessage, data)
		case <-done:
			return
		}
	}
}

func workerJobToMsg(job *model.UploadJob) map[string]interface{} {
	msg := map[string]interface{}{
		"job_id":   job.ID,
		"batch_id": job.BatchID,
		"filename": job.Filename,
		"status":   job.Status,
		"progress": job.Progress,
	}
	if job.FileID != nil {
		msg["file_id"] = *job.FileID
	}
	if job.Error != nil {
		msg["error"] = *job.Error
	}
	if job.Reason != nil {
		msg["reason"] = *job.Reason
	}
	if job.Stage != nil {
		msg["stage"] = *job.Stage
	}
	if job.FolderID != nil {
		msg["folder_id"] = *job.FolderID
	}
	return msg
}

func (c *UploadCtl) sendBatchSnapshot(conn *websocket.Conn, batchID, userID string) {
	jobs, err := c.UploadJobStore.ListByBatch(batchID)
	if err != nil {
		return
	}
	for _, j := range jobs {
		if j.UserID != userID {
			continue
		}
		data, _ := json.Marshal(workerJobToMsg(j))
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (c *UploadCtl) HandleUploadWSWithToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token query parameter")
		return
	}

	if !c.ValidateTokenAndSetContext(w, r, token) {
		return
	}

	c.HandleUploadWS(w, r)
}

func (c *UploadCtl) ValidateTokenAndSetContext(w http.ResponseWriter, r *http.Request, tokenString string) bool {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(c.Cfg.Auth.JWTSecret), nil
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
		return false
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
		return false
	}

	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)

	if userID == "" || role == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token payload")
		return false
	}

	ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
	ctx = context.WithValue(ctx, contextKeyUserRole, role)
	*r = *r.WithContext(ctx)

	return true
}

func (c *UploadCtl) CheckFolderAccess(folderID, userID string, r *http.Request) bool {
	fp, err := c.FolderPwStore.FindByFolderID(folderID)
	if err != nil {
		return true
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		c.FolderPwStore.DeleteByFolderID(folderID)
		return true
	}

	if _, err := c.FolderStore.FindByID(folderID); err != nil {
		return false
	}

	unlockToken := r.Header.Get("X-Folder-Unlock-Token")
	if unlockToken == "" {
		unlockToken = r.URL.Query().Get("folder_unlock_token")
	}
	if unlockToken != "" {
		unlockedFolderID, ok := c.ParseFolderUnlockToken(unlockToken)
		if ok && unlockedFolderID == folderID {
			return true
		}
	}

	return false
}

func (c *UploadCtl) folderUnlockSecret() string {
	return c.Cfg.Auth.JWTSecret + ":folder_unlock"
}

func (c *UploadCtl) ParseFolderUnlockToken(tokenStr string) (string, bool) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(c.folderUnlockSecret()), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	sub, _ := claims["sub"].(string)
	if sub != "folder_unlock" {
		return "", false
	}

	folderID, _ := claims["folder_id"].(string)
	return folderID, folderID != ""
}
