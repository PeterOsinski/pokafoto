package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/drive/drive/internal/backup"
	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/drive/drive/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg                 *config.Config
	db                  *store.DB
	router              *chi.Mux
	userStore           *store.UserStore
	sessStore           *store.SessionStore
	fileStore           *store.FileStore
	folderStore         *store.FolderStore
	exifStore           *store.ExifStore
	thumbnailStore      *store.ThumbnailStore
	geoStore            *store.GeoStore
	uploadJobStore      *store.UploadJobStore
	chunkStore          *store.ChunkStore
	settingStore        *store.SettingStore
	albumStore          *store.AlbumStore
	albumItemStore      *store.AlbumItemStore
	albumShareStore     *store.AlbumShareStore
	commentStore        *store.CommentStore
	reactionStore       *store.ReactionStore
	tagStore            *store.TagStore
	docStore            *store.DocumentStore
	folderPasswordStore *store.FolderPasswordStore
	folderShareStore    *store.FolderShareStore
	shareUploadStore    *store.ShareUploadStore
	storageService      *service.StorageService
	workerPool          *worker.Pool
	s3DeletionPool      *S3DeletionPool
	eventRecorder       *service.EventRecorder
	systemEventsStore   *store.SystemEventsStore
	backupScheduler     *backup.Scheduler
	stopCh              chan struct{}
}

func New(cfg *config.Config, db *store.DB) *Server {
	s := &Server{
		cfg:                cfg,
		db:                 db,
		userStore:          store.NewUserStore(db),
		sessStore:          store.NewSessionStore(db),
		fileStore:          store.NewFileStore(db),
		folderStore:        store.NewFolderStore(db),
		exifStore:          store.NewExifStore(db),
		thumbnailStore:     store.NewThumbnailStore(db),
		geoStore:           store.NewGeoStore(db),
		uploadJobStore:     store.NewUploadJobStore(db),
		chunkStore:         store.NewChunkStore(db),
		settingStore:       store.NewSettingStore(db),
		albumStore:         store.NewAlbumStore(db),
		albumItemStore:     store.NewAlbumItemStore(db),
		albumShareStore:    store.NewAlbumShareStore(db),
		commentStore:       store.NewCommentStore(db),
		reactionStore:      store.NewReactionStore(db),
		tagStore:           store.NewTagStore(db),
		docStore:           store.NewDocumentStore(db),
		folderPasswordStore: store.NewFolderPasswordStore(db),
		folderShareStore:    store.NewFolderShareStore(db),
		shareUploadStore:    store.NewShareUploadStore(db),
		stopCh:             make(chan struct{}),
	}

	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		slog.Warn("storage service init failed, continuing without S3", "error", err)
		storageService, _ = service.NewStorageService(&config.Config{}) // disabled client
	}

	s.workerPool = worker.NewPool(cfg, s.fileStore, s.exifStore, s.thumbnailStore, storageService, s.uploadJobStore, s.chunkStore, s.eventRecorder)
	s.storageService = storageService

	s.s3DeletionPool = NewS3DeletionPool(storageService)

	s.systemEventsStore = store.NewSystemEventsStore(db)
	s.eventRecorder = service.NewEventRecorder(db)

	if err != nil {
		s.eventRecorder.Warn("s3_disconnect", "S3 storage init failed", map[string]interface{}{"error": err.Error()})
	}

	s.eventRecorder.Info("server_start", "server startup complete", map[string]interface{}{
		"s3_enabled": cfg.Storage.S3.Enabled,
		"port":       cfg.Server.Port,
		"workers":    cfg.Upload.ConcurrentWorkers,
	})

	s.backupScheduler = backup.NewScheduler(cfg, db, storageService, s.eventRecorder)
	s.backupScheduler.Start()

	s.workerPool.StartReconciler(30 * time.Minute)

	go NewCacheEvictor(cfg, s.eventRecorder).Start()
	go s.startTrashCleanup()
	go s.startEventRetention()
	go s.startChunkCleanup()
	go s.startFolderPasswordCleanup()

	s.setupRouter()
	return s
}

func (s *Server) Shutdown() {
	s.eventRecorder.Info("server_shutdown", "server shutting down gracefully", nil)
	close(s.stopCh)
	s.backupScheduler.Shutdown()
	s.workerPool.Shutdown()
	s.s3DeletionPool.Shutdown()
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Folder-Unlock-Token", "X-Share-Session-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/v1/health", s.handleHealth)
	r.Get("/api/v1/auth/config", s.handleAuthConfig)

	r.Get("/api/v1/share/{token}", s.handleShareInfo)
	r.Post("/api/v1/share/{token}/unlock", s.handleShareUnlock)
	r.Get("/api/v1/share/{token}/files", s.handleShareListFiles)
	r.Get("/api/v1/share/{token}/files/{id}", s.handleShareGetFile)
	r.Get("/api/v1/share/{token}/download/{id}", s.handleShareDownload)
	r.Get("/api/v1/share/{token}/thumb/{fileID}/{size}", s.handleShareThumbnail)
	r.Post("/api/v1/share/{token}/upload", s.handleShareUpload)
	r.Delete("/api/v1/share/{token}/files/{id}", s.handleShareDeleteFile)
	r.Get("/api/v1/share/{token}/folders", s.handleShareListFolders)
	r.Post("/api/v1/share/{token}/folders", s.handleShareCreateFolder)
	r.Delete("/api/v1/share/{token}/folders/{id}", s.handleShareDeleteFolder)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/refresh", s.handleRefresh)
		r.Post("/auth/logout", s.handleLogout)

		r.Get("/upload/ws", s.handleUploadWSWithToken)

		r.Get("/thumb/{fileID}/{size}", s.handleServeThumbnail)

		r.Get("/video/{id}", s.handleVideoStreamWithToken)
		r.Get("/download/{id}", s.handleDownloadWithToken)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Get("/auth/me", s.handleMe)

			r.Post("/upload", s.handleUpload)
			r.Post("/upload/chunk", s.handleChunkUpload)
			r.Head("/upload/chunk/{resumeToken}", s.handleChunkUploadResume)
			r.Get("/upload/chunk/{resumeToken}", s.handleChunkUploadResume)
			r.Post("/upload/chunk/{resumeToken}/complete", s.handleChunkUploadComplete)
			r.Post("/upload/check", s.handleUploadCheck)
			r.Get("/upload/{batchID}/status", s.handleUploadStatus)
			r.Get("/upload/active", s.handleUploadActiveJobs)

			r.Get("/files", s.handleListFiles)
			r.Get("/files/{id}", s.handleGetFile)
			r.Delete("/files/{id}", s.handleSoftDeleteFile)
			r.Delete("/files/{id}/permanent", s.handlePermanentDeleteFile)
			r.Post("/files/batch-delete", s.handleBatchSoftDelete)
			r.Post("/files/batch-move", s.handleBatchMove)
			r.Post("/files/batch-copy", s.handleBatchCopy)

			r.Get("/dirs", s.handleListDirs)

			r.Get("/folders", s.handleListFolders)
			r.Post("/folders", s.handleCreateFolder)
			r.Put("/folders/{id}", s.handleRenameFolder)
			r.Delete("/folders/{id}", s.handleDeleteFolder)
			r.Post("/folders/{id}/password", s.handleSetFolderPassword)
			r.Delete("/folders/{id}/password", s.handleRemoveFolderPassword)
			r.Post("/folders/{id}/unlock", s.handleUnlockFolder)
			r.Get("/folders/{id}/password", s.handleGetFolderPasswordStatus)
			r.Post("/folders/{id}/shares", s.handleCreateShare)
			r.Get("/folders/{id}/shares", s.handleListShares)
			r.Put("/folders/{id}/shares/{shareId}", s.handleUpdateShare)
			r.Delete("/folders/{id}/shares/{shareId}", s.handleDeleteShare)

			r.Get("/search", s.handleSearch)

			r.Get("/timeline", s.handleTimeline)

			r.Get("/geo/points", s.handleGeoPoints)
			r.Get("/geo/clusters", s.handleGeoClusters)

			r.Get("/stats", s.handleStats)

			r.Post("/download/batch", s.handleBatchDownload)

			r.Get("/trash", s.handleListTrash)
			r.Get("/trash/stats", s.handleTrashStats)
			r.Post("/trash/{id}/restore", s.handleRestoreTrash)
			r.Post("/trash/batch-restore", s.handleBatchRestoreTrash)
			r.Delete("/trash/{id}", s.handlePermanentDeleteTrash)
			r.Post("/trash/batch-permanent-delete", s.handleBatchPermanentDeleteTrash)
			r.Post("/trash/empty", s.handleEmptyTrash)

			r.Get("/tags", s.handleListTags)
			r.Get("/tags/stats", s.handleTagStats)
			r.Get("/albums", s.handleListAlbums)
			r.Post("/albums", s.handleCreateAlbum)
			r.Get("/albums/{id}", s.handleGetAlbum)
			r.Put("/albums/{id}", s.handleUpdateAlbum)
			r.Delete("/albums/{id}", s.handleDeleteAlbum)
			r.Get("/albums/{id}/items", s.handleListAlbumItems)
			r.Post("/albums/{id}/items", s.handleAddAlbumItems)
			r.Delete("/albums/{id}/items/{itemId}", s.handleRemoveAlbumItem)
			r.Post("/albums/{id}/shares", s.handleShareAlbum)
			r.Delete("/albums/{id}/shares/{shareId}", s.handleRemoveShare)

			r.Get("/files/{id}/comments", s.handleListComments)
			r.Post("/files/{id}/comments", s.handleAddComment)
			r.Put("/files/{id}/comments/{commentId}", s.handleUpdateComment)
			r.Delete("/files/{id}/comments/{commentId}", s.handleDeleteComment)

			r.Get("/files/{id}/comments/{commentId}/reactions", s.handleGetReactions)
			r.Post("/files/{id}/comments/{commentId}/reactions", s.handleToggleReaction)
			r.Delete("/files/{id}/comments/{commentId}/reactions/{emoji}", s.handleRemoveReaction)

			r.Get("/files/{id}/tags", s.handleGetFileTags)
			r.Post("/files/{id}/tags", s.handleAddFileTags)
			r.Delete("/files/{id}/tags/{tagId}", s.handleRemoveFileTag)
			r.Get("/files/{id}/albums", s.handleGetFileAlbums)

			r.Post("/documents", s.handleCreateDocument)
			r.Get("/documents/{file_id}", s.handleGetDocument)
			r.Put("/documents/{file_id}", s.handleUpdateDocument)
			r.Delete("/documents/{file_id}", s.handleDeleteDocument)

			r.Route("/admin", func(r chi.Router) {
				r.Use(s.adminMiddleware)
				r.Get("/users", s.handleAdminListUsers)
				r.Post("/users", s.handleAdminCreateUser)
				r.Delete("/users/{id}", s.handleAdminDeleteUser)
				r.Put("/users/{id}/role", s.handleAdminUpdateRole)
				r.Put("/users/{id}/quota", s.handleAdminUpdateQuota)
				r.Get("/registration", s.handleAdminGetRegistration)
				r.Put("/registration", s.handleAdminToggleRegistration)
			r.Get("/stats", s.handleAdminStats)
			r.Get("/files/breakdown", s.handleAdminFileBreakdown)
			r.Get("/thumbnails/stats", s.handleAdminThumbnailStats)
			r.Get("/workers", s.handleAdminWorkers)
			r.Get("/jobs", s.handleAdminListJobs)
			r.Post("/jobs/{id}/retry", s.handleAdminRetryJob)
		r.Post("/jobs/reconcile", s.handleAdminReconcileJobs)
		r.Get("/s3-deletion-queue", s.handleAdminS3DeletionQueue)
		r.Get("/events", s.handleAdminListEvents)
		r.Get("/events/counts", s.handleAdminEventCounts)
		r.Get("/backup/status", s.handleAdminBackupStatus)
		r.Post("/backup", s.handleAdminTriggerBackup)
			})
		})
	})

	spa := s.serveSPA()
	if spa != nil {
		r.NotFound(spa.ServeHTTP)
	}

	s.router = r
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := true
	if err := s.db.Ping(); err != nil {
		dbOK = false
	}

	s3OK := !s.cfg.Storage.S3.Enabled || s.storageService.IsConnected()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"version":       "0.1.0",
		"db_connected":  dbOK,
		"s3_connected":  s3OK,
	})
}

func (s *Server) startEventRetention() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(24 * time.Hour):
		}
		deleted, err := s.systemEventsStore.PurgeOlderThan(90 * 24 * time.Hour)
		if err != nil {
			slog.Warn("event retention purge failed", "error", err)
		} else if deleted > 0 {
			slog.Info("purged old system events", "deleted", deleted)
		}
	}
}

func (s *Server) startFolderPasswordCleanup() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(5 * time.Minute):
		}
		s.folderPasswordStore.DeleteExpired()
	}
}
