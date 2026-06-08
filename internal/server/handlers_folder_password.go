package server

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (c *FileCtl) HandleSetFolderPassword(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := c.FolderStore.FindByID(folderID)
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), c.Cfg.Auth.AuthBcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to hash password")
		return
	}

	expiresAt := time.Now().UTC().Add(c.FolderPasswordExpiryDuration())
	existing, _ := c.FolderPwStore.FindByFolderID(folderID)
	if existing != nil {
		c.FolderPwStore.DeleteByFolderID(folderID)
	}

	fp, err := c.FolderPwStore.Create(folderID, string(hash), req.PasswordHint, expiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to set folder password")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "Password set for folder",
		"expires_at": fp.ExpiresAt.Format(time.RFC3339),
	})
}

func (c *FileCtl) HandleRemoveFolderPassword(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := c.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only folder owner can remove password")
		return
	}

	if err := c.FolderPwStore.DeleteByFolderID(folderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove folder password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleUnlockFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := c.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	fp, err := c.FolderPwStore.FindByFolderID(folderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder does not have a password")
		return
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		c.FolderPwStore.DeleteByFolderID(folderID)
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

	expiresAt := time.Now().UTC().Add(c.FolderPasswordExpiryDuration())
	c.FolderPwStore.DeleteByFolderID(folderID)
	if _, err := c.FolderPwStore.Create(folderID, fp.PasswordHash, fp.PasswordHint, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh unlock")
		return
	}

	unlockToken, err := c.GenerateFolderUnlockToken(folderID, expiresAt)
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

func (c *FileCtl) HandleGetFolderPasswordStatus(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := c.FolderStore.FindByID(folderID)
	if err != nil || folder == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}
	if folder.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your folder")
		return
	}

	fp, err := c.FolderPwStore.FindByFolderID(folderID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"has_password": false,
		})
		return
	}

	now := time.Now().UTC()
	active := now.Before(fp.ExpiresAt)
	if !active {
		c.FolderPwStore.DeleteByFolderID(folderID)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_password":  active,
		"expires_at":    fp.ExpiresAt.Format(time.RFC3339),
		"password_hint": fp.PasswordHint,
	})
}
