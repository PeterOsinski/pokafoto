package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	
	"path/filepath"
	"strconv"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleShareInfo(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	folder, err := s.file.FolderStore.FindByID(share.FolderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Shared folder not found")
		return
	}

	files, _, _, _ := s.file.FileStore.ListFilesByFolderID(share.FolderID, "", 0)
	fileCount := 0
	if files != nil {
		fileCount = len(files)
	}

	uploadedBytes, _ := s.share.ShareUploadStore.SumByShareID(share.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needs_password":    share.HasPassword,
		"permissions":       string(share.Permissions),
		"include_subdirs":   share.IncludeSubdirs,
		"upload_limit_bytes": share.UploadLimitBytes,
		"uploaded_bytes":    uploadedBytes,
		"expires_at": func() interface{} {
			if share.ExpiresAt == nil {
				return nil
			}
			return share.ExpiresAt.Format(time.RFC3339)
		}(),
		"folder_name": folder.Name,
		"file_count":  fileCount,
	})
}

func (s *Server) isFolderInShareTree(folderID, shareFolderID string) bool {
	if folderID == shareFolderID {
		return true
	}
	for i := 0; i < 50; i++ {
		f, err := s.file.FolderStore.FindByID(folderID)
		if err != nil || f == nil || f.ParentID == nil || *f.ParentID == "" {
			return false
		}
		if *f.ParentID == shareFolderID {
			return true
		}
		folderID = *f.ParentID
	}
	return false
}

func (s *Server) handleShareUnlock(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	if !share.HasPassword {
		expiryTime := time.Now().UTC().Add(24 * time.Hour)
		if share.ExpiresAt != nil && share.ExpiresAt.Before(expiryTime) {
			expiryTime = *share.ExpiresAt
		}
		sessionToken, err := s.generateShareSessionToken(share.ID, share.FolderID, share.Permissions, expiryTime)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate session token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"share_session_token": sessionToken,
			"expires_at":          expiryTime.Format(time.RFC3339),
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Password is required")
		return
	}

	if share.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_SHARE_PASSWORD", "Invalid password")
		return
	}

	expiryTime := time.Now().UTC().Add(24 * time.Hour)
	if share.ExpiresAt != nil && share.ExpiresAt.Before(expiryTime) {
		expiryTime = *share.ExpiresAt
	}
	sessionToken, err := s.generateShareSessionToken(share.ID, share.FolderID, share.Permissions, expiryTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate session token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"share_session_token": sessionToken,
		"expires_at":          expiryTime.Format(time.RFC3339),
	})
}

func (s *Server) handleShareListFiles(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareRead)
	if !ok {
		writeError(w, http.StatusForbidden, "SHARE_TOKEN_REQUIRED", "Share session token required — unlock the share first")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	targetFolderID := share.FolderID
	if subID := r.URL.Query().Get("folder_id"); subID != "" {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(subID, share.FolderID) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Folder is not part of this share tree")
			return
		}
		targetFolderID = subID
	}

	files, _, _, err := s.file.FileStore.ListFilesByFolderID(targetFolderID, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list files")
		return
	}

	items := make([]interface{}, 0, len(files))
	for _, f := range files {
		items = append(items, map[string]interface{}{
			"id":            f.ID,
			"original_name": f.OriginalName,
			"size_bytes":    f.SizeBytes,
			"mime_type":     f.MimeType,
			"media_type":    string(f.MediaType),
			"width":         f.Width,
			"height":        f.Height,
			"created_at":    f.CreatedAt.Format(time.RFC3339),
			"thumbnails":    buildThumbnailSet(f.ID, f.MediaType),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (s *Server) handleShareGetFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	fileID := r.PathValue("id")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareRead)
	if !ok {
		writeError(w, http.StatusForbidden, "SHARE_TOKEN_REQUIRED", "Share session token required")
		return
	}

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID == nil || *file.FolderID != share.FolderID {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(*file.FolderID, share.FolderID) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              file.ID,
		"original_name":   file.OriginalName,
		"filename":        file.Filename,
		"size_bytes":      file.SizeBytes,
		"mime_type":       file.MimeType,
		"media_type":      string(file.MediaType),
		"width":           file.Width,
		"height":          file.Height,
		"duration_sec":    file.DurationSec,
		"sha256":          file.SHA256,
		"taken_at":        file.TakenAt,
		"created_at":      file.CreatedAt.Format(time.RFC3339),
		"updated_at":      file.UpdatedAt.Format(time.RFC3339),
		"thumbnails":      buildThumbnailSet(file.ID, file.MediaType),
	})
}

func (s *Server) handleShareDownload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	fileID := r.PathValue("id")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareRead)
	if !ok {
		writeError(w, http.StatusForbidden, "SHARE_TOKEN_REQUIRED", "Share session token required")
		return
	}

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID == nil || *file.FolderID != share.FolderID {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(*file.FolderID, share.FolderID) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
	}

	if file.IsAppManaged {
		doc, err := s.doc.DocumentStore.FindByFileID(fileID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Document content not found")
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Length", strconv.Itoa(len(doc.Content)))
		w.Write([]byte(doc.Content))
		return
	}

	filePathStr := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := s.fs.Stat(filePathStr); err == nil {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Accept-Ranges", "bytes")
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		if file.SizeBytes > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
		}
		http.ServeFile(w, r, filePathStr)
		return
	}

	if s.cfg.Storage.S3.Enabled && s.file.Storage != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		stream, err := s.file.Storage.GetObjectStream(s3Key)
		if err != nil {
			slog.Warn("share download s3 stream failed", "file_id", fileID, "error", err)
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk or in S3")
			return
		}
		defer stream.Close()
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		if file.SizeBytes > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
		}
		io.Copy(w, stream)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk")
}

func (s *Server) handleShareThumbnail(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	fileID := r.PathValue("fileID")
	size := r.PathValue("size")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareRead)
	if !ok {
		writeError(w, http.StatusForbidden, "SHARE_TOKEN_REQUIRED", "Share session token required")
		return
	}

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID == nil || *file.FolderID != share.FolderID {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(*file.FolderID, share.FolderID) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
	}

	thumbPath := filepath.Join(s.cfg.ThumbnailsDir(), fileID, size)
	if _, err := s.fs.Stat(thumbPath); err == nil {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, thumbPath)
		return
	}

	s.fallbackThumbnail(w, r, fileID, size)
}

func (s *Server) handleShareUpload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareReadUpload)
	if !ok {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "This share does not permit uploads")
		return
	}

	if err := r.ParseMultipartForm(s.cfg.MaxFileSize()); err != nil {
		writeError(w, http.StatusBadRequest, "UPLOAD_ERROR", fmt.Sprintf("Failed to parse upload: %v", err))
		return
	}

	formFiles := r.MultipartForm.File["files"]
	if len(formFiles) == 0 {
		writeError(w, http.StatusBadRequest, "NO_FILES", "No files provided")
		return
	}

	var incomingTotal int64
	for _, fh := range formFiles {
		incomingTotal += fh.Size
	}

	if share.UploadLimitBytes != nil {
		used, _ := s.share.ShareUploadStore.SumByShareID(share.ID)
		if used+incomingTotal > *share.UploadLimitBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "SHARE_QUOTA_EXCEEDED",
				fmt.Sprintf("Upload would exceed share quota (%d used + %d incoming > %d limit)", used, incomingTotal, *share.UploadLimitBytes))
			return
		}
	}

	folder, err := s.file.FolderStore.FindByID(share.FolderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Shared folder not found")
		return
	}

	targetFolderID := share.FolderID
	if targetID := r.FormValue("folder_id"); targetID != "" {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(targetID, share.FolderID) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Folder is not part of this share tree")
			return
		}
		targetFolderID = targetID
	}

	batchID := uuid.New().String()
	jobs := make([]map[string]interface{}, 0, len(formFiles))

	for _, fh := range formFiles {
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

		tempDir := s.cfg.StoragePath("tmp")
		if err := s.fs.MkdirAll(tempDir, 0755); err != nil {
			file.Close()
			continue
		}
		tempFile, err := s.fs.CreateTemp(tempDir, "drive-share-upload-*")
		if err != nil {
			file.Close()
			continue
		}

		if _, err := io.Copy(tempFile, file); err != nil {
			file.Close()
			tempFile.Close()
			s.fs.Remove(tempFile.Name())
			continue
		}
		file.Close()
		tempFile.Close()

		job := &model.UploadJob{
			BatchID:          batchID,
			UserID:           folder.UserID,
			Filename:         fh.Filename,
			SizeBytes:        fh.Size,
			TempPath:         tempFile.Name(),
			FolderID:         &targetFolderID,
			SkipNameSizeDedup: true,
			Status:           model.JobStatusQueued,
			Progress:         0,
			UploadMode:       model.UploadModeFull,
		}

		if err := s.upload.UploadJobStore.Create(job); err != nil {
			s.fs.Remove(tempFile.Name())
			jobs = append(jobs, map[string]interface{}{
				"job_id":   uuid.New().String(),
				"filename": fh.Filename,
				"status":   "failed",
				"reason":   "job_create_error",
			})
			continue
		}

		if share.UploadLimitBytes != nil {
			s.share.ShareUploadStore.Create(share.ID, uuid.New().String(), fh.Size)
		}

		jobs = append(jobs, map[string]interface{}{
			"job_id":   job.ID,
			"filename": fh.Filename,
			"status":   "queued",
		})
	}

	s.upload.WorkerPool.NotifyJobsAvailable()

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"batch_id": batchID,
		"jobs":     jobs,
	})
}

func (s *Server) handleShareDeleteFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	fileID := r.PathValue("id")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareReadWrite)
	if !ok {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "This share does not permit deleting files")
		return
	}

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID == nil || *file.FolderID != share.FolderID {
		if !share.IncludeSubdirs || !s.isFolderInShareTree(*file.FolderID, share.FolderID) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
			return
		}
	}

	if err := s.file.FileStore.SoftDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleShareListFolders(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	if !share.IncludeSubdirs {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "This share does not include subdirectories")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareRead)
	if !ok {
		writeError(w, http.StatusForbidden, "SHARE_TOKEN_REQUIRED", "Share session token required")
		return
	}

	parentID := r.URL.Query().Get("parent_id")
	if parentID == "" {
		parentID = share.FolderID
	} else if !s.isFolderInShareTree(parentID, share.FolderID) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Folder is not part of this share tree")
		return
	}

	folders, err := s.file.FolderStore.FindByParentID(parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list folders")
		return
	}

	items := make([]map[string]interface{}, 0, len(folders))
	for _, f := range folders {
		files, _, _, _ := s.file.FileStore.ListFilesByFolderID(f.ID, "", 0)
		fileCount := 0
		if files != nil {
			fileCount = len(files)
		}
		items = append(items, map[string]interface{}{
			"id":         f.ID,
			"name":       f.Name,
			"parent_id":  f.ParentID,
			"file_count": fileCount,
			"created_at": f.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"folders": items,
	})
}

func (s *Server) handleShareCreateFolder(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	if !share.IncludeSubdirs {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "This share does not include subdirectories")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareReadWrite)
	if !ok {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "This share does not permit write operations")
		return
	}

	var req struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Folder name is required")
		return
	}

	parentID := share.FolderID
	if req.ParentID != "" {
		if !s.isFolderInShareTree(req.ParentID, share.FolderID) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Folder is not part of this share tree")
			return
		}
		parentID = req.ParentID
	}

	folder, err := s.file.FolderStore.FindByID(share.FolderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Shared folder not found")
		return
	}

	newFolder, err := s.file.FolderStore.Create(folder.UserID, req.Name, &parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create folder")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        newFolder.ID,
		"name":      newFolder.Name,
		"parent_id": newFolder.ParentID,
	})
}

func (s *Server) handleShareDeleteFolder(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	folderID := r.PathValue("id")

	share, err := s.share.FolderShareStore.FindByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "SHARE_NOT_FOUND", "Share link not found")
		return
	}

	if share.ExpiresAt != nil && time.Now().UTC().After(*share.ExpiresAt) {
		writeError(w, http.StatusGone, "SHARE_EXPIRED", "Share link has expired")
		return
	}

	if !share.IncludeSubdirs {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "This share does not include subdirectories")
		return
	}

	_, ok := s.checkShareAccess(r, model.ShareReadWrite)
	if !ok {
		writeError(w, http.StatusForbidden, "PERMISSION_DENIED", "This share does not permit write operations")
		return
	}

	if !s.isFolderInShareTree(folderID, share.FolderID) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Folder is not part of this share tree")
		return
	}

	if folderID == share.FolderID {
		writeError(w, http.StatusBadRequest, "CANNOT_DELETE_ROOT", "Cannot delete the root shared folder")
		return
	}

	if err := s.file.FolderStore.Delete(folderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete folder")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
