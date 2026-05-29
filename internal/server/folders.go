package server

import (
	"encoding/json"
	"net/http"

	"github.com/drive/drive/internal/model"
)

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty name")
		return
	}

	folder, err := s.folderStore.Create(userID, req.Name, req.ParentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create folder")
		return
	}

	writeJSON(w, http.StatusCreated, folder)
}

func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	root, err := s.folderStore.ListTree(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list folders")
		return
	}

	if root == nil {
		root = &model.FolderTreeNode{Children: []*model.FolderTreeNode{}}
	}

	writeJSON(w, http.StatusOK, root)
}

func (s *Server) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty name")
		return
	}

	if err := s.folderStore.UpdateName(folderID, req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rename folder")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")

	if err := s.folderStore.Delete(folderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete folder")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
