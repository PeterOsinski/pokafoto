package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"golang.org/x/sys/unix"
)

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 500
	}

	opts := store.FileListOptions{
		UserID:     userID,
		Path:       r.URL.Query().Get("path"),
		FolderID:   folderIDFromQuery(r),
		AllFolders: r.URL.Query().Get("all_folders") == "true",
		Cursor:     r.URL.Query().Get("cursor"),
		Limit:      limit,
		Sort:       r.URL.Query().Get("sort"),
		Order:      r.URL.Query().Get("order"),
		MediaType:  r.URL.Query().Get("media_type"),
		DateFrom:   r.URL.Query().Get("date_from"),
		DateTo:     r.URL.Query().Get("date_to"),
		Camera:     r.URL.Query().Get("camera"),
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
			s.fallbackThumbnail(w, r, fileID, size)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, thumbPath)
		return
	}

	s.fallbackThumbnail(w, r, fileID, size)
}

func (s *Server) fallbackThumbnail(w http.ResponseWriter, r *http.Request, fileID, size string) {
	fallbackMap := map[string]string{
		"preview.webp":  "md.jpg",
		"lg.jpg":        "md.jpg",
		"md.jpg":        "sm.jpg",
	}
	if fallback, ok := fallbackMap[size]; ok {
		fallbackPath := filepath.Join(s.cfg.ThumbnailsDir(), fileID, fallback)
		if _, err := os.Stat(fallbackPath); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			http.ServeFile(w, r, fallbackPath)
			return
		}
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "Thumbnail not found")
}

func (s *Server) handleListDirs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	allFolders := r.URL.Query().Get("all_folders") == "true"

	root, err := s.fileStore.ListDirs(userID, allFolders)
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
	if _, err := os.Stat(filePath); err == nil {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		http.ServeFile(w, r, filePath)
		return
	}

	if s.cfg.Storage.S3.Enabled && s.storageService != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		stream, err := s.storageService.GetObjectStream(s3Key)
		if err != nil {
			slog.Warn("s3 download stream failed", "file_id", fileID, "error", err)
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk or in S3")
			return
		}
		defer stream.Close()
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		io.Copy(w, stream)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk")
}

func (s *Server) handleBatchDownload(w http.ResponseWriter, r *http.Request) {
	s.handleBatchDownloadReal(w, r)
}

func (s *Server) handleListTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	opts := store.FileListOptions{
		UserID: userID,
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  limit,
		Order:  order,
	}

	files, nextCursor, total, err := s.fileStore.ListTrash(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list trash")
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
			UpdatedAt:    f.UpdatedAt.Format(timeRFC3339),
			Thumbnails:   buildThumbnailSet(f.ID, f.MediaType),
			DeletedAt:    formatDeletedAt(f.DeletedAt),
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":      items,
		"nextCursor": nextCursor,
		"total":      total,
	})
}

func (s *Server) handleTrashStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	stats, err := s.fileStore.TrashStats(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get trash stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      stats.Count,
		"size_bytes": stats.SizeBytes,
	})
}

func (s *Server) handleRestoreTrash(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID || !file.IsDeleted {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found in trash")
		return
	}

	if err := s.fileStore.Restore(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchRestoreTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := s.fileStore.BatchRestore(userID, req.IDs); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePermanentDeleteTrash(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID || !file.IsDeleted {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found in trash")
		return
	}

	originalPath := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
	os.Remove(originalPath)
	thumbDir := filepath.Join(s.cfg.ThumbnailsDir(), file.ID)
	os.RemoveAll(thumbDir)

	s.enqueueS3Deletion(file.ID, file.UserID, file.Filename)

	if err := s.fileStore.PermanentDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBatchPermanentDeleteTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	placeholders := make([]string, len(req.IDs))
	args := make([]interface{}, 0, len(req.IDs)+1)
	args = append(args, userID)
	for i, id := range req.IDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(`SELECT id, user_id, filename FROM files WHERE user_id = ? AND is_deleted = 1 AND id IN (%s)`, strings.Join(placeholders, ", "))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query files")
		return
	}
	defer rows.Close()

	type deleteRec struct {
		id   string
		uid  string
		fn   string
	}
	var records []deleteRec
	for rows.Next() {
		var id, uid, fn string
		if err := rows.Scan(&id, &uid, &fn); err != nil {
			continue
		}
		originalPath := filepath.Join(s.cfg.OriginalsDir(), uid, fn)
		os.Remove(originalPath)
		thumbDir := filepath.Join(s.cfg.ThumbnailsDir(), id)
		os.RemoveAll(thumbDir)
		records = append(records, deleteRec{id, uid, fn})
	}
	rows.Close()

	var deleteIDs []string
	for _, r := range records {
		s.enqueueS3Deletion(r.id, r.uid, r.fn)
		deleteIDs = append(deleteIDs, r.id)
	}

	if len(deleteIDs) > 0 {
		if err := s.fileStore.BatchPermanentDelete(userID, deleteIDs); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete files")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEmptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := s.db.Query(`SELECT id, user_id, filename FROM files WHERE user_id = ? AND is_deleted = 1`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query trash")
		return
	}
	defer rows.Close()

	type emptyRecord struct {
		id  string
		uid string
		fn  string
	}
	var records []emptyRecord
	for rows.Next() {
		var id, uid, fn string
		if err := rows.Scan(&id, &uid, &fn); err != nil {
			continue
		}
		originalPath := filepath.Join(s.cfg.OriginalsDir(), uid, fn)
		os.Remove(originalPath)
		thumbDir := filepath.Join(s.cfg.ThumbnailsDir(), id)
		os.RemoveAll(thumbDir)
		records = append(records, emptyRecord{id, uid, fn})
	}
	rows.Close()

	var allIDs []string
	for _, r := range records {
		s.enqueueS3Deletion(r.id, r.uid, r.fn)
		allIDs = append(allIDs, r.id)
	}

	for i := 0; i < len(allIDs); i += 100 {
		end := i + 100
		if end > len(allIDs) {
			end = len(allIDs)
		}
		if err := s.fileStore.BatchPermanentDelete(userID, allIDs[i:end]); err != nil {
			slog.Warn("empty trash chunk failed", "error", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminS3DeletionQueue(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pending": s.s3DeletionPool.PendingCount(),
	})
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string  `json:"username"`
		Password    string  `json:"password"`
		Role        string  `json:"role"`
		DisplayName *string `json:"display_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 32 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username must be 3-32 characters")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}
	if req.Role != "admin" && req.Role != "member" && req.Role != "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Role must be 'admin' or 'member'")
		return
	}

	role := model.RoleMember
	if req.Role == "admin" {
		role = model.RoleAdmin
	}

	existing, _ := s.userStore.FindByUsername(req.Username)
	if existing != nil {
		writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username is already taken")
		return
	}

	user, err := s.userStore.Create(req.Username, req.Password, role, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        string(user.Role),
		CreatedAt:   user.CreatedAt.Format(timeRFC3339),
	})
}

func (s *Server) handleAdminGetRegistration(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow_registration": s.isRegistrationAllowed(),
	})
}

func (s *Server) handleAdminToggleRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	val := "false"
	if req.Enabled {
		val = "true"
	}
	if err := s.settingStore.Set("allow_registration", val); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update setting")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow_registration": req.Enabled,
	})
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	userResponses := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		fileCount, _ := s.fileStore.Stats(u.ID)
		thumbSize, _ := s.userStore.GetThumbnailSize(u.ID)

		resp := map[string]interface{}{
			"id":          u.ID,
			"username":    u.Username,
			"display_name": u.DisplayName,
			"role":        string(u.Role),
			"created_at":  u.CreatedAt.Format(timeRFC3339),
		}
		if u.SpaceQuota != nil {
			resp["space_quota"] = *u.SpaceQuota
		} else {
			resp["space_quota"] = nil
		}
		if fileCount != nil {
			resp["file_count"] = fileCount.TotalFiles
			resp["total_size_bytes"] = fileCount.TotalSize
		} else {
			resp["file_count"] = 0
			resp["total_size_bytes"] = 0
		}
		resp["thumbnail_size_bytes"] = thumbSize
		userResponses = append(userResponses, resp)
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

func (s *Server) handleAdminUpdateQuota(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	var req struct {
		SpaceQuota *int64 `json:"space_quota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.SpaceQuota != nil && *req.SpaceQuota < 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Quota must be non-negative")
		return
	}

	if req.SpaceQuota != nil {
		used, err := s.userStore.GetUsedSpace(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check usage")
			return
		}
		if *req.SpaceQuota < used {
			writeError(w, http.StatusUnprocessableEntity, "QUOTA_BELOW_USAGE", "Quota cannot be below current usage")
			return
		}
	}

	if err := s.userStore.UpdateSpaceQuota(userID, req.SpaceQuota); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update quota")
		return
	}

	user, _ := s.userStore.FindByID(userID)
	if user == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
		return
	}

	resp := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     string(user.Role),
	}
	if user.SpaceQuota != nil {
		resp["space_quota"] = *user.SpaceQuota
	} else {
		resp["space_quota"] = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminFileBreakdown(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	var breakdown *store.AdminFileBreakdown
	var err error
	if userID != "" {
		breakdown, err = s.fileStore.AdminFileBreakdownByUser(userID)
	} else {
		breakdown, err = s.fileStore.AdminFileBreakdown()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get file breakdown")
		return
	}

	writeJSON(w, http.StatusOK, breakdown)
}

func (s *Server) handleAdminListJobs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	statusFilter := r.URL.Query().Get("status")

	jobs, total, err := s.uploadJobStore.ListAll(limit, offset, statusFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list jobs")
		return
	}

	summary, _ := s.uploadJobStore.CountByStatus()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":    jobs,
		"total":   total,
		"summary": summary,
	})
}

func (s *Server) handleAdminRetryJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if err := s.uploadJobStore.Requeue(jobID); err != nil {
		writeError(w, http.StatusConflict, "RETRY_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (s *Server) handleAdminReconcileJobs(w http.ResponseWriter, r *http.Request) {
	result := s.workerPool.RunReconciliation()
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAdminWorkers(w http.ResponseWriter, r *http.Request) {
	stats := s.workerPool.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAdminThumbnailStats(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	var breakdown []store.ThumbnailBreakdown
	var err error
	if userID != "" {
		breakdown, err = s.thumbnailStore.BreakdownByUser(userID)
	} else {
		breakdown, err = s.thumbnailStore.Breakdown()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get thumbnail stats")
		return
	}
	if breakdown == nil {
		breakdown = []store.ThumbnailBreakdown{}
	}

	var totalCount int64
	var totalSizeBytes int64
	for _, b := range breakdown {
		totalCount += b.Count
		totalSizeBytes += b.TotalSize
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"breakdown":        breakdown,
		"total_count":      totalCount,
		"total_size_bytes": totalSizeBytes,
	})
}

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	totalFiles := int64(0)
	totalSize := int64(0)
	userStats := make([]map[string]interface{}, 0, len(users))

	for _, u := range users {
		stats, err := s.fileStore.Stats(u.ID)
		if err != nil {
			continue
		}
		thumbSize, _ := s.userStore.GetThumbnailSize(u.ID)
		totalFiles += stats.TotalFiles
		totalSize += stats.TotalSize
		ustat := map[string]interface{}{
			"id":                  u.ID,
			"username":            u.Username,
			"role":                string(u.Role),
			"file_count":          stats.TotalFiles,
			"total_size_bytes":    stats.TotalSize,
			"thumbnail_size_bytes": thumbSize,
		}
		if u.SpaceQuota != nil {
			ustat["space_quota"] = *u.SpaceQuota
		} else {
			ustat["space_quota"] = nil
		}
		userStats = append(userStats, ustat)
	}

	cacheSize, _ := s.thumbnailStore.TotalSize()
	diskTotal, diskFree, diskUsed := diskUsage(s.cfg.Storage.Local.Path)
	diskPct := float64(0)
	if diskTotal > 0 {
		diskPct = float64(diskUsed) / float64(diskTotal) * 100
	}

	var originalsSize int64
	filepath.Walk(s.cfg.OriginalsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			originalsSize += info.Size()
		}
		return nil
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_files":          totalFiles,
		"total_size_bytes":     totalSize,
		"cache_size_bytes":     cacheSize,
		"disk_total_bytes":     diskTotal,
		"disk_free_bytes":      diskFree,
		"disk_used_bytes":      diskUsed,
		"disk_utilization_pct": diskPct,
		"max_disk_usage_pct":   s.cfg.MaxDiskUsagePercent(),
		"originals_size_bytes": originalsSize,
		"users":                userStats,
	})
}

func diskUsage(path string) (total, free, used uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	used = total - free
	return
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
	DeletedAt    *string               `json:"deletedAt,omitempty"`
	Thumbnails   *thumbnailSetResponse `json:"thumbnails,omitempty"`
	EXIF         interface{}           `json:"exif,omitempty"`
}

type thumbnailSetResponse struct {
	SM         *thumbnailInfoResponse `json:"sm"`
	LG         *thumbnailInfoResponse `json:"lg"`
	MD         *thumbnailInfoResponse `json:"md"`
	XL         *thumbnailInfoResponse `json:"xl,omitempty"`
	Preview    *thumbnailInfoResponse `json:"preview"`
	VideoStill *thumbnailInfoResponse `json:"videoStill,omitempty"`
}

type thumbnailInfoResponse struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func buildThumbnailSet(fileID string, mediaType model.MediaType) *thumbnailSetResponse {
	ver := fileID
	if len(ver) > 8 {
		ver = ver[:8]
	}
	if mediaType == model.MediaTypeFile {
		return nil
	}
	if mediaType == model.MediaTypeVideo {
		return &thumbnailSetResponse{
			VideoStill: &thumbnailInfoResponse{
				URL:    fmt.Sprintf("/api/v1/thumb/%s/video_still.jpg?v=%s", fileID, ver),
				Width:  600,
				Height: 338,
			},
		}
	}
	return &thumbnailSetResponse{
		SM: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/sm.jpg?v=%s", fileID, ver),
			Width:  60,
			Height: 60,
		},
		LG: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/lg.jpg?v=%s", fileID, ver),
			Width:  300,
			Height: 300,
		},
		MD: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/md.jpg?v=%s", fileID, ver),
			Width:  600,
			Height: 600,
		},
		XL: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/xl.jpg?v=%s", fileID, ver),
			Width:  2000,
			Height: 2000,
		},
		Preview: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/preview.webp?v=%s", fileID, ver),
			Width:  1280,
			Height: 720,
		},
	}
}

var timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (s *Server) handleAdminListEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	events, total, err := s.systemEventsStore.List(
		limit, offset,
		r.URL.Query().Get("event_type"),
		r.URL.Query().Get("severity"),
		r.URL.Query().Get("date_from"),
		r.URL.Query().Get("date_to"),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
	})
}

func (s *Server) handleAdminEventCounts(w http.ResponseWriter, r *http.Request) {
	counts, err := s.systemEventsStore.EventCounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get event counts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"by_type": counts,
	})
}

func (s *Server) handleAdminBackupStatus(w http.ResponseWriter, r *http.Request) {
	result := s.backupScheduler.LastResult()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":        s.cfg.Backup.Enabled,
		"interval_h":     s.cfg.Backup.IntervalH,
		"retention_days": s.cfg.Backup.RetentionDays,
		"last_result":    result,
	})
}

func (s *Server) handleAdminTriggerBackup(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Backup.Enabled || !s.storageService.IsConnected() {
		writeError(w, http.StatusConflict, "BACKUP_UNAVAILABLE", "Backup is not enabled or S3 is not connected")
		return
	}
	go s.backupScheduler.RunBackup()
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status": "backup_started",
	})
}

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

func formatDeletedAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeRFC3339)
	return &s
}
