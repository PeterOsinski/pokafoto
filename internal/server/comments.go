package server

import (
	"encoding/json"
	"net/http"

	"github.com/drive/drive/internal/model"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListComments(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	comments, err := s.comment.CommentStore.FindByFileID(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list comments")
		return
	}

	type commentResponse struct {
		ID        string              `json:"id"`
		FileID    string              `json:"file_id"`
		UserID    string              `json:"user_id"`
		Username  string              `json:"username"`
		Content   string              `json:"content"`
		CreatedAt string              `json:"created_at"`
		UpdatedAt string              `json:"updated_at"`
		Reactions []model.ReactionGroup `json:"reactions,omitempty"`
	}

	items := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		username := ""
		if u, _ := s.auth.UserStore.FindByID(c.UserID); u != nil {
			username = u.Username
		}

		reactions, _ := s.comment.ReactionStore.FindByCommentID(c.ID, userID)
		if reactions == nil {
			reactions = []model.ReactionGroup{}
		}

		items = append(items, commentResponse{
			ID:        c.ID,
			FileID:    c.FileID,
			UserID:    c.UserID,
			Username:  username,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format(timeRFC3339),
			UpdatedAt: c.UpdatedAt.Format(timeRFC3339),
			Reactions: reactions,
		})
	}

	if items == nil {
		items = []commentResponse{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"comments": items,
	})
}

func (s *Server) handleAddComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := s.checkCommentWriteAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found or you don't have permission to comment")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Comment content is required")
		return
	}

	comment, err := s.comment.CommentStore.Create(fileID, userID, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create comment")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         comment.ID,
		"file_id":    comment.FileID,
		"user_id":    comment.UserID,
		"content":    comment.Content,
		"created_at": comment.CreatedAt.Format(timeRFC3339),
	})
}

func (s *Server) handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	comment, err := s.comment.CommentStore.FindByID(commentID)
	if err != nil || comment.UserID != userID || comment.FileID != fileID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Comment not found")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Comment content is required")
		return
	}

	if err := s.comment.CommentStore.Update(commentID, userID, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update comment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	comment, err := s.comment.CommentStore.FindByID(commentID)
	if err != nil || comment.UserID != userID || comment.FileID != fileID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Comment not found")
		return
	}

	if err := s.comment.CommentStore.Delete(commentID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comment")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) checkFileAccess(fileID, userID string) bool {
	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.IsDeleted {
		return false
	}

	if file.UserID == userID {
		return true
	}

	hasAccess, err := s.album.AlbumItemStore.HasSharedAccess(fileID, userID)
	if err != nil {
		return false
	}
	return hasAccess
}

func (s *Server) checkCommentWriteAccess(fileID, userID string) bool {
	file, err := s.file.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.IsDeleted {
		return false
	}

	if file.UserID == userID {
		return true
	}

	perm, err := s.album.AlbumItemStore.GetSharedPermission(fileID, userID)
	if err != nil {
		return false
	}

	return perm == "comment" || perm == "edit"
}
