package server

import (
	"log/slog"
	"net/http"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/drive/drive/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	cfg            *config.Config
	db             *store.DB
	router         *chi.Mux
	userStore      *store.UserStore
	sessStore      *store.SessionStore
	fileStore      *store.FileStore
	folderStore    *store.FolderStore
	exifStore      *store.ExifStore
	thumbnailStore *store.ThumbnailStore
	geoStore       *store.GeoStore
	uploadJobStore *store.UploadJobStore
	storageService *service.StorageService
	workerPool     *worker.Pool
}

func New(cfg *config.Config, db *store.DB) *Server {
	s := &Server{
		cfg:            cfg,
		db:             db,
		userStore:      store.NewUserStore(db),
		sessStore:      store.NewSessionStore(db),
		fileStore:      store.NewFileStore(db),
		folderStore:    store.NewFolderStore(db),
		exifStore:      store.NewExifStore(db),
		thumbnailStore: store.NewThumbnailStore(db),
		geoStore:       store.NewGeoStore(db),
		uploadJobStore: store.NewUploadJobStore(db),
	}

	storageService, err := service.NewStorageService(cfg)
	if err != nil {
		slog.Warn("storage service init failed, continuing without S3", "error", err)
		storageService, _ = service.NewStorageService(&config.Config{}) // disabled client
	}

	s.workerPool = worker.NewPool(cfg, s.fileStore, s.exifStore, s.thumbnailStore, storageService, s.uploadJobStore)
	s.storageService = storageService

	go NewCacheEvictor(cfg.ThumbnailsDir()).Start()

	s.setupRouter()
	return s
}

func (s *Server) Shutdown() {
	s.workerPool.Shutdown()
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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/api/v1/health", s.handleHealth)

	s.registerRoutes(r)

	spa := s.serveSPA()
	if spa != nil {
		r.NotFound(spa.ServeHTTP)
	}

	s.router = r
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) registerRoutes(r *chi.Mux) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/refresh", s.handleRefresh)
		r.Post("/auth/logout", s.handleLogout)

		r.Get("/upload/ws", s.handleUploadWSWithToken)

		r.Get("/thumb/{fileID}/{size}", s.handleServeThumbnail)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Get("/auth/me", s.handleMe)

			r.Post("/upload", s.handleUpload)
			r.Post("/upload/check", s.handleUploadCheck)
			r.Get("/upload/{batchID}/status", s.handleUploadStatus)

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

			r.Get("/search", s.handleSearch)

			r.Get("/timeline", s.handleTimeline)

			r.Get("/geo/points", s.handleGeoPoints)
			r.Get("/geo/clusters", s.handleGeoClusters)

			r.Get("/stats", s.handleStats)

			r.Get("/download/{id}", s.handleDownload)
			r.Post("/download/batch", s.handleBatchDownload)

			r.Route("/admin", func(r chi.Router) {
				r.Use(s.adminMiddleware)
				r.Get("/users", s.handleAdminListUsers)
				r.Delete("/users/{id}", s.handleAdminDeleteUser)
				r.Put("/users/{id}/role", s.handleAdminUpdateRole)
			r.Get("/stats", s.handleAdminStats)
			r.Get("/files/breakdown", s.handleAdminFileBreakdown)
			r.Get("/workers", s.handleAdminWorkers)
			})
		})
	})
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
