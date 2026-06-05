package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"golang.org/x/sys/unix"
)

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

	existing, _ := s.auth.UserStore.FindByUsername(req.Username)
	if existing != nil {
		writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username is already taken")
		return
	}

	user, err := s.auth.UserStore.Create(req.Username, req.Password, role, req.DisplayName)
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
	if err := s.auth.SettingStore.Set("allow_registration", val); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update setting")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allow_registration": req.Enabled,
	})
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.auth.UserStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	userResponses := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		fileCount, _ := s.file.FileStore.Stats(u.ID)
		thumbSize, _ := s.auth.UserStore.GetThumbnailSize(u.ID)

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
	if err := s.auth.UserStore.Delete(userID); err != nil {
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

	if err := s.auth.UserStore.UpdateRole(userID, model.UserRole(req.Role)); err != nil {
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
		used, err := s.auth.UserStore.GetUsedSpace(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check usage")
			return
		}
		if *req.SpaceQuota < used {
			writeError(w, http.StatusUnprocessableEntity, "QUOTA_BELOW_USAGE", "Quota cannot be below current usage")
			return
		}
	}

	if err := s.auth.UserStore.UpdateSpaceQuota(userID, req.SpaceQuota); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update quota")
		return
	}

	user, _ := s.auth.UserStore.FindByID(userID)
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
		breakdown, err = s.file.FileStore.AdminFileBreakdownByUser(userID)
	} else {
		breakdown, err = s.file.FileStore.AdminFileBreakdown()
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

	jobs, total, err := s.upload.UploadJobStore.ListAll(limit, offset, statusFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list jobs")
		return
	}

	summary, _ := s.upload.UploadJobStore.CountByStatus()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":    jobs,
		"total":   total,
		"summary": summary,
	})
}

func (s *Server) handleAdminRetryJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if err := s.upload.UploadJobStore.Requeue(jobID); err != nil {
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
		breakdown, err = s.file.ThumbnailStore.BreakdownByUser(userID)
	} else {
		breakdown, err = s.file.ThumbnailStore.Breakdown()
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
	users, err := s.auth.UserStore.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	totalFiles := int64(0)
	totalSize := int64(0)
	userStats := make([]map[string]interface{}, 0, len(users))

	for _, u := range users {
		stats, err := s.file.FileStore.Stats(u.ID)
		if err != nil {
			continue
		}
		thumbSize, _ := s.auth.UserStore.GetThumbnailSize(u.ID)
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

	cacheSize, _ := s.file.ThumbnailStore.TotalSize()
	diskTotal, diskFree, diskUsed := diskUsage(s.cfg.Storage.Local.Path)
	diskPct := float64(0)
	if diskTotal > 0 {
		diskPct = float64(diskUsed) / float64(diskTotal) * 100
	}

	var originalsSize int64
	s.fs.Walk(s.cfg.OriginalsDir(), func(path string, info os.FileInfo, err error) error {
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

func (s *Server) handleAdminListEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	events, total, err := s.admin.SystemEventsStore.List(
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
	counts, err := s.admin.SystemEventsStore.EventCounts()
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
