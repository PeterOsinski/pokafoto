package service

import (
	"log/slog"
	"sync/atomic"

	"github.com/drive/drive/internal/store"
)

type S3ThumbItem struct {
	Size   string
	Format string
}

type S3DeleteTask struct {
	FileID   string
	UserID   string
	Filename string
	Thumbs   []S3ThumbItem
}

type S3DeletionPool struct {
	taskCh       chan *S3DeleteTask
	storage      StorageProvider
	thumbnailStore store.ThumbnailRepository
	stopCh       chan struct{}
	pendingCount atomic.Int64
}

const s3DeletionWorkers = 2

func NewS3DeletionPool(storage StorageProvider, thumbnailStore store.ThumbnailRepository) *S3DeletionPool {
	p := &S3DeletionPool{
		taskCh:         make(chan *S3DeleteTask, 256),
		storage:        storage,
		thumbnailStore: thumbnailStore,
		stopCh:         make(chan struct{}),
	}
	for i := 0; i < s3DeletionWorkers; i++ {
		go p.worker(i)
	}
	return p
}

func (p *S3DeletionPool) worker(id int) {
	for {
		select {
		case <-p.stopCh:
			return
		case task := <-p.taskCh:
			if p.storage == nil {
				continue
			}
			if err := p.storage.DeleteOriginal(task.UserID, task.Filename); err != nil {
				slog.Warn("s3 delete original failed", "file_id", task.FileID, "error", err)
			}
			for _, t := range task.Thumbs {
				if err := p.storage.DeleteThumbnail(task.FileID, t.Size, t.Format); err != nil {
					slog.Warn("s3 delete thumbnail failed", "file_id", task.FileID, "size", t.Size, "error", err)
				}
			}
			p.pendingCount.Add(-1)
		}
	}
}

func (p *S3DeletionPool) Enqueue(task *S3DeleteTask) {
	p.pendingCount.Add(1)
	select {
	case p.taskCh <- task:
	default:
		slog.Warn("s3 deletion queue full, dropping task", "file_id", task.FileID)
		p.pendingCount.Add(-1)
	}
}

func (p *S3DeletionPool) PendingCount() int64 {
	return p.pendingCount.Load()
}

func (p *S3DeletionPool) Shutdown() {
	close(p.stopCh)
}
