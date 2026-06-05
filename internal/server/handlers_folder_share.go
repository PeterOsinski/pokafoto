package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/drive/drive/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type createShareRequest struct {
	Permissions      string `json:"permissions"`
	IncludeSubdirs   bool   `json:"include_subdirs"`
	UploadLimitBytes *int64 `json:"upload_limit_bytes"`
	ExpiresAt        string `json:"expires_at"`
	Password         string `json:"password"`
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only folder owner can create shares")
		return
	}

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	permissions := model.SharePermission(req.Permissions)
	if permissions == "" {
		permissions = model.ShareRead
	}
	if permissions != model.ShareRead && permissions != model.ShareReadUpload && permissions != model.ShareReadWrite {
		writeError(w, http.StatusBadRequest, "INVALID_PERMISSIONS", "Permissions must be read, read_upload, or read_write")
		return
	}

	if len(req.Password) > 128 {
		writeError(w, http.StatusBadRequest, "PASSWORD_TOO_LONG", "Password must be 128 characters or fewer")
		return
	}

	share := &model.FolderShare{
		FolderID:         folderID,
		Permissions:      permissions,
		IncludeSubdirs:   req.IncludeSubdirs,
		UploadLimitBytes: req.UploadLimitBytes,
		HasPassword:      req.Password != "",
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_EXPIRY", "Invalid expires_at format, use ISO 8601")
			return
		}
		share.ExpiresAt = &t
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
			return
		}
		hashStr := string(hash)
		share.PasswordHash = &hashStr
	}

	if err := s.share.FolderShareStore.Create(share); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create share")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":               share.ID,
		"token":            share.Token,
		"share_url":        "/share/" + share.Token,
		"folder_id":        share.FolderID,
		"permissions":      string(share.Permissions),
		"include_subdirs":  share.IncludeSubdirs,
		"upload_limit_bytes": share.UploadLimitBytes,
		"expires_at":       req.ExpiresAt,
		"has_password":     share.HasPassword,
		"created_at":       share.CreatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleListShares(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	shares, err := s.share.FolderShareStore.ListByFolder(folderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list shares")
		return
	}

	items := make([]map[string]interface{}, 0, len(shares))
	for _, share := range shares {
		item := map[string]interface{}{
			"id":               share.ID,
			"token":            share.Token,
			"permissions":      string(share.Permissions),
			"include_subdirs":  share.IncludeSubdirs,
			"upload_limit_bytes": share.UploadLimitBytes,
			"has_password":     share.HasPassword,
			"created_at":       share.CreatedAt.Format(time.RFC3339),
			"updated_at":       share.UpdatedAt.Format(time.RFC3339),
		}
		if share.ExpiresAt != nil {
			item["expires_at"] = share.ExpiresAt.Format(time.RFC3339)
		}
		uploaded, _ := s.share.ShareUploadStore.SumByShareID(share.ID)
		item["uploaded_bytes"] = uploaded
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"shares": items,
	})
}

func (s *Server) handleUpdateShare(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	shareID := r.PathValue("shareId")
	userID := getUserID(r)

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	share, err := s.share.FolderShareStore.FindByID(shareID)
	if err != nil || share.FolderID != folderID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Share not found")
		return
	}

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	permissions := model.SharePermission(req.Permissions)
	if permissions == "" {
		permissions = share.Permissions
	}
	if permissions != model.ShareRead && permissions != model.ShareReadUpload && permissions != model.ShareReadWrite {
		writeError(w, http.StatusBadRequest, "INVALID_PERMISSIONS", "Permissions must be read, read_upload, or read_write")
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_EXPIRY", "Invalid expires_at format")
			return
		}
		expiresAt = &t
	} else {
		expiresAt = share.ExpiresAt
	}

	uploadLimitBytes := req.UploadLimitBytes
	if uploadLimitBytes == nil {
		uploadLimitBytes = share.UploadLimitBytes
	}

	includeSubdirs := req.IncludeSubdirs
	if !req.IncludeSubdirs {
		includeSubdirs = share.IncludeSubdirs
	}

	hasPassword := share.HasPassword
	var passwordHash *string = share.PasswordHash
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
			return
		}
		h := string(hash)
		passwordHash = &h
		hasPassword = true
	}

	if err := s.share.FolderShareStore.Update(shareID, permissions, includeSubdirs, uploadLimitBytes, expiresAt, hasPassword, passwordHash); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update share")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "updated",
	})
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	shareID := r.PathValue("shareId")
	userID := getUserID(r)

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	share, err := s.share.FolderShareStore.FindByID(shareID)
	if err != nil || share.FolderID != folderID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Share not found")
		return
	}

	if err := s.share.FolderShareStore.Delete(shareID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete share")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
