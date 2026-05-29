package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
)

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}

	opts := store.FileListOptions{
		UserID:    userID,
		Path:      r.URL.Query().Get("path"),
		FolderID:  folderIDFromQuery(r),
		Cursor:    r.URL.Query().Get("cursor"),
		Limit:     limit,
		Sort:      r.URL.Query().Get("sort"),
		Order:     r.URL.Query().Get("order"),
		MediaType: r.URL.Query().Get("media_type"),
		DateFrom:  r.URL.Query().Get("date_from"),
		DateTo:    r.URL.Query().Get("date_to"),
		Camera:    r.URL.Query().Get("camera"),
	}

	if opts.Sort == "" {
		opts.Sort = "taken_at"
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	files, nextCursor, total, err := s.fileStore.List(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list files")
		return
	}

	items := make([]interface{}, 0, len(files))
	for _, f := range files {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":      items,
		"nextCursor": nextCursor,
		"total":      total,
	})
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	exif, _ := s.exifStore.FindByFileID(fileID)

	item := fileResponse{
		ID:           file.ID,
		UserID:       &file.UserID,
		Filename:     file.Filename,
		OriginalName: file.OriginalName,
		Path:         file.Path,
		SizeBytes:    file.SizeBytes,
		MimeType:     file.MimeType,
		SHA256:       &file.SHA256,
		MediaType:    string(file.MediaType),
		Width:        file.Width,
		Height:       file.Height,
		DurationSec:  file.DurationSec,
		TakenAt:      file.TakenAt,
		FolderID:     file.FolderID,
		CreatedAt:    file.CreatedAt.Format(timeRFC3339),
		UpdatedAt:    file.UpdatedAt.Format(timeRFC3339),
		Thumbnails:   buildThumbnailSet(file.ID, file.MediaType),
		EXIF:         exif,
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleSoftDeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := s.fileStore.SoftDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchSoftDelete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := s.fileStore.BatchSoftDelete(userID, req.IDs); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchMove(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs      []string `json:"ids"`
		FolderID *string  `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := s.fileStore.BatchMove(userID, req.IDs, req.FolderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchCopy(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs      []string `json:"ids"`
		FolderID *string  `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	copies, err := s.fileStore.BatchCopy(userID, req.IDs, req.FolderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to copy files")
		return
	}

	copyIDs := make([]string, len(copies))
	for i, c := range copies {
		copyIDs[i] = c.ID
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(copies),
		"ids":   copyIDs,
	})
}

func (s *Server) handlePermanentDeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	filePath := filepath.Join(s.cfg.OriginalsDir(), userID, file.Filename)
	os.Remove(filePath)

	if err := s.fileStore.PermanentDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleServeThumbnail(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fileID")
	size := r.PathValue("size")
	thumbPath := filepath.Join(s.cfg.ThumbnailsDir(), fileID, size)

	if _, err := os.Stat(thumbPath); err == nil {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, thumbPath)
		return
	}

	if s.cfg.Storage.S3.Enabled && s.storageService != nil {
		s3Key := fmt.Sprintf("thumbnails/%s/%s", fileID, size)
		if err := s.storageService.GetObject(s3Key, thumbPath); err != nil {
			slog.Warn("s3 thumbnail fallback failed", "key", s3Key, "error", err)
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Thumbnail not found")
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, thumbPath)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "Thumbnail not found")
}

func (s *Server) handleListDirs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	root, err := s.fileStore.ListDirs(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list directories")
		return
	}

	writeJSON(w, http.StatusOK, root)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	query := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	if query == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"items": []interface{}{},
			"total": 0,
		})
		return
	}

	result, err := s.fileStore.Search(userID, query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Search failed")
		return
	}

	items := make([]interface{}, 0, len(result.Files))
	for _, f := range result.Files {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"total": result.Total,
	})
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	granularity := r.URL.Query().Get("granularity")

	groups, err := s.fileStore.Timeline(userID, granularity)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get timeline")
		return
	}
	if groups == nil {
		groups = []store.TimelineGroup{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"groups": groups,
	})
}

func (s *Server) handleGeoPoints(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	latMin, _ := strconv.ParseFloat(r.URL.Query().Get("lat_min"), 64)
	latMax, _ := strconv.ParseFloat(r.URL.Query().Get("lat_max"), 64)
	lonMin, _ := strconv.ParseFloat(r.URL.Query().Get("lon_min"), 64)
	lonMax, _ := strconv.ParseFloat(r.URL.Query().Get("lon_max"), 64)

	points, err := s.geoStore.GetPoints(userID, store.GeoBounds{
		LatMin: latMin,
		LatMax: latMax,
		LonMin: lonMin,
		LonMax: lonMax,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get geo points")
		return
	}

	if points == nil {
		points = []store.GeoPoint{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"points": points,
		"total":  len(points),
	})
}

func (s *Server) handleGeoClusters(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	latMin, _ := strconv.ParseFloat(r.URL.Query().Get("lat_min"), 64)
	latMax, _ := strconv.ParseFloat(r.URL.Query().Get("lat_max"), 64)
	lonMin, _ := strconv.ParseFloat(r.URL.Query().Get("lon_min"), 64)
	lonMax, _ := strconv.ParseFloat(r.URL.Query().Get("lon_max"), 64)
	zoom, _ := strconv.Atoi(r.URL.Query().Get("zoom"))

	points, err := s.geoStore.GetPoints(userID, store.GeoBounds{
		LatMin: latMin,
		LatMax: latMax,
		LonMin: lonMin,
		LonMax: lonMax,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get geo points")
		return
	}

	gridSize := gridSizeForZoom(zoom)
	type clusterKey struct {
		latBlock int
		lonBlock int
	}
	clusters := make(map[clusterKey]*struct {
		latSum  float64
		lonSum  float64
		count   int
		fileID  string
	})

	for _, p := range points {
		k := clusterKey{
			latBlock: int(p.Latitude / gridSize),
			lonBlock: int(p.Longitude / gridSize),
		}
		if c, ok := clusters[k]; ok {
			c.latSum += p.Latitude
			c.lonSum += p.Longitude
			c.count++
		} else {
			clusters[k] = &struct {
				latSum  float64
				lonSum  float64
				count   int
				fileID  string
			}{p.Latitude, p.Longitude, 1, p.FileID}
		}
	}

	result := make([]interface{}, 0, len(clusters))
	for _, c := range clusters {
		result = append(result, map[string]interface{}{
			"latitude":     c.latSum / float64(c.count),
			"longitude":    c.lonSum / float64(c.count),
			"count":        c.count,
			"thumbnailUrl": fmt.Sprintf("/api/v1/thumb/%s/sm.jpg", c.fileID),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": result,
	})
}

func gridSizeForZoom(zoom int) float64 {
	switch {
	case zoom >= 12:
		return 0.001
	case zoom >= 10:
		return 0.01
	case zoom >= 8:
		return 0.1
	case zoom >= 5:
		return 1.0
	default:
		return 5.0
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	stats, err := s.fileStore.Stats(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get stats")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_files":       stats.TotalFiles,
		"total_photos":      stats.TotalPhotos,
		"total_videos":      stats.TotalVideos,
		"total_size_bytes":  stats.TotalSize,
		"photos_with_gps":   stats.PhotosWithGPS,
		"date_range": map[string]interface{}{
			"oldest": stats.DateOldest,
			"newest": stats.DateNewest,
		},
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	filePath := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleBatchDownload(w http.ResponseWriter, r *http.Request) {
	s.handleBatchDownloadReal(w, r)
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	userResponses := make([]userResponse, 0, len(users))
	for _, u := range users {
		userResponses = append(userResponses, userResponse{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Role:        string(u.Role),
			CreatedAt:   u.CreatedAt.Format(timeRFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": userResponses,
		"total": len(users),
	})
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if err := s.userStore.Delete(userID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Role != "admin" && req.Role != "member" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Role must be 'admin' or 'member'")
		return
	}

	if err := s.userStore.UpdateRole(userID, model.UserRole(req.Role)); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update role")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

type fileResponse struct {
	ID           string                `json:"id"`
	UserID       *string               `json:"user_id,omitempty"`
	Filename     string                `json:"filename"`
	OriginalName string                `json:"originalName"`
	Path         string                `json:"path"`
	SizeBytes    int64                 `json:"sizeBytes"`
	MimeType     string                `json:"mimeType"`
	SHA256       *string               `json:"sha256,omitempty"`
	MediaType    string                `json:"mediaType"`
	Width        *int                  `json:"width,omitempty"`
	Height       *int                  `json:"height,omitempty"`
	DurationSec  *float64              `json:"durationSec,omitempty"`
	TakenAt      *string               `json:"takenAt,omitempty"`
	FolderID     *string               `json:"folder_id,omitempty"`
	CreatedAt    string                `json:"createdAt"`
	UpdatedAt    string                `json:"updatedAt,omitempty"`
	Thumbnails   *thumbnailSetResponse `json:"thumbnails,omitempty"`
	EXIF         interface{}           `json:"exif,omitempty"`
}

type thumbnailSetResponse struct {
	SM         *thumbnailInfoResponse `json:"sm"`
	LG         *thumbnailInfoResponse `json:"lg"`
	MD         *thumbnailInfoResponse `json:"md"`
	Preview    *thumbnailInfoResponse `json:"preview"`
	VideoStill *thumbnailInfoResponse `json:"videoStill,omitempty"`
}

type thumbnailInfoResponse struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func buildThumbnailSet(fileID string, mediaType model.MediaType) *thumbnailSetResponse {
	if mediaType == model.MediaTypeFile {
		return nil
	}
	if mediaType == model.MediaTypeVideo {
		return &thumbnailSetResponse{
			VideoStill: &thumbnailInfoResponse{
				URL:    fmt.Sprintf("/api/v1/thumb/%s/video_still.jpg", fileID),
				Width:  600,
				Height: 338,
			},
		}
	}
	return &thumbnailSetResponse{
		SM: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/sm.jpg", fileID),
			Width:  60,
			Height: 60,
		},
		LG: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/lg.jpg", fileID),
			Width:  300,
			Height: 300,
		},
		MD: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/md.jpg", fileID),
			Width:  600,
			Height: 600,
		},
		Preview: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/preview.webp", fileID),
			Width:  1280,
			Height: 720,
		},
	}
}

var timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func folderIDFromQuery(r *http.Request) *string {
	v := r.URL.Query().Get("folder_id")
	if v == "" {
		return nil
	}
	s := new(string)
	if v == "root" {
		*s = ""
	} else {
		*s = v
	}
	return s
}
