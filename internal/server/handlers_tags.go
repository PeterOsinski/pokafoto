package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (c *FileCtl) HandleListTags(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	tags, err := c.TagStore.Search(q)
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

func (c *FileCtl) HandleTagStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	tags, err := c.TagStore.ListWithCount(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get tag stats")
		return
	}

	type tagStatResponse struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	items := make([]tagStatResponse, 0, len(tags))
	for _, t := range tags {
		items = append(items, tagStatResponse{ID: t.ID, Name: t.Name, Count: t.Count})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tags": items,
	})
}

func (c *FileCtl) HandleGetFileTags(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	tags, err := c.TagStore.FindByFileID(fileID)
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

func (c *FileCtl) HandleAddFileTags(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
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
		tag, err := c.TagStore.FindOrCreate(tagName)
		if err != nil {
			continue
		}
		if err := c.TagStore.AddToFile(fileID, tag.ID, userID); err != nil {
			continue
		}
		added++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"added": added,
	})
}

func (c *FileCtl) HandleRemoveFileTag(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := c.TagStore.RemoveFromFile(fileID, tagID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove tag")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (c *FileCtl) HandleGetFileAlbums(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "id")
	userID := getUserID(r)

	hasAccess := c.CheckFileAccess(fileID, userID)
	if !hasAccess {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	albumInfos, err := c.AlbumItemStore.ListAlbumsByFile(fileID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get file albums")
		return
	}

	type albumInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
		IsOwner bool   `json:"is_owner"`
	}

	albums := make([]albumInfo, 0, len(albumInfos))
	for _, a := range albumInfos {
		albums = append(albums, albumInfo{
			ID:      a.ID,
			Name:    a.Name,
			OwnerID: a.OwnerID,
			IsOwner: a.IsOwner,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"albums": albums,
	})
}
