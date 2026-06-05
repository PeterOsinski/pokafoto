package server

import (
	"encoding/json"
	"net/http"

	"github.com/drive/drive/internal/model"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleToggleReaction(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	comment, err := s.comment.CommentStore.FindByID(commentID)
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

	added, err := s.comment.ReactionStore.Toggle(commentID, userID, req.Emoji)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to toggle reaction")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"emoji": req.Emoji,
		"added": added,
	})
}

func (s *Server) handleRemoveReaction(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	emoji := chi.URLParam(r, "emoji")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := s.comment.ReactionStore.Remove(commentID, userID, emoji); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove reaction")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleGetReactions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	commentID := chi.URLParam(r, "commentId")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	reactions, err := s.comment.ReactionStore.FindByCommentID(commentID, userID)
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
