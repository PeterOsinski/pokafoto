package server

import (
	"encoding/json"
	"net/http"

	"github.com/drive/drive/internal/model"
	"github.com/go-chi/chi/v5"
)

func (c *CommentCtl) HandleListComments(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	comments, err := c.CommentStore.FindByFileID(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list comments")
		return
	}

	type commentResponse struct {
		ID        string               `json:"id"`
		FileID    string               `json:"file_id"`
		UserID    string               `json:"user_id"`
		Username  string               `json:"username"`
		Content   string               `json:"content"`
		CreatedAt string               `json:"created_at"`
		UpdatedAt string               `json:"updated_at"`
		Reactions []model.ReactionGroup `json:"reactions,omitempty"`
	}

	items := make([]commentResponse, 0, len(comments))
	for _, com := range comments {
		username := ""
		if u, _ := c.UserStore.FindByID(com.UserID); u != nil {
			username = u.Username
		}

		reactions, _ := c.ReactionStore.FindByCommentID(com.ID, userID)
		if reactions == nil {
			reactions = []model.ReactionGroup{}
		}

		items = append(items, commentResponse{
			ID:        com.ID,
			FileID:    com.FileID,
			UserID:    com.UserID,
			Username:  username,
			Content:   com.Content,
			CreatedAt: com.CreatedAt.Format(timeRFC3339),
			UpdatedAt: com.UpdatedAt.Format(timeRFC3339),
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

func (c *CommentCtl) HandleAddComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := c.CheckCommentWriteAccess(fileID, userID)
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

	comment, err := c.CommentStore.Create(fileID, userID, req.Content)
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

func (c *CommentCtl) HandleUpdateComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	comment, err := c.CommentStore.FindByID(commentID)
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

	if err := c.CommentStore.Update(commentID, userID, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update comment")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (c *CommentCtl) HandleDeleteComment(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	comment, err := c.CommentStore.FindByID(commentID)
	if err != nil || comment.UserID != userID || comment.FileID != fileID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Comment not found")
		return
	}

	if err := c.CommentStore.Delete(commentID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comment")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (c *CommentCtl) CheckFileAccess(fileID, userID string) bool {
	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.IsDeleted {
		return false
	}

	if file.UserID == userID {
		return true
	}

	hasAccess, err := c.AlbumItemStore.HasSharedAccess(fileID, userID)
	if err != nil {
		return false
	}
	return hasAccess
}

func (c *CommentCtl) CheckCommentWriteAccess(fileID, userID string) bool {
	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.IsDeleted {
		return false
	}

	if file.UserID == userID {
		return true
	}

	perm, err := c.AlbumItemStore.GetSharedPermission(fileID, userID)
	if err != nil {
		return false
	}

	return perm == "comment" || perm == "edit"
}
