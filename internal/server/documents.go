package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drive/drive/internal/model"
)

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		Name     string  `json:"name"`
		FolderID *string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Name is required")
		return
	}

	f := &model.File{
		UserID:       userID,
		Filename:     "_app_documents/" + req.Name + ".md",
		OriginalName: req.Name + ".md",
		Path:         "_app_documents",
		SizeBytes:    0,
		MimeType:     "text/markdown",
		SHA256:       "",
		MediaType:    model.MediaTypeFile,
		FolderID:     req.FolderID,
		IsAppManaged: true,
	}

	if err := s.file.FileStore.Create(f); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create document")
		return
	}

	if err := s.doc.DocumentStore.Create(f.ID, ""); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create document content")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           f.ID,
		"content":      "",
		"originalName": f.OriginalName,
	})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("file_id")
	userID := getUserID(r)

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found")
		return
	}

	doc, err := s.doc.DocumentStore.FindByFileID(fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Document content not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           fileID,
		"content":      doc.Content,
		"originalName": file.OriginalName,
	})
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("file_id")
	userID := getUserID(r)

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found")
		return
	}

	if !file.IsAppManaged {
		writeError(w, http.StatusBadRequest, "NOT_A_DOCUMENT", "File is not an app-managed document")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := s.doc.DocumentStore.UpdateContent(fileID, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update document")
		return
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.Content)))
	if err := s.file.FileStore.UpdateSizeAndHash(fileID, int64(len(req.Content)), hash); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update file metadata")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("file_id")
	userID := getUserID(r)

	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found")
		return
	}

	if err := s.file.FileStore.SoftDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete document")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
