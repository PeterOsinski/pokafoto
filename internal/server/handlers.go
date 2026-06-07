package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/minio/minio-go/v7"
)

func (c *FileCtl) HandleListFiles(w http.ResponseWriter, r *http.Request) {
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
		if !c.CheckFolderAccess(*opts.FolderID, userID, r) {
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

	files, nextCursor, total, err := c.FileStore.List(opts)
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

func (c *FileCtl) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !c.CheckFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !c.CheckFolderAccess(*file.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
	}

	exif, _ := c.ExifStore.FindByFileID(fileID)

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

func (c *FileCtl) HandleSoftDeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if err := c.FileStore.SoftDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleBatchSoftDelete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := c.FileStore.BatchSoftDelete(userID, req.IDs); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleBatchMove(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs      []string `json:"ids"`
		FolderID *string  `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := c.FileStore.BatchMove(userID, req.IDs, req.FolderID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleBatchCopy(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs      []string `json:"ids"`
		FolderID *string  `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	copies, err := c.FileStore.BatchCopy(userID, req.IDs, req.FolderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to copy files")
		return
	}

	copyIDs := make([]string, len(copies))
	for i, cp := range copies {
		copyIDs[i] = cp.ID
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(copies),
		"ids":   copyIDs,
	})
}

func (c *FileCtl) HandlePermanentDeleteFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	filePath := filepath.Join(c.Cfg.OriginalsDir(), userID, file.Filename)
	c.FS.Remove(filePath)

	if err := c.FileStore.PermanentDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleServeThumbnail(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fileID")
	size := r.PathValue("size")

	if err := c.checkThumbnailAccess(w, r, fileID); err != nil {
		return
	}

	thumbPath := filepath.Join(c.Cfg.ThumbnailsDir(), fileID, size)

	if _, err := c.FS.Stat(thumbPath); err == nil {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, thumbPath)
		return
	}

	if c.Cfg.Storage.S3.Enabled && c.Storage != nil {
		s3Key := fmt.Sprintf("thumbnails/%s/%s", fileID, size)
		if err := c.Storage.GetObject(s3Key, thumbPath); err != nil {
			slog.Warn("s3 thumbnail fallback failed", "key", s3Key, "error", err)
			c.fallbackThumbnail(w, r, fileID, size)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, thumbPath)
		return
	}

	c.fallbackThumbnail(w, r, fileID, size)
}

func (c *FileCtl) checkThumbnailAccess(w http.ResponseWriter, r *http.Request, fileID string) error {
	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		return nil
	}

	if file.FolderID == nil {
		return nil
	}

	fp, err := c.FolderPwStore.FindByFolderID(*file.FolderID)
	if err != nil {
		return nil
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		c.FolderPwStore.DeleteByFolderID(*file.FolderID)
		return nil
	}

	unlockToken := r.Header.Get("X-Folder-Unlock-Token")
	if unlockToken != "" {
		unlockedFolderID, ok := c.ParseFolderUnlockToken(unlockToken)
		if ok && unlockedFolderID == *file.FolderID {
			return nil
		}
	}

	writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock to view thumbnails")
	return fmt.Errorf("folder password required")
}

func (c *FileCtl) fallbackThumbnail(w http.ResponseWriter, r *http.Request, fileID, size string) {
	fallbackMap := map[string]string{
		"preview.webp": "md.jpg",
		"lg.jpg":       "md.jpg",
		"md.jpg":       "sm.jpg",
	}
	if fallback, ok := fallbackMap[size]; ok {
		fallbackPath := filepath.Join(c.Cfg.ThumbnailsDir(), fileID, fallback)
		if _, err := c.FS.Stat(fallbackPath); err == nil {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			http.ServeFile(w, r, fallbackPath)
			return
		}
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "Thumbnail not found")
}

func (c *FileCtl) HandleListDirs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	allFolders := r.URL.Query().Get("all_folders") == "true"

	root, err := c.FileStore.ListDirs(userID, allFolders)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list directories")
		return
	}

	writeJSON(w, http.StatusOK, root)
}

func (c *FileCtl) HandleSearch(w http.ResponseWriter, r *http.Request) {
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

	result, folderPaths, err := c.FileStore.SearchEnhanced(opts)
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

func (c *FileCtl) HandleTimeline(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	granularity := r.URL.Query().Get("granularity")

	groups, err := c.FileStore.Timeline(userID, granularity)
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

func (c *FileCtl) HandleGeoPoints(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	latMin, _ := strconv.ParseFloat(r.URL.Query().Get("lat_min"), 64)
	latMax, _ := strconv.ParseFloat(r.URL.Query().Get("lat_max"), 64)
	lonMin, _ := strconv.ParseFloat(r.URL.Query().Get("lon_min"), 64)
	lonMax, _ := strconv.ParseFloat(r.URL.Query().Get("lon_max"), 64)

	points, err := c.GeoStore.GetPoints(userID, store.GeoBounds{
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

func (c *FileCtl) HandleGeoClusters(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	latMin, _ := strconv.ParseFloat(r.URL.Query().Get("lat_min"), 64)
	latMax, _ := strconv.ParseFloat(r.URL.Query().Get("lat_max"), 64)
	lonMin, _ := strconv.ParseFloat(r.URL.Query().Get("lon_min"), 64)
	lonMax, _ := strconv.ParseFloat(r.URL.Query().Get("lon_max"), 64)
	zoom, _ := strconv.Atoi(r.URL.Query().Get("zoom"))

	points, err := c.GeoStore.GetPoints(userID, store.GeoBounds{
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
		if cl, ok := clusters[k]; ok {
			cl.latSum += p.Latitude
			cl.lonSum += p.Longitude
			cl.count++
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
	for _, cl := range clusters {
		result = append(result, map[string]interface{}{
			"latitude":     cl.latSum / float64(cl.count),
			"longitude":    cl.lonSum / float64(cl.count),
			"count":        cl.count,
			"thumbnailUrl": fmt.Sprintf("/api/v1/thumb/%s/sm.jpg", cl.fileID),
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

func (c *FileCtl) HandleStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	stats, err := c.FileStore.Stats(userID)
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

func (c *FileCtl) HandleDownload(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !c.CheckFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !c.CheckFolderAccess(*file.FolderID, userID, r) {
			writeError(w, http.StatusForbidden, "FOLDER_PASSWORD_REQUIRED", "Folder requires password unlock")
			return
		}
	}

	if file.IsAppManaged {
		doc, err := c.DocumentStore.FindByFileID(fileID)
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

	filePath := filepath.Join(c.Cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := c.FS.Stat(filePath); err == nil {
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

	if c.Cfg.Storage.S3.Enabled && c.Storage != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalName))
		w.Header().Set("Accept-Ranges", "bytes")
		if file.MimeType != "" {
			w.Header().Set("Content-Type", file.MimeType)
		}
		if file.SHA256 != "" {
			w.Header().Set("ETag", fmt.Sprintf(`"%s"`, file.SHA256))
		}
		serveFileWithRange(w, r, "", &s3Key, c.Cfg, c.Storage, c.FS, file.MimeType, file.SizeBytes)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk")
}

func (c *FileCtl) HandleVideoStream(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.UserID != userID && !c.CheckFileAccess(fileID, userID) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found")
		return
	}

	if file.FolderID != nil {
		if !c.CheckFolderAccess(*file.FolderID, userID, r) {
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
		thumb, err := c.ThumbnailStore.FindByFileIDAndSize(fileID, model.ThumbSizeVideoProxy)
		if err == nil && thumb != nil {
			serveFileWithRange(w, r, thumb.LocalPath, thumb.S3Key, c.Cfg, c.Storage, c.FS, "video/mp4", thumb.SizeBytes)
			return
		}
	}

	filePath := filepath.Join(c.Cfg.OriginalsDir(), file.UserID, file.Filename)
	if _, err := c.FS.Stat(filePath); err == nil {
		serveFileWithRange(w, r, filePath, nil, c.Cfg, c.Storage, c.FS, file.MimeType, file.SizeBytes)
		return
	}

	if c.Cfg.Storage.S3.Enabled && c.Storage != nil {
		s3Key := fmt.Sprintf("originals/%s/%s", file.UserID, file.Filename)
		serveFileWithRange(w, r, "", &s3Key, c.Cfg, c.Storage, c.FS, file.MimeType, file.SizeBytes)
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

func serveFileWithRange(w http.ResponseWriter, r *http.Request, localPath string, s3Key *string, cfg *config.Config, storageService *service.StorageService, fs service.FileSystem, contentType string, fileSize int64) {
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)

	if localPath != "" {
		f, err := fs.Open(localPath)
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

func (c *FileCtl) HandleListTrash(w http.ResponseWriter, r *http.Request) {
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

	files, nextCursor, total, err := c.FileStore.ListTrash(opts)
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

func (c *FileCtl) HandleTrashStats(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	stats, err := c.FileStore.TrashStats(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get trash stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      stats.Count,
		"size_bytes": stats.SizeBytes,
	})
}

func (c *FileCtl) HandleRestoreTrash(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID || !file.IsDeleted {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found in trash")
		return
	}

	if err := c.FileStore.Restore(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleBatchRestoreTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	if err := c.FileStore.BatchRestore(userID, req.IDs); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore files")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandlePermanentDeleteTrash(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	file, err := c.FileStore.FindByID(fileID)
	if err != nil || file == nil || file.UserID != userID || !file.IsDeleted {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found in trash")
		return
	}

	originalPath := filepath.Join(c.Cfg.OriginalsDir(), file.UserID, file.Filename)
	c.FS.Remove(originalPath)
	thumbDir := filepath.Join(c.Cfg.ThumbnailsDir(), file.ID)
	c.FS.RemoveAll(thumbDir)

	c.enqueueS3Deletion(file.ID, file.UserID, file.Filename)

	if err := c.FileStore.PermanentDelete(fileID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete file")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleBatchPermanentDeleteTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty ids array")
		return
	}

	files, err := c.FileStore.ListTrashFiles(userID, req.IDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query files")
		return
	}

	var deleteIDs []string
	for _, f := range files {
		originalPath := filepath.Join(c.Cfg.OriginalsDir(), f.UserID, f.Filename)
		c.FS.Remove(originalPath)
		thumbDir := filepath.Join(c.Cfg.ThumbnailsDir(), f.ID)
		c.FS.RemoveAll(thumbDir)
		c.enqueueS3Deletion(f.ID, f.UserID, f.Filename)
		deleteIDs = append(deleteIDs, f.ID)
	}

	if len(deleteIDs) > 0 {
		if err := c.FileStore.BatchPermanentDelete(userID, deleteIDs); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to permanently delete files")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleEmptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	files, err := c.FileStore.ListAllTrashFiles(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query trash")
		return
	}

	type emptyRecord struct {
		id  string
		uid string
		fn  string
	}
	var records []emptyRecord
	for _, f := range files {
		originalPath := filepath.Join(c.Cfg.OriginalsDir(), f.UserID, f.Filename)
		c.FS.Remove(originalPath)
		thumbDir := filepath.Join(c.Cfg.ThumbnailsDir(), f.ID)
		c.FS.RemoveAll(thumbDir)
		records = append(records, emptyRecord{f.ID, f.UserID, f.Filename})
	}

	var allIDs []string
	for _, r := range records {
		c.enqueueS3Deletion(r.id, r.uid, r.fn)
		allIDs = append(allIDs, r.id)
	}

	for i := 0; i < len(allIDs); i += 100 {
		end := i + 100
		if end > len(allIDs) {
			end = len(allIDs)
		}
		if err := c.FileStore.BatchPermanentDelete(userID, allIDs[i:end]); err != nil {
			slog.Warn("empty trash chunk failed", "error", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) CheckFileAccess(fileID, userID string) bool {
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

func (c *FileCtl) CheckCommentWriteAccess(fileID, userID string) bool {
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

func (c *FileCtl) FolderPasswordExpiryDuration() time.Duration {
	minutes := 30
	if c.Cfg.Auth.FolderPasswordExpiryMinutes > 0 {
		minutes = c.Cfg.Auth.FolderPasswordExpiryMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func (c *FileCtl) folderUnlockSecret() string {
	return c.Cfg.Auth.JWTSecret + ":folder_unlock"
}

func (c *FileCtl) ParseFolderUnlockToken(tokenStr string) (string, bool) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(c.folderUnlockSecret()), nil
	})
	if err != nil || !token.Valid {
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	sub, _ := claims["sub"].(string)
	if sub != "folder_unlock" {
		return "", false
	}

	folderID, _ := claims["folder_id"].(string)
	return folderID, folderID != ""
}

func (c *FileCtl) GenerateFolderUnlockToken(folderID string, expiryTime time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":       "folder_unlock",
		"folder_id": folderID,
		"iat":       time.Now().Unix(),
		"exp":       expiryTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.folderUnlockSecret()))
}

func (c *FileCtl) CheckFolderAccess(folderID, userID string, r *http.Request) bool {
	fp, err := c.FolderPwStore.FindByFolderID(folderID)
	if err != nil {
		return true
	}

	now := time.Now().UTC()
	if now.After(fp.ExpiresAt) {
		c.FolderPwStore.DeleteByFolderID(folderID)
		return true
	}

	if _, err := c.FolderStore.FindByID(folderID); err != nil {
		return false
	}

	unlockToken := r.Header.Get("X-Folder-Unlock-Token")
	if unlockToken == "" {
		unlockToken = r.URL.Query().Get("folder_unlock_token")
	}
	if unlockToken != "" {
		unlockedFolderID, ok := c.ParseFolderUnlockToken(unlockToken)
		if ok && unlockedFolderID == folderID {
			return true
		}
	}

	return false
}

func (c *FileCtl) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty name")
		return
	}

	folder, err := c.FolderStore.Create(userID, req.Name, req.ParentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create folder")
		return
	}

	writeJSON(w, http.StatusCreated, folder)
}

func (c *FileCtl) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	root, err := c.FolderStore.ListTree(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list folders")
		return
	}

	if root == nil {
		root = &model.FolderTreeNode{Children: []*model.FolderTreeNode{}}
	}

	writeJSON(w, http.StatusOK, root)
}

func (c *FileCtl) HandleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Name     *string `json:"name"`
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Name == nil && req.ParentID == nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide name or parent_id")
		return
	}

	folder, err := c.FolderStore.FindByID(folderID)
	if err != nil || folder == nil || folder.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}

	if req.Name != nil {
		if *req.Name == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Name cannot be empty")
			return
		}
		if err := c.FolderStore.UpdateName(folderID, *req.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rename folder")
			return
		}
	}

	if req.ParentID != nil {
		if *req.ParentID != "" {
			if *req.ParentID == folderID {
				writeError(w, http.StatusBadRequest, "CIRCULAR_MOVE", "Cannot move folder into itself")
				return
			}
			isDesc, err := c.FolderStore.IsDescendant(*req.ParentID, folderID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate move")
				return
			}
			if isDesc {
				writeError(w, http.StatusBadRequest, "CIRCULAR_MOVE", "Cannot move folder into its own descendant")
				return
			}
			parent, err := c.FolderStore.FindByID(*req.ParentID)
			if err != nil || parent == nil || parent.UserID != userID {
				writeError(w, http.StatusBadRequest, "INVALID_PARENT", "Target folder not found")
				return
			}
			if err := c.FolderStore.UpdateParent(folderID, req.ParentID); err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move folder")
				return
			}
		} else {
			if err := c.FolderStore.UpdateParent(folderID, nil); err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to move folder")
				return
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *FileCtl) HandleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	folderID := r.PathValue("id")
	userID := getUserID(r)

	folder, err := c.FolderStore.FindByID(folderID)
	if err != nil || folder == nil || folder.UserID != userID {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Folder not found")
		return
	}

	result, err := c.FolderStore.DeleteRecursive(folderID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete folder")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted_files":   result.DeletedFiles,
		"deleted_folders": result.DeletedFolders,
	})
}

func (c *FileCtl) HandleRenameFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	userID := getUserID(r)

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Provide a non-empty name")
		return
	}

	if err := c.FileStore.Rename(fileID, userID, req.Name); err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "File not found or access denied")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
