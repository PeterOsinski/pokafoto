package server

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleSetFolderPassword(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.folderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only folder owner can set password")
		return
	}

	var req struct {
		Password     string `json:"password"`
		PasswordHint string `json:"password_hint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Password is required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
		return
	}

	expiresAt := time.Now().UTC().Add(s.folderPasswordExpiryDuration())
	existing, _ := s.folderPasswordStore.FindByFolderID(folderID)
	if existing != nil {
		s.folderPasswordStore.DeleteByFolderID(folderID)
	}

	fp, err := s.folderPasswordStore.Create(folderID, string(hash), req.PasswordHint, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to set folder password")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "Password set for folder",
		"expires_at": fp.ExpiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleRemoveFolderPassword(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.folderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only folder owner can remove password")
		return
	}

	if err := s.folderPasswordStore.DeleteByFolderID(folderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove folder password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUnlockFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.folderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	fp, err := s.folderPasswordStore.FindByFolderID(folderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder does not have a password")
		return
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		s.folderPasswordStore.DeleteByFolderID(folderID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":      "Folder password has expired, no unlock needed",
			"unlock_token": nil,
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Password is required")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(fp.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_FOLDER_PASSWORD", "Invalid password")
		return
	}

	expiresAt := time.Now().UTC().Add(s.folderPasswordExpiryDuration())
	s.folderPasswordStore.DeleteByFolderID(folderID)
	if _, err := s.folderPasswordStore.Create(folderID, fp.PasswordHash, fp.PasswordHint, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh unlock")
		return
	}

	unlockToken, err := s.generateFolderUnlockToken(folderID, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate unlock token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"unlock_token": unlockToken,
		"expires_at":   expiresAt.Format(time.RFC3339),
		"folder_id":    folderID,
	})
}

func (s *Server) handleGetFolderPasswordStatus(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.folderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	fp, err := s.folderPasswordStore.FindByFolderID(folderID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"has_password": false,
		})
		return
	}

	now := time.Now().UTC()
	active := now.Before(fp.ExpiresAt)
	if !active {
		s.folderPasswordStore.DeleteByFolderID(folderID)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_password":  active,
		"expires_at":    fp.ExpiresAt.Format(time.RFC3339),
		"password_hint": fp.PasswordHint,
	})
}
