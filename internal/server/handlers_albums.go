package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/drive/drive/internal/model"
	"github.com/go-chi/chi/v5"
)

func (c *AlbumCtl) HandleListAlbums(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	ownAlbums, err := c.AlbumStore.ListByUser(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list albums")
		return
	}

	sharedAlbums, err := c.AlbumStore.ListSharedWithUser(userID)
	if err != nil {
		sharedAlbums = nil
	}

	type albumResponse struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description *string `json:"description,omitempty"`
		ItemCount   int64   `json:"item_count"`
		OwnerID     string  `json:"owner_id"`
		OwnerName   string  `json:"owner_name"`
		IsShared    bool    `json:"is_shared"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}

	myAlbums := make([]albumResponse, 0, len(ownAlbums))
	for _, a := range ownAlbums {
		ar := albumResponse{
			ID:          a.ID,
			Name:        a.Name,
			Description: a.Description,
			ItemCount:   c.AlbumStore.ItemCount(a.ID),
			OwnerID:     a.UserID,
			IsShared:    c.AlbumStore.HasShares(a.ID),
			CreatedAt:   a.CreatedAt.Format(timeRFC3339),
			UpdatedAt:   a.UpdatedAt.Format(timeRFC3339),
		}
		myAlbums = append(myAlbums, ar)
	}

	shared := make([]albumResponse, 0, len(sharedAlbums))
	for _, sa := range sharedAlbums {
		ar := albumResponse{
			ID:          sa.Album.ID,
			Name:        sa.Album.Name,
			Description: sa.Album.Description,
			ItemCount:   sa.ItemCount,
			OwnerID:     sa.Album.UserID,
			OwnerName:   sa.OwnerName,
			IsShared:    true,
			CreatedAt:   sa.Album.CreatedAt.Format(timeRFC3339),
			UpdatedAt:   sa.Album.UpdatedAt.Format(timeRFC3339),
		}
		shared = append(shared, ar)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"myAlbums":     myAlbums,
		"sharedAlbums": shared,
	})
}

func (c *AlbumCtl) HandleCreateAlbum(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Album name is required")
		return
	}

	album, err := c.AlbumStore.Create(userID, req.Name, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create album")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          album.ID,
		"name":        album.Name,
		"description": album.Description,
		"created_at":  album.CreatedAt.Format(timeRFC3339),
	})
}

func (c *AlbumCtl) HandleGetAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	album, err := c.AlbumStore.FindByID(albumID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	_, found, err := c.AlbumStore.CheckAccess(albumID, userID)
	if err != nil || !found {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	awd, err := c.AlbumStore.FindByIDWithOwner(albumID)
	if err != nil {
		awd = &model.AlbumWithDetails{Album: album}
	}

	shares, _ := c.AlbumStore.ListShares(albumID)
	if shares == nil {
		shares = []model.SharedUser{}
	}

	isOwner := album.UserID == userID
	perm := ""
	if !isOwner {
		p, _, _ := c.AlbumStore.CheckAccess(albumID, userID)
		perm = p
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":               album.ID,
		"name":             album.Name,
		"description":      album.Description,
		"owner_id":         album.UserID,
		"owner_name":       awd.OwnerName,
		"item_count":       awd.ItemCount,
		"is_owner":         isOwner,
		"share_permission": perm,
		"shared_users":     shares,
		"created_at":       album.CreatedAt.Format(timeRFC3339),
		"updated_at":       album.UpdatedAt.Format(timeRFC3339),
	})
}

func (c *AlbumCtl) HandleUpdateAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	album, err := c.AlbumStore.FindByID(albumID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if album.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the album owner can update it")
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Album name is required")
		return
	}

	if err := c.AlbumStore.Update(albumID, req.Name, req.Description); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update album")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (c *AlbumCtl) HandleDeleteAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	album, err := c.AlbumStore.FindByID(albumID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if album.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the album owner can delete it")
		return
	}

	if err := c.AlbumStore.Delete(albumID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete album")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (c *AlbumCtl) HandleListAlbumItems(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	_, found, err := c.AlbumStore.CheckAccess(albumID, userID)
	if err != nil || !found {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}

	fileIDs, _, err := c.AlbumItemStore.ListFileIDs(albumID, limit+1, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list album items")
		return
	}

	var items []interface{}
	for _, fileID := range fileIDs {
		f, err := c.FileStore.FindByID(fileID)
		if err != nil || f == nil || f.IsDeleted {
			continue
		}
		item := fileResponse{
			ID:           f.ID,
			Filename:     f.Filename,
			OriginalName: f.OriginalName,
			Path:         f.Path,
			SizeBytes:    f.SizeBytes,
			MimeType:     f.MimeType,
			MediaType:    string(f.MediaType),
			Width:        f.Width,
			Height:       f.Height,
			DurationSec:  f.DurationSec,
			TakenAt:      f.TakenAt,
			FolderID:     f.FolderID,
			CreatedAt:    f.CreatedAt.Format(timeRFC3339),
			Thumbnails:   buildThumbnailSet(f.ID, f.MediaType),
		}
		items = append(items, item)
	}

	if items == nil {
		items = []interface{}{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (c *AlbumCtl) HandleAddAlbumItems(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	perm, found, err := c.AlbumStore.CheckAccess(albumID, userID)
	if err != nil || !found {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if perm != "edit" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "You don't have permission to add items to this album")
		return
	}

	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	added := 0
	for _, fileID := range req.FileIDs {
		_, err := c.AlbumItemStore.Add(albumID, fileID, userID)
		if err == nil {
			added++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"added": added,
	})
}

func (c *AlbumCtl) HandleRemoveAlbumItem(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	itemID := chi.URLParam(r, "itemId")
	userID := getUserID(r)

	perm, found, err := c.AlbumStore.CheckAccess(albumID, userID)
	if err != nil || !found {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if perm != "edit" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "You don't have permission to remove items from this album")
		return
	}

	if err := c.AlbumItemStore.RemoveByID(itemID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove item")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (c *AlbumCtl) HandleShareAlbum(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	userID := getUserID(r)

	album, err := c.AlbumStore.FindByID(albumID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if album.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the album owner can manage shares")
		return
	}

	var req struct {
		Username   string `json:"username"`
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Permission == "" {
		req.Permission = "view"
	}
	if req.Permission != "view" && req.Permission != "comment" && req.Permission != "edit" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid permission. Must be view, comment, or edit")
		return
	}

	targetUser, err := c.UserStore.FindByUsername(req.Username)
	if err != nil || targetUser == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	if targetUser.ID == userID {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Cannot share with yourself")
		return
	}

	share, err := c.AlbumShareStore.Add(albumID, targetUser.ID, req.Permission)
	if err != nil {
		writeError(w, http.StatusConflict, "CONFLICT", "Already shared with this user")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":                  share.ID,
		"shared_with_user_id": share.SharedWithUserID,
		"username":            targetUser.Username,
		"permission":          share.Permission,
	})
}

func (c *AlbumCtl) HandleRemoveShare(w http.ResponseWriter, r *http.Request) {
	albumID := chi.URLParam(r, "id")
	shareID := chi.URLParam(r, "shareId")
	userID := getUserID(r)

	album, err := c.AlbumStore.FindByID(albumID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Album not found")
		return
	}

	if album.UserID != userID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Only the album owner can manage shares")
		return
	}

	if err := c.AlbumShareStore.Remove(shareID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove share")
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
