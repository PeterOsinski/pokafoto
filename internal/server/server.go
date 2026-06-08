package server

import (
	"log/slog"
	"net/http"
	"strings"
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
	cfg           *config.Config
	router        *chi.Mux
	fs            service.FileSystem
	eventRecorder *service.EventRecorder
	stopCh        chan struct{}

	auth     *AuthCtl
	file     *FileCtl
	upload   *UploadCtl
	folder   *FolderCtl
	album    *AlbumCtl
	comment  *CommentCtl
	doc      *DocCtl
	download *DownloadCtl
	share    *ShareCtl
	admin    *AdminCtl
}

func New(cfg *config.Config, db *store.DB) *Server {
	fs := service.NewRealFS()

	userStore := store.NewUserStore(db)
	sessStore := store.NewSessionStore(db)
	fileStore := store.NewFileStore(db)
	folderStore := store.NewFolderStore(db)
	thumbnailStore := store.NewThumbnailStore(db)
	uploadJobStore := store.NewUploadJobStore(db)
	chunkStore := store.NewChunkStore(db, fs)
	settingStore := store.NewSettingStore(db)
	albumStore := store.NewAlbumStore(db)
	albumItemStore := store.NewAlbumItemStore(db)
	albumShareStore := store.NewAlbumShareStore(db)
	commentStore := store.NewCommentStore(db)
	reactionStore := store.NewReactionStore(db)
	tagStore := store.NewTagStore(db)
	docStore := store.NewDocumentStore(db)
	folderPasswordStore := store.NewFolderPasswordStore(db)
	folderShareStore := store.NewFolderShareStore(db)
	shareUploadStore := store.NewShareUploadStore(db)
	exifStore := store.NewExifStore(db)
	geoStore := store.NewGeoStore(db)
	systemEventsStore := store.NewSystemEventsStore(db)

	storageService, err := service.NewStorageService(cfg, fs)
	if err != nil {
		slog.Warn("storage service init failed, continuing without S3", "error", err)
		storageService, _ = service.NewStorageService(&config.Config{}, fs)
	}

	s3DeletionPool := service.NewS3DeletionPool(storageService, thumbnailStore)
	eventRecorder := service.NewEventRecorder(db)
	workerPool := worker.NewPool(cfg, fs, fileStore, exifStore, thumbnailStore, storageService, uploadJobStore, chunkStore, eventRecorder)

	s := &Server{
		cfg:           cfg,
		fs:            fs,
		stopCh:        make(chan struct{}),
		eventRecorder: eventRecorder,
		auth: &AuthCtl{
			UserStore:    userStore,
			SessionStore: sessStore,
			SettingStore: settingStore,
			Cfg:          cfg,
		},
		file: &FileCtl{
			FileStore:      fileStore,
			ExifStore:      exifStore,
			GeoStore:       geoStore,
			TagStore:       tagStore,
			ThumbnailStore: thumbnailStore,
			FolderStore:    folderStore,
			FolderPwStore:  folderPasswordStore,
			AlbumStore:     albumStore,
			Storage:        storageService,
			S3DeletionPool: s3DeletionPool,
			Cfg:            cfg,
			FS:             fs,
			DocumentStore:  docStore,
			AlbumItemStore: albumItemStore,
		},
		upload: &UploadCtl{
			UploadJobStore: uploadJobStore,
			ChunkStore:     chunkStore,
			FileStore:      fileStore,
			UserStore:      userStore,
			WorkerPool:     workerPool,
			Cfg:            cfg,
			FS:             fs,
			FolderPwStore:  folderPasswordStore,
			FolderStore:    folderStore,
		},
		folder: &FolderCtl{
			FolderStore:      folderStore,
			FolderPwStore:    folderPasswordStore,
			FolderShareStore: folderShareStore,
			ShareUploadStore: shareUploadStore,
		},
		album: &AlbumCtl{
			AlbumStore:      albumStore,
			AlbumItemStore:  albumItemStore,
			AlbumShareStore: albumShareStore,
			UserStore:       userStore,
			FileStore:       fileStore,
		},
		comment: &CommentCtl{
			CommentStore:   commentStore,
			ReactionStore:  reactionStore,
			UserStore:      userStore,
			FileStore:      fileStore,
			AlbumItemStore: albumItemStore,
		},
		doc: &DocCtl{
			DocumentStore: docStore,
			FileStore:     fileStore,
		},
		download: &DownloadCtl{
			FileStore:     fileStore,
			Storage:       storageService,
			Cfg:           cfg,
			FS:            fs,
			DocumentStore: docStore,
		},
		share: &ShareCtl{
			FolderShareStore: folderShareStore,
			ShareUploadStore: shareUploadStore,
			FolderStore:      folderStore,
			FileStore:        fileStore,
			FolderPwStore:    folderPasswordStore,
			Cfg:              cfg,
			FS:               fs,
			Storage:          storageService,
			DocumentStore:    docStore,
			S3DeletionPool:   s3DeletionPool,
			UploadJobStore:   uploadJobStore,
			WorkerPool:       workerPool,
		},
		admin: &AdminCtl{
			UserStore:         userStore,
			FileStore:         fileStore,
			ThumbnailStore:    thumbnailStore,
			SystemEventsStore: systemEventsStore,
			SettingStore:      settingStore,
			ExifStore:         exifStore,
			GeoStore:          geoStore,
			DB:                db,
			Storage:           storageService,
			WorkerPool:        workerPool,
			S3DeletionPool:    s3DeletionPool,
			S3Enabled:         cfg.Storage.S3.Enabled,
			Cfg:               cfg,
			FS:                fs,
			UploadJobStore:    uploadJobStore,
		},
	}

	if err != nil {
		s.eventRecorder.Warn("s3_disconnect", "S3 storage init failed", map[string]interface{}{"error": err.Error()})
	}

	s.eventRecorder.Info("server_start", "server startup complete", map[string]interface{}{
		"s3_enabled": cfg.Storage.S3.Enabled,
		"port":       cfg.Server.Port,
		"workers":    cfg.Upload.ConcurrentWorkers,
	})

	s.admin.Scheduler = backup.NewScheduler(cfg, db, storageService, s.eventRecorder, fs)
	s.admin.Scheduler.Start()

	s.upload.WorkerPool.StartReconciler(30 * time.Minute)

	service.NewCacheEvictor(cfg, s.fs, s.eventRecorder).Start()
	service.NewTrashCleanup(s.file.FileStore, s.fs, s.cfg.OriginalsDir(), s.cfg.ThumbnailsDir(), s.cfg.TrashExpirationDays, s.file.enqueueS3Deletion).Start(s.stopCh)
	service.NewEventRetention(s.admin.SystemEventsStore).Start(s.stopCh)
	service.NewChunkCleanup(s.upload.ChunkStore, s.cfg.Upload.ChunkCleanupHours, s.cfg.Upload.MaxChunkUploadAgeHours).Start(s.stopCh)
	service.NewFolderPasswordCleanup(s.file.FolderPwStore).Start(s.stopCh)

	s.setupRouter()
	return s
}

func (s *Server) Shutdown() {
	s.eventRecorder.Info("server_shutdown", "server shutting down gracefully", nil)
	close(s.stopCh)
	s.admin.Scheduler.Shutdown()
	s.upload.WorkerPool.Shutdown()
	s.admin.S3DeletionPool.Shutdown()
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

	r.Get("/api/v1/health", s.admin.handleHealth)
	r.Get("/api/v1/auth/config", s.auth.HandleAuthConfig)

	r.Get("/api/v1/share/{token}", s.share.HandleShareInfo)
	r.Post("/api/v1/share/{token}/unlock", s.share.HandleShareUnlock)
	r.Get("/api/v1/share/{token}/files", s.share.HandleShareListFiles)
	r.Get("/api/v1/share/{token}/files/{id}", s.share.HandleShareGetFile)
	r.Get("/api/v1/share/{token}/download/{id}", s.share.HandleShareDownload)
	r.Get("/api/v1/share/{token}/thumb/{fileID}/{size}", s.share.HandleShareThumbnail)
	r.Post("/api/v1/share/{token}/upload", s.share.HandleShareUpload)
	r.Delete("/api/v1/share/{token}/files/{id}", s.share.HandleShareDeleteFile)
	r.Get("/api/v1/share/{token}/folders", s.share.HandleShareListFolders)
	r.Post("/api/v1/share/{token}/folders", s.share.HandleShareCreateFolder)
	r.Delete("/api/v1/share/{token}/folders/{id}", s.share.HandleShareDeleteFolder)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", s.auth.HandleRegister)
		r.Post("/auth/login", s.auth.HandleLogin)
		r.Post("/auth/refresh", s.auth.HandleRefresh)
		r.Post("/auth/logout", s.auth.HandleLogout)

		r.Get("/upload/ws", s.upload.HandleUploadWSWithToken)
		r.Post("/upload/progress-flush", s.upload.HandleProgressFlush)

		r.Get("/thumb/{fileID}/{size}", s.file.HandleServeThumbnail)

		r.Get("/video/{id}", s.handleVideoStreamWithToken)
		r.Get("/download/{id}", s.handleDownloadWithToken)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Get("/auth/me", s.auth.HandleMe)

			r.Post("/upload", s.upload.HandleUpload)
			r.Post("/upload/chunk", s.upload.HandleChunkUpload)
			r.Head("/upload/chunk/{resumeToken}", s.upload.HandleChunkUploadResume)
			r.Get("/upload/chunk/{resumeToken}", s.upload.HandleChunkUploadResume)
			r.Post("/upload/chunk/{resumeToken}/complete", s.upload.HandleChunkUploadComplete)
			r.Post("/upload/check", s.upload.HandleUploadCheck)
			r.Get("/upload/{batchID}/status", s.upload.HandleUploadStatus)
			r.Get("/upload/active", s.upload.HandleUploadActiveJobs)

			r.Get("/files", s.file.HandleListFiles)
			r.Get("/files/{id}", s.file.HandleGetFile)
			r.Delete("/files/{id}", s.file.HandleSoftDeleteFile)
			r.Delete("/files/{id}/permanent", s.file.HandlePermanentDeleteFile)
			r.Put("/files/{id}/rename", s.file.HandleRenameFile)
			r.Post("/files/batch-delete", s.file.HandleBatchSoftDelete)
			r.Post("/files/batch-move", s.file.HandleBatchMove)
			r.Post("/files/batch-copy", s.file.HandleBatchCopy)

			r.Get("/dirs", s.file.HandleListDirs)

			r.Get("/folders", s.file.HandleListFolders)
			r.Post("/folders", s.file.HandleCreateFolder)
			r.Put("/folders/{id}", s.file.HandleUpdateFolder)
			r.Delete("/folders/{id}", s.file.HandleDeleteFolder)
			r.Post("/folders/{id}/password", s.file.HandleSetFolderPassword)
			r.Delete("/folders/{id}/password", s.file.HandleRemoveFolderPassword)
			r.Post("/folders/{id}/unlock", s.file.HandleUnlockFolder)
			r.Get("/folders/{id}/password", s.file.HandleGetFolderPasswordStatus)
			r.Post("/folders/{id}/shares", s.share.HandleCreateShare)
			r.Get("/folders/{id}/shares", s.share.HandleListShares)
			r.Put("/folders/{id}/shares/{shareId}", s.share.HandleUpdateShare)
			r.Delete("/folders/{id}/shares/{shareId}", s.share.HandleDeleteShare)

			r.Get("/search", s.file.HandleSearch)

			r.Get("/timeline", s.file.HandleTimeline)

			r.Get("/geo/points", s.file.HandleGeoPoints)
			r.Get("/geo/clusters", s.file.HandleGeoClusters)

			r.Get("/stats", s.file.HandleStats)

			r.Post("/download/batch", s.file.HandleBatchDownload)

			r.Get("/trash", s.file.HandleListTrash)
			r.Get("/trash/stats", s.file.HandleTrashStats)
			r.Post("/trash/{id}/restore", s.file.HandleRestoreTrash)
			r.Post("/trash/batch-restore", s.file.HandleBatchRestoreTrash)
			r.Delete("/trash/{id}", s.file.HandlePermanentDeleteTrash)
			r.Post("/trash/batch-permanent-delete", s.file.HandleBatchPermanentDeleteTrash)
			r.Post("/trash/empty", s.file.HandleEmptyTrash)

			r.Get("/tags", s.file.HandleListTags)
			r.Get("/tags/stats", s.file.HandleTagStats)
			r.Get("/albums", s.album.HandleListAlbums)
			r.Post("/albums", s.album.HandleCreateAlbum)
			r.Get("/albums/{id}", s.album.HandleGetAlbum)
			r.Put("/albums/{id}", s.album.HandleUpdateAlbum)
			r.Delete("/albums/{id}", s.album.HandleDeleteAlbum)
			r.Get("/albums/{id}/items", s.album.HandleListAlbumItems)
			r.Post("/albums/{id}/items", s.album.HandleAddAlbumItems)
			r.Delete("/albums/{id}/items/{itemId}", s.album.HandleRemoveAlbumItem)
			r.Post("/albums/{id}/shares", s.album.HandleShareAlbum)
			r.Delete("/albums/{id}/shares/{shareId}", s.album.HandleRemoveShare)

			r.Get("/files/{id}/comments", s.comment.HandleListComments)
			r.Post("/files/{id}/comments", s.comment.HandleAddComment)
			r.Put("/files/{id}/comments/{commentId}", s.comment.HandleUpdateComment)
			r.Delete("/files/{id}/comments/{commentId}", s.comment.HandleDeleteComment)

			r.Get("/files/{id}/comments/{commentId}/reactions", s.comment.HandleGetReactions)
			r.Post("/files/{id}/comments/{commentId}/reactions", s.comment.HandleToggleReaction)
			r.Delete("/files/{id}/comments/{commentId}/reactions/{emoji}", s.comment.HandleRemoveReaction)

			r.Get("/files/{id}/tags", s.file.HandleGetFileTags)
			r.Post("/files/{id}/tags", s.file.HandleAddFileTags)
			r.Delete("/files/{id}/tags/{tagId}", s.file.HandleRemoveFileTag)
			r.Get("/files/{id}/albums", s.file.HandleGetFileAlbums)

			r.Post("/documents", s.doc.HandleCreateDocument)
			r.Get("/documents/{file_id}", s.doc.HandleGetDocument)
			r.Put("/documents/{file_id}", s.doc.HandleUpdateDocument)
			r.Delete("/documents/{file_id}", s.doc.HandleDeleteDocument)

			r.Route("/admin", func(r chi.Router) {
				r.Use(s.adminMiddleware)
				r.Get("/users", s.admin.HandleAdminListUsers)
				r.Post("/users", s.admin.HandleAdminCreateUser)
				r.Delete("/users/{id}", s.admin.HandleAdminDeleteUser)
				r.Put("/users/{id}/role", s.admin.HandleAdminUpdateRole)
				r.Put("/users/{id}/quota", s.admin.HandleAdminUpdateQuota)
				r.Get("/registration", s.admin.HandleAdminGetRegistration)
				r.Put("/registration", s.admin.HandleAdminToggleRegistration)
				r.Get("/stats", s.admin.HandleAdminStats)
				r.Get("/files/breakdown", s.admin.HandleAdminFileBreakdown)
				r.Get("/thumbnails/stats", s.admin.HandleAdminThumbnailStats)
				r.Get("/workers", s.admin.HandleAdminWorkers)
				r.Get("/jobs", s.admin.HandleAdminListJobs)
				r.Post("/jobs/{id}/retry", s.admin.HandleAdminRetryJob)
				r.Post("/jobs/reconcile", s.admin.HandleAdminReconcileJobs)
				r.Get("/s3-deletion-queue", s.admin.HandleAdminS3DeletionQueue)
				r.Get("/events", s.admin.HandleAdminListEvents)
				r.Get("/events/counts", s.admin.HandleAdminEventCounts)
				r.Get("/backup/status", s.admin.HandleAdminBackupStatus)
				r.Post("/backup", s.admin.HandleAdminTriggerBackup)
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

func (s *Server) handleVideoStreamWithToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if !s.upload.ValidateTokenAndSetContext(w, r, tokenStr) {
			return
		}
	} else {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			return
		}
		if !s.upload.ValidateTokenAndSetContext(w, r, token) {
			return
		}
	}
	s.file.HandleVideoStream(w, r)
}

func (s *Server) handleDownloadWithToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if !s.upload.ValidateTokenAndSetContext(w, r, tokenStr) {
			return
		}
	} else {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing token")
			return
		}
		if !s.upload.ValidateTokenAndSetContext(w, r, token) {
			return
		}
	}
	s.file.HandleDownload(w, r)
}

func (c *AdminCtl) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbOK := true
	if err := c.DB.Ping(); err != nil {
		dbOK = false
	}

	s3OK := !c.S3Enabled || c.Storage.IsConnected()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"version":       "0.1.0",
		"db_connected":  dbOK,
		"s3_connected":  s3OK,
	})
}
