package server

import (
	"encoding/json"
	"net/http"

	"github.com/drive/drive/internal/model"
	"github.com/go-chi/chi/v5"
)

func (c *CommentCtl) HandleToggleReaction(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	comment, err := c.CommentStore.FindByID(commentID)
	if err != nil || comment.FileID != fileID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Comment not found")
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Emoji == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Emoji is required")
		return
	}

	added, err := c.ReactionStore.Toggle(commentID, userID, req.Emoji)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to toggle reaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"emoji": req.Emoji,
		"added": added,
	})
}

func (c *CommentCtl) HandleRemoveReaction(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	emoji := chi.URLParam(r, "emoji")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := c.ReactionStore.Remove(commentID, userID, emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove reaction")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (c *CommentCtl) HandleGetReactions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	reactions, err := c.ReactionStore.FindByCommentID(commentID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get reactions")
		return
	}

	if reactions == nil {
		reactions = []model.ReactionGroup{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"reactions": reactions,
	})
}
