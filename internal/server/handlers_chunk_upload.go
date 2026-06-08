package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func (c *UploadCtl) HandleChunkUpload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	resumeToken := r.Header.Get("X-Resume-Token")

	var job *model.UploadJob

	if resumeToken != "" {
		var err error
		job, err = c.UploadJobStore.FindByResumeToken(resumeToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to look up upload")
			return
		}
		if job == nil || job.UserID != userID {
			writeError(w, http.StatusNotFound, "UPLOAD_NOT_FOUND", "Upload session not found")
			return
		}
		if job.Status != model.JobStatusQueued {
			writeError(w, http.StatusConflict, "UPLOAD_CONFIGURED", "Upload session is no longer accepting chunks")
			return
		}
	} else {
		filename := r.Header.Get("X-Filename")
		totalSizeStr := r.Header.Get("X-Total-Size")
		totalChunksStr := r.Header.Get("X-Total-Chunks")
		folderID := r.Header.Get("X-Folder-ID")
		skipDedup := r.Header.Get("X-Skip-Name-Size-Dedup") == "true"

		if folderID != "" {
			if !c.CheckFolderAccess(folderID, getUserID(r), r) {
				writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock to upload")
				return
			}
		}

		if filename == "" || totalSizeStr == "" || totalChunksStr == "" {
			writeError(w, http.StatusBadRequest, "MISSING_HEADERS", "X-Filename, X-Total-Size, and X-Total-Chunks required for new upload")
			return
		}

		totalSize, err := strconv.ParseInt(totalSizeStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_SIZE", "X-Total-Size must be an integer")
			return
		}

		totalChunks, err := strconv.Atoi(totalChunksStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_CHUNKS", "X-Total-Chunks must be an integer")
			return
		}

		var folderIDPtr *string
		if folderID != "" {
			folderIDPtr = &folderID
		}

		chunkSizeMB := c.Cfg.Upload.ChunkSizeMB
		if chunkSizeMB <= 0 {
			chunkSizeMB = 5
		}
		chunkSize := int64(chunkSizeMB) * 1024 * 1024

		token := uuid.New().String()
		chunkDir := store.ChunkTempDir(c.Cfg.OriginalsDir())

		job = &model.UploadJob{
			BatchID:           "chunked-" + token,
			UserID:            userID,
			Filename:          filename,
			SizeBytes:         totalSize,
			TempPath:          chunkDir,
			FolderID:          folderIDPtr,
			SkipNameSizeDedup: skipDedup,
			Status:            model.JobStatusQueued,
			UploadMode:        model.UploadModeChunked,
			ResumeToken:       &token,
			TotalChunks:       &totalChunks,
		}
		cs := chunkSize
		job.ChunkSize = &cs

		if err := c.UploadJobStore.Create(job); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create upload session")
			return
		}
		resumeToken = token
	}

	chunkIndexStr := r.Header.Get("X-Chunk-Index")
	chunkSizeStr := r.Header.Get("X-Chunk-Size")
	chunkSHA256Header := r.Header.Get("X-Chunk-SHA256")

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		writeError(w, http.StatusBadRequest, "INVALID_CHUNK_INDEX", "X-Chunk-Index must be a non-negative integer")
		return
	}

	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil || chunkSize <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_CHUNK_SIZE", "X-Chunk-Size must be a positive integer")
		return
	}

	chunkDir := store.ChunkTempDir(c.Cfg.OriginalsDir())
	if err := c.FS.MkdirAll(chunkDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create chunk directory")
		return
	}

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%s-%d", job.ID, chunkIndex))

	chunkFile, err := c.FS.Create(chunkPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create chunk file")
		return
	}

	hasher := sha256.New()
	teeReader := io.TeeReader(r.Body, hasher)

	written, err := io.Copy(chunkFile, teeReader)
	if err != nil {
		chunkFile.Close()
		c.FS.Remove(chunkPath)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to write chunk")
		return
	}
	chunkFile.Close()

	if written != chunkSize {
		c.FS.Remove(chunkPath)
		writeError(w, http.StatusBadRequest, "SIZE_MISMATCH", fmt.Sprintf("Chunk size mismatch: expected %d, got %d", chunkSize, written))
		return
	}

	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if chunkSHA256Header != "" && actualSHA256 != chunkSHA256Header {
		c.FS.Remove(chunkPath)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error": map[string]interface{}{
				"code":     "CHUNK_HASH_MISMATCH",
				"message":  "Chunk SHA-256 mismatch",
				"expected": chunkSHA256Header,
				"actual":   actualSHA256,
			},
		})
		return
	}

	offset := int64(chunkIndex) * *job.ChunkSize
	if err := c.ChunkStore.CreateChunkRecord(job.ID, chunkIndex, written, offset, actualSHA256, chunkPath); err != nil {
		c.FS.Remove(chunkPath)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store chunk record")
		return
	}

	if job.TotalChunks != nil && *job.TotalChunks == 0 {
		totalChunks, _ := strconv.Atoi(r.Header.Get("X-Total-Chunks"))
		if totalChunks > 0 {
			job.TotalChunks = &totalChunks
			c.UploadJobStore.CompleteChunked(job.ID, totalChunks)
		}
	}

	stored, _ := c.ChunkStore.GetStoredChunks(job.ID)
	missing, _ := c.ChunkStore.FindMissingChunks(job.ID, *job.TotalChunks)

	nextChunk := chunkIndex + 1
	if len(missing) > 0 {
		nextChunk = missing[0]
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"upload_id":      job.ID,
		"resume_token":   job.ResumeToken,
		"chunk_index":    chunkIndex,
		"stored_chunks":  stored,
		"missing_chunks": missing,
		"next_chunk":     nextChunk,
	})
}

func (c *UploadCtl) HandleChunkUploadResume(w http.ResponseWriter, r *http.Request) {
	resumeToken := r.PathValue("resumeToken")
	if resumeToken == "" {
		resumeToken = r.Header.Get("X-Resume-Token")
	}
	if resumeToken == "" {
		writeError(w, http.StatusBadRequest, "MISSING_TOKEN", "Resume token required")
		return
	}

	userID := getUserID(r)

	job, err := c.UploadJobStore.FindByResumeToken(resumeToken)
	if err != nil || job == nil || job.UserID != userID {
		writeError(w, http.StatusNotFound, "UPLOAD_NOT_FOUND", "Upload session not found")
		return
	}

	stored, _ := c.ChunkStore.GetStoredChunks(job.ID)
	missing, _ := c.ChunkStore.FindMissingChunks(job.ID, *job.TotalChunks)

	nextChunk := 0
	if len(missing) > 0 {
		nextChunk = missing[0]
	}

	storedList, _ := json.Marshal(stored)

	w.Header().Set("X-Upload-Status", string(job.Status))
	w.Header().Set("X-Total-Chunks", strconv.Itoa(*job.TotalChunks))
	w.Header().Set("X-Stored-Count", strconv.Itoa(len(stored)))
	w.Header().Set("X-Next-Chunk", strconv.Itoa(nextChunk))
	w.Header().Set("X-Total-Size", strconv.FormatInt(job.SizeBytes, 10))
	w.Header().Set("X-Stored-Chunks", string(storedList))
	w.Header().Set("X-Upload-ID", job.ID)

	w.WriteHeader(http.StatusOK)
}

func (c *UploadCtl) HandleChunkUploadComplete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.UploadID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_UPLOAD_ID", "upload_id required")
		return
	}

	job, err := c.UploadJobStore.FindByID(req.UploadID)
	if err != nil || job == nil || job.UserID != userID {
		writeError(w, http.StatusNotFound, "UPLOAD_NOT_FOUND", "Upload session not found")
		return
	}

	if job.UploadMode != model.UploadModeChunked {
		writeError(w, http.StatusBadRequest, "NOT_CHUNKED", "Not a chunked upload")
		return
	}

	storedCount, _ := c.ChunkStore.GetStoredChunkCount(job.ID)
	missing, _ := c.ChunkStore.FindMissingChunks(job.ID, *job.TotalChunks)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"upload_id":      job.ID,
		"batch_id":       job.BatchID,
		"job_id":         job.ID,
		"status":         "assembling",
		"stored_chunks":  storedCount,
		"missing_chunks": missing,
		"total_chunks":   *job.TotalChunks,
	})

	if len(missing) == 0 {
		c.UploadJobStore.SetStatus(job.ID, model.JobStatusReady)
		c.WorkerPool.NotifyJobsAvailable()
		slog.Info("chunked upload ready for assembly", "upload_id", job.ID, "filename", job.Filename, "total_chunks", *job.TotalChunks)
	}
}

func (c *UploadCtl) HandleProgressFlush(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
