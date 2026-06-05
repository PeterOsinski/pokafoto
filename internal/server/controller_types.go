package server

import "github.com/drive/drive/internal/store"

type AuthCtl struct {
	UserStore    *store.UserStore
	SessionStore *store.SessionStore
	SettingStore *store.SettingStore
}

type FileCtl struct {
	FileStore      *store.FileStore
	ExifStore      *store.ExifStore
	GeoStore       *store.GeoStore
	TagStore       *store.TagStore
	ThumbnailStore *store.ThumbnailStore
	FolderStore    *store.FolderStore
	FolderPwStore  *store.FolderPasswordStore
	AlbumStore     *store.AlbumStore
}

type UploadCtl struct {
	UploadJobStore *store.UploadJobStore
	ChunkStore     *store.ChunkStore
	FileStore      *store.FileStore
	UserStore      *store.UserStore
}

type FolderCtl struct {
	FolderStore     *store.FolderStore
	FolderPwStore   *store.FolderPasswordStore
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
	CommentStore  *store.CommentStore
	ReactionStore *store.ReactionStore
	UserStore     *store.UserStore
	FileStore     *store.FileStore
}

type DocCtl struct {
	DocumentStore *store.DocumentStore
	FileStore     *store.FileStore
}

type DownloadCtl struct {
	FileStore *store.FileStore
}

type ShareCtl struct {
	FolderShareStore *store.FolderShareStore
	ShareUploadStore *store.ShareUploadStore
	FolderStore      *store.FolderStore
	FileStore        *store.FileStore
	FolderPwStore    *store.FolderPasswordStore
}

type AdminCtl struct {
	UserStore         *store.UserStore
	FileStore         *store.FileStore
	ThumbnailStore    *store.ThumbnailStore
	SystemEventsStore *store.SystemEventsStore
	SettingStore      *store.SettingStore
	ExifStore         *store.ExifStore
	GeoStore          *store.GeoStore
}
