package store

import (
	"time"

	"github.com/drive/drive/internal/model"
)

type UserRepository interface {
	Create(username, password string, role model.UserRole, displayName *string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindByID(id string) (*model.User, error)
	List() ([]*model.User, error)
	UpdateRole(id string, role model.UserRole) error
	Delete(id string) error
	Count() (int, error)
	UpdateSpaceQuota(id string, quota *int64) error
	GetUsedSpace(userID string) (int64, error)
	GetThumbnailSize(userID string) (int64, error)
}

type SessionRepository interface {
	Create(userID string, expiresAt time.Time) (*model.Session, error)
	FindByRefreshToken(token string) (*model.Session, error)
	Delete(id string) error
	DeleteByRefreshToken(token string) error
	DeleteByUserID(userID string) error
}

type FileRepository interface {
	Create(file *model.File) error
	FindByID(id string) (*model.File, error)
	FindBySHA256(userID, sha256 string) (*model.File, error)
	FindByNameAndSize(userID, name string, size int64) (*model.File, error)
	FindByNameAndSizeBatch(userID string, nameSizes []FileRecord) ([]*model.File, error)
	List(opts FileListOptions) ([]*model.File, string, int, error)
	SoftDelete(id string) error
	PermanentDelete(id string) error
	Stats(userID string) (*StatsResult, error)
	ListDirs(userID string, allFolders bool) (*DirEntry, error)
	SearchEnhanced(opts SearchOptions) (*SearchResult, map[string]string, error)
	Search(userID, query string, limit int) (*SearchResult, error)
	Timeline(userID, granularity string) ([]TimelineGroup, error)
	BatchSoftDelete(userID string, ids []string) error
	BatchMove(userID string, ids []string, folderID *string) error
	BatchCopy(userID string, ids []string, folderID *string) ([]*model.File, error)
	FindPhotosMissingThumbnails() ([]*model.File, error)
	CountPhotosMissingThumbnailPreview() (int, error)
	Restore(id string) error
	BatchRestore(userID string, ids []string) error
	ListTrash(opts FileListOptions) ([]*model.File, string, int, error)
	TrashStats(userID string) (*TrashStatsResult, error)
	GetExpiredFiles(cutoff string, limit int) ([]ExpiredFile, error)
	PermanentDeleteByIDs(ids []string) error
	BatchPermanentDelete(userID string, ids []string) error
	UpdateSizeAndHash(id string, sizeBytes int64, sha256 string) error
	ListFilesByFolderID(folderID, cursor string, limit int) ([]*model.File, string, int, error)
	Rename(id, userID, newName string) error
	SoftDeleteByFolderIDs(userID string, folderIDs []string) (int64, error)
	ListTrashFiles(userID string, ids []string) ([]ExpiredFile, error)
	ListAllTrashFiles(userID string) ([]ExpiredFile, error)
	AdminFileBreakdown() (*AdminFileBreakdown, error)
	AdminFileBreakdownByUser(userID string) (*AdminFileBreakdown, error)
}

type FolderRepository interface {
	Create(userID, name string, parentID *string) (*model.Folder, error)
	FindByID(id string) (*model.Folder, error)
	ListByUser(userID string) ([]*model.Folder, error)
	ListTree(userID string) (*model.FolderTreeNode, error)
	UpdateName(id, name string) error
	Delete(id string) error
	FindByParentID(parentID string) ([]*model.Folder, error)
	UpdateParent(id string, parentID *string) error
	IsDescendant(id, potentialAncestor string) (bool, error)
	FindDescendantIDs(id string) ([]string, error)
	DeleteRecursive(folderID, userID string) (*DeleteRecursiveResult, error)
}

type ExifRepository interface {
	Create(exif *model.ExifData) error
	FindByFileID(fileID string) (*model.ExifData, error)
}

type ThumbnailRepository interface {
	Create(thumb *model.Thumbnail) error
	FindByFileIDAndSize(fileID string, size model.ThumbnailSize) (*model.Thumbnail, error)
	FindThumbnailRefsByFileID(fileID string) ([]ThumbnailRef, error)
	CountByFileID(fileID string) (int, error)
	SetS3Key(fileID string, size model.ThumbnailSize, s3Key string) error
	TotalSize() (int64, error)
	Breakdown() ([]ThumbnailBreakdown, error)
	BreakdownByUser(userID string) ([]ThumbnailBreakdown, error)
}

type GeoRepository interface {
	GetPoints(userID string, bounds GeoBounds) ([]GeoPoint, error)
}

type UploadJobRepository interface {
	Create(job *model.UploadJob) error
	Claim() (*model.UploadJob, error)
	FindByID(id string) (*model.UploadJob, error)
	FindByResumeToken(token string) (*model.UploadJob, error)
	UpdateProgress(id string, stage model.JobStage, progress float64) error
	Complete(id, fileID string) error
	Fail(id, errorMsg string) error
	Skip(id, reason, fileID string) error
	SetProcessing(id string) error
	SetStatus(id string, status model.JobStatus) error
	ListByBatch(batchID string) ([]*model.UploadJob, error)
	CountProcessing() (int, error)
	RecoverStuckJobs() (int64, error)
	ListActiveByUser(userID string) ([]*model.UploadJob, error)
	DeleteByID(id string) error
	ListAll(limit, offset int, statusFilter string) ([]*model.UploadJob, int, error)
	CountByStatus() (map[string]int, error)
	Requeue(id string) error
	CompleteChunked(id string, totalChunks int) error
}

type ChunkRepository interface {
	CreateChunkRecord(uploadID string, index int, size, offset int64, sha256hex, tempPath string) error
	GetStoredChunks(uploadID string) ([]int, error)
	GetStoredChunkCount(uploadID string) (int, error)
	GetChunkPath(uploadID string, index int) (string, error)
	FindMissingChunks(uploadID string, totalChunks int) ([]int, error)
	AssembleFile(uploadID string, totalChunks int, destPath string) (string, error)
	DeleteChunks(uploadID string) error
	DeleteAbandonedChunks(maxAgeHours int) (int64, error)
	CleanupOrphanedTempFiles(uploadID string)
	CleanupOldUploads(maxAgeHours int) ([]string, error)
}

type SettingRepository interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

type AlbumRepository interface {
	Create(userID, name string, description *string) (*model.Album, error)
	FindByID(id string) (*model.Album, error)
	FindByIDWithOwner(id string) (*model.AlbumWithDetails, error)
	ListByUser(userID string) ([]*model.Album, error)
	ListSharedWithUser(userID string) ([]*model.AlbumWithDetails, error)
	Update(id, name string, description *string) error
	Delete(id string) error
	CheckAccess(albumID, userID string) (permission string, found bool, err error)
	ListShares(albumID string) ([]model.SharedUser, error)
	ItemCount(albumID string) int64
	HasShares(albumID string) bool
}

type AlbumFileInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
	IsOwner bool   `json:"is_owner"`
}

type AlbumItemRepository interface {
	Add(albumID, fileID, addedByUserID string) (*model.AlbumItem, error)
	Remove(albumID, fileID string) error
	RemoveByID(id string) error
	ListFileIDs(albumID string, limit, offset int) ([]string, int64, error)
	FindByAlbumAndFile(albumID, fileID string) (*model.AlbumItem, error)
	HasSharedAccess(fileID, userID string) (bool, error)
	GetSharedPermission(fileID, userID string) (string, error)
	ListAlbumsByFile(fileID, userID string) ([]AlbumFileInfo, error)
}

type AlbumShareRepository interface {
	Add(albumID, sharedWithUserID, permission string) (*model.AlbumShare, error)
	Remove(id string) error
	FindByAlbumAndUser(albumID, userID string) (*model.AlbumShare, error)
}

type CommentRepository interface {
	Create(fileID, userID, content string) (*model.Comment, error)
	FindByFileID(fileID string) ([]*model.Comment, error)
	FindByID(id string) (*model.Comment, error)
	Update(id, userID, content string) error
	Delete(id, userID string) error
}

type ReactionRepository interface {
	Toggle(commentID, userID, emoji string) (added bool, err error)
	FindByCommentID(commentID, viewerUserID string) ([]model.ReactionGroup, error)
	Remove(commentID, userID, emoji string) error
}

type TagRepository interface {
	FindOrCreate(name string) (*model.Tag, error)
	Search(prefix string) ([]*model.Tag, error)
	AddToFile(fileID, tagID, userID string) error
	RemoveFromFile(fileID, tagID string) error
	FindByFileID(fileID string) ([]*model.Tag, error)
	ListWithCount(userID string) ([]model.TagWithCount, error)
}

type DocumentRepository interface {
	Create(fileID, content string) error
	FindByFileID(fileID string) (*model.Document, error)
	UpdateContent(fileID, content string) error
	Delete(fileID string) error
}

type FolderPasswordRepository interface {
	Create(folderID, passwordHash, passwordHint string, expiresAt time.Time) (*model.FolderPassword, error)
	FindByFolderID(folderID string) (*model.FolderPassword, error)
	DeleteByFolderID(folderID string) error
	DeleteExpired() (int64, error)
}

type FolderShareRepository interface {
	Create(share *model.FolderShare) error
	FindByID(id string) (*model.FolderShare, error)
	FindByToken(token string) (*model.FolderShare, error)
	ListByFolder(folderID string) ([]*model.FolderShare, error)
	Update(id string, permissions model.SharePermission, includeSubdirs bool, uploadLimitBytes *int64, expiresAt *time.Time, hasPassword bool, passwordHash *string) error
	Delete(id string) error
}

type ShareUploadRepository interface {
	Create(shareID, fileID string, sizeBytes int64) (*model.ShareUpload, error)
	SumByShareID(shareID string) (int64, error)
	ListByShareID(shareID string) ([]*model.ShareUpload, error)
}

type SystemEventsRepository interface {
	Create(event *model.SystemEvent) error
	List(limit, offset int, eventType, severity, dateFrom, dateTo string) ([]model.SystemEvent, int, error)
	EventCounts() (map[string]int, error)
	PurgeOlderThan(age time.Duration) (int64, error)
}
