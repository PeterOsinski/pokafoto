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

	folder, err := s.file.FolderStore.Create(userID, req.Name, req.ParentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create folder")
		return
	}

	writeJSON(w, http.StatusCreated, folder)
}

func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	root, err := s.file.FolderStore.ListTree(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list folders")
		return
	}

	if root == nil {
		root = &model.FolderTreeNode{Children: []*model.FolderTreeNode{}}
	}

	writeJSON(w, http.StatusOK, root)
}

func (s *Server) handleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Name     *string `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Name == nil && req.ParentID == nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide name or parent_id")
		return
	}

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil || folder.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}

	if req.Name != nil {
		if *req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Name cannot be empty")
			return
		}
		if err := s.file.FolderStore.UpdateName(folderID, *req.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rename folder")
			return
		}
	}

	if req.ParentID != nil {
		if *req.ParentID != "" {
			if *req.ParentID == folderID {
				writeError(w, http.StatusBadRequest, "CIRCULAR_MOVE", "Cannot move folder into itself")
				return
			}
			isDesc, err := s.file.FolderStore.IsDescendant(*req.ParentID, folderID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate move")
				return
			}
			if isDesc {
				writeError(w, http.StatusBadRequest, "CIRCULAR_MOVE", "Cannot move folder into its own descendant")
				return
			}
			parent, err := s.file.FolderStore.FindByID(*req.ParentID)
			if err != nil || parent == nil || parent.UserID != userID {
				writeError(w, http.StatusBadRequest, "INVALID_PARENT", "Target folder not found")
				return
			}
			if err := s.file.FolderStore.UpdateParent(folderID, req.ParentID); err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move folder")
				return
			}
		} else {
			if err := s.file.FolderStore.UpdateParent(folderID, nil); err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move folder")
				return
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := s.file.FolderStore.FindByID(folderID)
	if err != nil || folder == nil || folder.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}

	result, err := s.file.FolderStore.DeleteRecursive(folderID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete folder")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted_files":   result.DeletedFiles,
		"deleted_folders": result.DeletedFolders,
	})
}

func (s *Server) handleRenameFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty name")
		return
	}

	if err := s.file.FileStore.Rename(fileID, userID, req.Name); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found or access denied")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
