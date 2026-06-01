package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	tags, err := s.tagStore.Search(q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list tags")
		return
	}

	type tagResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	items := make([]tagResponse, 0, len(tags))
	for _, t := range tags {
		items = append(items, tagResponse{ID: t.ID, Name: t.Name})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tags": items,
	})
}

func (s *Server) handleGetFileTags(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	tags, err := s.tagStore.FindByFileID(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get file tags")
		return
	}

	type tagResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	items := make([]tagResponse, 0, len(tags))
	for _, t := range tags {
		items = append(items, tagResponse{ID: t.ID, Name: t.Name})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tags": items,
	})
}

func (s *Server) handleAddFileTags(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	var req struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	added := 0
	for _, tagName := range req.Tags {
		tag, err := s.tagStore.FindOrCreate(tagName)
		if err != nil {
			continue
		}
		if err := s.tagStore.AddToFile(fileID, tag.ID, userID); err != nil {
			continue
		}
		added++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"added": added,
	})
}

func (s *Server) handleRemoveFileTag(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := s.tagStore.RemoveFromFile(fileID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove tag")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleGetFileAlbums(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := s.checkFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	rows, err := s.db.Query(
		`SELECT a.id, a.name, a.user_id, a.user_id = ? as is_owner
		 FROM albums a
		 JOIN album_items ai ON a.id = ai.album_id
		 WHERE ai.file_id = ?
		 AND (a.user_id = ? OR EXISTS (
			SELECT 1 FROM album_shares s
			WHERE s.album_id = a.id AND s.shared_with_user_id = ?
		 ))
		 ORDER BY a.created_at DESC`,
		userID, fileID, userID, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get file albums")
		return
	}
	defer rows.Close()

	type albumInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
		IsOwner bool   `json:"is_owner"`
	}

	albums := []albumInfo{}
	for rows.Next() {
		a := albumInfo{}
		if err := rows.Scan(&a.ID, &a.Name, &a.OwnerID, &a.IsOwner); err != nil {
			continue
		}
		albums = append(albums, a)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"albums": albums,
	})
}
