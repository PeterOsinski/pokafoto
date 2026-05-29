package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/drive/drive/internal/store"
	"github.com/drive/drive/internal/worker"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	if err := r.ParseMultipartForm(s.cfg.MaxFileSize()); err != nil {
		writeError(w, http.StatusBadRequest, "UPLOAD_ERROR", fmt.Sprintf("Failed to parse upload: %v", err))
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "NO_FILES", "No files provided")
		return
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

		tempFile, err := os.CreateTemp("", "drive-upload-*")
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
			os.Remove(tempFile.Name())
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "write_error",
			})
			continue
		}
		file.Close()
		tempFile.Seek(0, 0)

		job := s.workerPool.Enqueue(batchID, userID, fh.Filename, fh.Size, tempFile.Name(), folderIDFromForm(r))

		tempFile.Close()

		jobs = append(jobs, map[string]interface{}{
			"job_id":    job.JobID,
			"filename":  fh.Filename,
			"status":    "queued",
		})
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

func (s *Server) handleUploadStatus(w http.ResponseWriter, r *http.Request) {
	batchID := r.PathValue("batchID")
	batch := s.workerPool.GetBatch(batchID)
	if batch == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Batch not found")
		return
	}

	jobs := make([]map[string]interface{}, 0, len(batch.Jobs))
	completed := 0
	failed := 0
	for _, j := range batch.Jobs {
		if j.Status == worker.JobCompleted || j.Status == worker.JobSkipped {
			completed++
		}
		if j.Status == worker.JobFailed {
			failed++
		}
		jobMap := map[string]interface{}{
			"job_id":   j.JobID,
			"filename": j.Filename,
			"status":   j.Status,
		}
		if j.FileID != "" {
			jobMap["file_id"] = j.FileID
		}
		if j.Reason != "" {
			jobMap["reason"] = j.Reason
		}
		if j.Error != "" {
			jobMap["error"] = j.Error
		}
		jobs = append(jobs, jobMap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"batch_id":  batchID,
		"total":     batch.Total,
		"completed": completed,
		"failed":    failed,
		"jobs":      jobs,
	})
}

func (s *Server) handleUploadCheck(w http.ResponseWriter, r *http.Request) {
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

	records := make([]store.FileRecord, 0, len(input))
	for _, item := range input {
		records = append(records, store.FileRecord{
			OriginalName: item.Filename,
			SizeBytes:    item.Size,
		})
	}

	existing, err := s.fileStore.FindByNameAndSizeBatch(records)
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

func (s *Server) handleUploadWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userID := getUserID(r)
	batchID := r.URL.Query().Get("batch")
	wsID := uuid.New().String()

	var ch chan *worker.UploadJob

	if batchID != "" {
		ch = s.workerPool.Subscribe(batchID, wsID)
		defer s.workerPool.Unsubscribe(batchID, wsID)
		s.sendBatchSnapshot(conn, batchID, userID)
	} else {
		ch = s.workerPool.SubscribeUser(userID, wsID)
		defer s.workerPool.UnsubscribeUser(userID, wsID)
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

func workerJobToMsg(job *worker.UploadJob) map[string]interface{} {
	msg := map[string]interface{}{
		"job_id":   job.JobID,
		"filename": job.Filename,
		"status":   job.Status,
		"progress": job.Progress,
	}
	if job.FileID != "" {
		msg["file_id"] = job.FileID
	}
	if job.Error != "" {
		msg["error"] = job.Error
	}
	if job.Stage != "" {
		msg["stage"] = job.Stage
	}
	return msg
}

func (s *Server) sendBatchSnapshot(conn *websocket.Conn, batchID, userID string) {
	batch := s.workerPool.GetBatch(batchID)
	if batch == nil {
		return
	}
	for _, j := range batch.Jobs {
		if j.UserID != userID {
			continue
		}
		data, _ := json.Marshal(workerJobToMsg(j))
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (s *Server) handleUploadWSWithToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token query parameter")
		return
	}

	if !s.validateTokenAndSetContext(w, r, token) {
		return
	}

	s.handleUploadWS(w, r)
}

func (s *Server) validateTokenAndSetContext(w http.ResponseWriter, r *http.Request, tokenString string) bool {
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
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
