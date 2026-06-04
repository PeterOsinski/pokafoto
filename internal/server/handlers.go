package server

import (
	"context"
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

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/minio/minio-go/v7"
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

	if opts.FolderID != nil {
		if !s.checkFolderAccess(*opts.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
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
			IsAppManaged: f.IsAppManaged,
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
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !s.checkFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !s.checkFolderAccess(*file.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
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
		DownloadInfo: &downloadInfoResponse{
			SizeBytes:     file.SizeBytes,
			MimeType:      file.MimeType,
			SupportsRange: true,
		},
		IsAppManaged: file.IsAppManaged,
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

	if err := s.checkThumbnailAccess(w, r, fileID); err != nil {
		return
	}

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

func (s *Server) checkThumbnailAccess(w http.ResponseWriter, r *http.Request, fileID string) error {
	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil {
		return nil
	}

	if file.FolderID == nil {
		return nil
	}

	fp, err := s.folderPasswordStore.FindByFolderID(*file.FolderID)
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		s.folderPasswordStore.DeleteByFolderID(*file.FolderID)
		return nil
	}

	unlockToken := r.Header.Get("X-Folder-Unlock-Token")
	if unlockToken != "" {
		unlockedFolderID, ok := s.parseFolderUnlockToken(unlockToken)
		if ok && unlockedFolderID == *file.FolderID {
			return nil
		}
	}

	writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock to view thumbnails")
	return fmt.Errorf("folder password required")
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
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	opts := store.SearchOptions{
		UserID: userID,
		Query:  r.URL.Query().Get("q"),
		Limit:  limit,
		Cursor: r.URL.Query().Get("cursor"),
	}

	if v := r.URL.Query().Get("size_min"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		opts.SizeMin = &n
	}
	if v := r.URL.Query().Get("size_max"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		opts.SizeMax = &n
	}
	if v := r.URL.Query().Get("created_after"); v != "" {
		opts.CreatedAfter = &v
	}
	if v := r.URL.Query().Get("created_before"); v != "" {
		opts.CreatedBefore = &v
	}
	if v := r.URL.Query().Get("taken_after"); v != "" {
		opts.TakenAfter = &v
	}
	if v := r.URL.Query().Get("taken_before"); v != "" {
		opts.TakenBefore = &v
	}
	if tags := r.URL.Query().Get("tags"); tags != "" {
		opts.Tags = strings.Split(tags, ",")
	}

	result, folderPaths, err := s.fileStore.SearchEnhanced(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Search failed")
		return
	}

	type searchItemResponse struct {
		ID           string                `json:"id"`
		Filename     string                `json:"filename"`
		OriginalName string                `json:"originalName"`
		Path         string                `json:"path"`
		SizeBytes    int64                 `json:"sizeBytes"`
		MimeType     string                `json:"mimeType"`
		MediaType    string                `json:"mediaType"`
		Width        *int                  `json:"width,omitempty"`
		Height       *int                  `json:"height,omitempty"`
		DurationSec  *float64              `json:"durationSec,omitempty"`
		TakenAt      *string               `json:"takenAt,omitempty"`
		FolderID     *string               `json:"folder_id,omitempty"`
		FolderPath   string                `json:"folder_path,omitempty"`
		CreatedAt    string                `json:"createdAt"`
		Thumbnails   *thumbnailSetResponse `json:"thumbnails,omitempty"`
	}

	items := make([]searchItemResponse, 0, len(result.Files))
	for _, f := range result.Files {
		item := searchItemResponse{
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
		if fp, ok := folderPaths[f.ID]; ok {
			item.FolderPath = fp
		}
		items = append(items, item)
	}

	if items == nil {
		items = []searchItemResponse{}
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
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !s.checkFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !s.checkFolderAccess(*file.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
	}

	if file.IsAppManaged {
		doc, err := s.docStore.FindByFileID(fileID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Document content not found")
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Length", strconv.Itoa(len(doc.Content)))
		w.Write([]byte(doc.Content))
		return
	}

	filePath := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := os.Stat(filePath); err == nil {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Accept-Ranges", "bytes")
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		if file.SizeBytes > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
		}
		if file.SHA256 != "" {
			w.Header().Set("ETag", fmt.Sprintf(`"%s"`, file.SHA256))
		}
		http.ServeFile(w, r, filePath)
		return
	}

	if s.cfg.Storage.S3.Enabled && s.storageService != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Accept-Ranges", "bytes")
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		if file.SHA256 != "" {
			w.Header().Set("ETag", fmt.Sprintf(`"%s"`, file.SHA256))
		}
		serveFileWithRange(w, r, "", &s3Key, s.cfg, s.storageService, file.MimeType, file.SizeBytes)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk")
}

func (s *Server) handleVideoStreamWithToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if !s.validateTokenAndSetContext(w, r, tokenStr) {
			return
		}
	} else {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			return
		}
		if !s.validateTokenAndSetContext(w, r, token) {
			return
		}
	}
	s.handleVideoStream(w, r)
}

func (s *Server) handleDownloadWithToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if !s.validateTokenAndSetContext(w, r, tokenStr) {
			return
		}
	} else {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			return
		}
		if !s.validateTokenAndSetContext(w, r, token) {
			return
		}
	}
	s.handleDownload(w, r)
}

func (s *Server) handleVideoStream(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := s.fileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !s.checkFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !s.checkFolderAccess(*file.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
	}

	if file.MediaType != model.MediaTypeVideo {
		writeError(w, http.StatusBadRequest, "NOT_VIDEO", "File is not a video")
		return
	}

	quality := r.URL.Query().Get("quality")

	if quality == "proxy" {
		thumb, err := s.thumbnailStore.FindByFileIDAndSize(fileID, model.ThumbSizeVideoProxy)
		if err == nil && thumb != nil {
			serveFileWithRange(w, r, thumb.LocalPath, thumb.S3Key, s.cfg, s.storageService, "video/mp4", thumb.SizeBytes)
			return
		}
	}

	filePath := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := os.Stat(filePath); err == nil {
		serveFileWithRange(w, r, filePath, nil, s.cfg, s.storageService, file.MimeType, file.SizeBytes)
		return
	}

	if s.cfg.Storage.S3.Enabled && s.storageService != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		serveFileWithRange(w, r, "", &s3Key, s.cfg, s.storageService, file.MimeType, file.SizeBytes)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "Video not found on disk or in S3")
}

func parseRange(rangeHeader string, fileSize int64) (int64, int64, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return -1, -1, fmt.Errorf("range does not start with bytes=")
	}
	rangeVal := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeVal, "-", 2)
	if len(parts) != 2 {
		return -1, -1, fmt.Errorf("invalid range format: %s", rangeHeader)
	}

	var start, end int64 = -1, -1
	if parts[0] != "" {
		var n uint64
		n, err := strconv.ParseUint(parts[0], 10, 63)
		if err != nil {
			return -1, -1, fmt.Errorf("invalid range start: %s", parts[0])
		}
		start = int64(n)
	}
	if parts[1] != "" {
		var n uint64
		n, err := strconv.ParseUint(parts[1], 10, 63)
		if err != nil {
			return -1, -1, fmt.Errorf("invalid range end: %s", parts[1])
		}
		end = int64(n)
	}

	if start < 0 && end < 0 {
		return -1, -1, fmt.Errorf("no start or end in range")
	}

	if start < 0 {
		start = fileSize - end
		end = fileSize - 1
	} else if end < 0 {
		end = fileSize - 1
	}

	if start >= fileSize || start > end {
		return -1, -1, fmt.Errorf("range not satisfiable: start=%d end=%d size=%d", start, end, fileSize)
	}

	return start, end, nil
}

func serveFileWithRange(w http.ResponseWriter, r *http.Request, localPath string, s3Key *string, cfg *config.Config, storageService *service.StorageService, contentType string, fileSize int64) {
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)

	if localPath != "" {
		f, err := os.Open(localPath)
		if err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			if fileSize <= 0 {
				fileSize = stat.Size()
			}
			w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
			http.ServeContent(w, r, filepath.Base(localPath), stat.ModTime(), f)
			return
		}
	}

	if s3Key != nil && *s3Key != "" && cfg.Storage.S3.Enabled && storageService != nil {
		client := storageService.Client()
		if client == nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "S3 not available")
			return
		}

		opts := minio.GetObjectOptions{}
		rangeHeader := r.Header.Get("Range")
		var rangeStart, rangeEnd int64 = -1, -1
		if rangeHeader != "" {
			var err error
			rangeStart, rangeEnd, err = parseRange(rangeHeader, fileSize)
			if err != nil {
				slog.Warn("invalid range header", "range", rangeHeader, "error", err)
			} else {
				if setRangeErr := opts.SetRange(rangeStart, rangeEnd); setRangeErr != nil {
					slog.Warn("s3 setrange failed", "range", rangeHeader, "error", setRangeErr)
				}
			}
		}

		obj, err := client.GetObject(context.Background(), cfg.Storage.S3.Bucket, *s3Key, opts)
		if err != nil {
			slog.Warn("s3 stream failed", "key", *s3Key, "error", err)
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File not available in S3")
			return
		}
		defer obj.Close()

		stat, err := obj.Stat()
		if err != nil {
			slog.Warn("s3 stat failed", "key", *s3Key, "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to read file info")
			return
		}

		if rangeHeader != "" && rangeStart >= 0 {
			actualEnd := stat.Size - 1
			if rangeEnd >= 0 {
				actualEnd = rangeEnd
			}
			contentLen := stat.Size - rangeStart
			if rangeEnd >= 0 {
				contentLen = rangeEnd - rangeStart + 1
			}
			if contentLen > 0 {
				w.Header().Set("Content-Length", strconv.FormatInt(contentLen, 10))
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, actualEnd, stat.Size))
			w.WriteHeader(http.StatusPartialContent)
		} else if fileSize > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		}

		io.Copy(w, obj)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
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

