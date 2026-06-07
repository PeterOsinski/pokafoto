package server

import (
	"github.com/drive/drive/internal/backup"
	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/drive/drive/internal/worker"
)

type AuthCtl struct {
	UserStore    *store.UserStore
	SessionStore *store.SessionStore
	SettingStore *store.SettingStore
	Cfg          *config.Config
}

type FileCtl struct {
	FileStore        *store.FileStore
	ExifStore        *store.ExifStore
	GeoStore         *store.GeoStore
	TagStore         *store.TagStore
	ThumbnailStore   *store.ThumbnailStore
	FolderStore      *store.FolderStore
	FolderPwStore    *store.FolderPasswordStore
	AlbumStore       *store.AlbumStore
	Storage          *service.StorageService
	S3DeletionPool   *service.S3DeletionPool
	Cfg              *config.Config
	FS               service.FileSystem
	DocumentStore    *store.DocumentStore
	AlbumItemStore   *store.AlbumItemStore
}

type UploadCtl struct {
	UploadJobStore *store.UploadJobStore
	ChunkStore     *store.ChunkStore
	FileStore      *store.FileStore
	UserStore      *store.UserStore
	WorkerPool     *worker.Pool
	Cfg            *config.Config
	FS             service.FileSystem
	FolderPwStore  *store.FolderPasswordStore
	FolderStore    *store.FolderStore
}

type FolderCtl struct {
	FolderStore      *store.FolderStore
	FolderPwStore    *store.FolderPasswordStore
	FolderShareStore *store.FolderShareStore
	ShareUploadStore *store.ShareUploadStore
}

type AlbumCtl struct {
	AlbumStore      *store.AlbumStore
	AlbumItemStore  *store.AlbumItemStore
	AlbumShareStore *store.AlbumShareStore
	UserStore       *store.UserStore
	FileStore       *store.FileStore
}

type CommentCtl struct {
	CommentStore   *store.CommentStore
	ReactionStore  *store.ReactionStore
	UserStore      *store.UserStore
	FileStore      *store.FileStore
	AlbumItemStore *store.AlbumItemStore
}

type DocCtl struct {
	DocumentStore *store.DocumentStore
	FileStore     *store.FileStore
}

type DownloadCtl struct {
	FileStore     *store.FileStore
	Storage       *service.StorageService
	Cfg           *config.Config
	FS            service.FileSystem
	DocumentStore *store.DocumentStore
}

type ShareCtl struct {
	FolderShareStore *store.FolderShareStore
	ShareUploadStore *store.ShareUploadStore
	FolderStore      *store.FolderStore
	FileStore        *store.FileStore
	FolderPwStore    *store.FolderPasswordStore
	Cfg              *config.Config
	FS               service.FileSystem
	Storage          *service.StorageService
	DocumentStore    *store.DocumentStore
	S3DeletionPool   *service.S3DeletionPool
	UploadJobStore   *store.UploadJobStore
	WorkerPool       *worker.Pool
}

type AdminCtl struct {
	UserStore         *store.UserStore
	FileStore         *store.FileStore
	ThumbnailStore    *store.ThumbnailStore
	SystemEventsStore *store.SystemEventsStore
	SettingStore      *store.SettingStore
	ExifStore         *store.ExifStore
	GeoStore          *store.GeoStore
	DB                *store.DB
	Storage           *service.StorageService
	WorkerPool        *worker.Pool
	S3DeletionPool    *service.S3DeletionPool
	Scheduler         *backup.Scheduler
	S3Enabled         bool
	Cfg               *config.Config
	FS                service.FileSystem
	UploadJobStore    *store.UploadJobStore
}
