package worker

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

const pollInterval = 1 * time.Second

type s3Task struct {
	UserID     string
	Filename   string
	DestPath   string
	FileID     string
	Thumbnails []*model.Thumbnail
}

type Pool struct {
	cfg              *config.Config
	fileStore        *store.FileStore
	exifStore        *store.ExifStore
	thumbnailStore   *store.ThumbnailStore
	exifService      *service.ExifService
	thumbnailService *service.ThumbnailService
	storageService   *service.StorageService
	uploadJobStore   *store.UploadJobStore
	chunkStore       *store.ChunkStore
	eventRecorder    *service.EventRecorder
	totalWorkers     int
	wg               sync.WaitGroup
	stopCh           chan struct{}
	jobCh            chan *model.UploadJob
	s3JobCh          chan *s3Task
	s3Workers        int
	subscribers      map[string]map[string]chan *model.UploadJob
	subMu            sync.RWMutex
	userSubscribers  map[string]map[string]chan *model.UploadJob
	userSubMu        sync.RWMutex
	processingJobs   map[string]*model.UploadJob
	processingMu     sync.RWMutex
	notifyCh         chan struct{}
	completedTotal   atomic.Int64
	failedTotal      atomic.Int64
	skippedTotal     atomic.Int64
}

type PoolStats struct {
	QueueLength    int       `json:"queue_length"`
	ActiveWorkers  int       `json:"active_workers"`
	TotalWorkers   int       `json:"total_workers"`
	ProcessingJobs []JobInfo `json:"processing_jobs"`
	CompletedTotal int64     `json:"completed_total"`
	FailedTotal    int64     `json:"failed_total"`
	SkippedTotal   int64     `json:"skipped_total"`
}

type JobInfo struct {
	JobID    string  `json:"job_id"`
	Filename string  `json:"filename"`
	Status   string  `json:"status"`
	Stage    string  `json:"stage,omitempty"`
	Progress float64 `json:"progress"`
}

func NewPool(cfg *config.Config, fileStore *store.FileStore, exifStore *store.ExifStore, thumbnailStore *store.ThumbnailStore, storageService *service.StorageService, uploadJobStore *store.UploadJobStore, chunkStore *store.ChunkStore, eventRecorder *service.EventRecorder) *Pool {
	workers := cfg.Upload.ConcurrentWorkers
	if workers <= 0 {
		workers = 4
	}
	p := &Pool{
		cfg:              cfg,
		fileStore:        fileStore,
		exifStore:        exifStore,
		thumbnailStore:   thumbnailStore,
		exifService:      service.NewExifService(),
		thumbnailService: service.NewThumbnailService(cfg.ThumbnailsDir()),
		storageService:   storageService,
		uploadJobStore:   uploadJobStore,
		chunkStore:       chunkStore,
		eventRecorder:    eventRecorder,
		totalWorkers:     workers,
		stopCh:           make(chan struct{}),
		jobCh:            make(chan *model.UploadJob, workers*2),
		s3JobCh:          make(chan *s3Task, workers*4),
		s3Workers:        2,
		subscribers:      make(map[string]map[string]chan *model.UploadJob),
		userSubscribers:  make(map[string]map[string]chan *model.UploadJob),
		processingJobs:   make(map[string]*model.UploadJob),
		notifyCh:         make(chan struct{}, 1),
	}

	p.recoverStuckJobs()

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	for i := 0; i < p.s3Workers; i++ {
		p.wg.Add(1)
		go p.s3Worker(i)
	}

	p.wg.Add(1)
	go p.claimer()

	return p
}

func (p *Pool) recoverStuckJobs() {
	processingCount, err := p.uploadJobStore.CountProcessing()
	if err != nil {
		slog.Warn("failed to count stuck jobs", "error", err)
		return
	}

	if processingCount > 0 {
		slog.Warn("found stuck processing jobs, recovering", "count", processingCount)
		recovered, err := p.uploadJobStore.RecoverStuckJobs()
		if err != nil {
			slog.Warn("failed to recover stuck jobs", "error", err)
		} else if recovered > 0 {
			slog.Info("recovered stuck upload jobs", "count", recovered)
		}
	}
}

func (p *Pool) NotifyJobsAvailable() {
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

func (p *Pool) claimer() {
	defer p.wg.Done()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-p.notifyCh:
		case <-ticker.C:
		}

		if len(p.jobCh) >= cap(p.jobCh) {
			continue
		}

		job, err := p.uploadJobStore.Claim()
		if err != nil {
			slog.Warn("claimer claim failed", "error", err)
			continue
		}
		if job == nil {
			continue
		}
		p.jobCh <- job
	}
}

func (p *Pool) s3Worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCh:
			return
		case task, ok := <-p.s3JobCh:
			if !ok {
				return
			}
			if p.storageService == nil {
				continue
			}
			if task.DestPath != "" {
				if err := p.storageService.PutOriginals(task.UserID, task.Filename, task.DestPath); err != nil {
					slog.Warn("s3 upload failed for original", "file_id", task.FileID, "error", err)
					p.eventRecorder.Error("s3_upload_error", "S3 upload failed for original", map[string]interface{}{
						"file_id": task.FileID,
						"error":   err.Error(),
					})
				} else {
					if err := os.Remove(task.DestPath); err != nil {
						slog.Warn("failed to remove local original after s3 upload", "file_id", task.FileID, "error", err)
					}
				}
			}
			for _, t := range task.Thumbnails {
				if _, err := os.Stat(t.LocalPath); os.IsNotExist(err) {
					slog.Warn("thumbnail missing before s3 upload, skipping", "file_id", task.FileID, "size", t.Size, "path", t.LocalPath)
					continue
				}
				format := t.Format
				if format == "" {
					format = "jpg"
				}
				if err := p.storageService.PutThumbnail(task.FileID, string(t.Size), format, t.LocalPath); err != nil {
					slog.Warn("s3 upload failed for thumbnail", "file_id", task.FileID, "size", t.Size, "error", err)
					p.eventRecorder.Error("s3_upload_error", "S3 upload failed for thumbnail", map[string]interface{}{
						"file_id": task.FileID,
						"size":    string(t.Size),
						"error":   err.Error(),
					})
				} else {
					s3Key := fmt.Sprintf("thumbnails/%s/%s.%s", task.FileID, t.Size, format)
					if err := p.thumbnailStore.SetS3Key(task.FileID, t.Size, s3Key); err != nil {
						slog.Warn("failed to persist s3_key for thumbnail", "file_id", task.FileID, "size", t.Size, "error", err)
					}
					if t.Size == model.ThumbSizeVideoProxy {
						if err := os.Remove(t.LocalPath); err != nil {
							slog.Warn("failed to remove local video proxy after s3 upload", "file_id", task.FileID, "error", err)
						}
					}
				}
			}
		}
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCh:
			return
		case job, ok := <-p.jobCh:
			if !ok {
				return
			}

			if _, err := os.Stat(job.TempPath); os.IsNotExist(err) {
				p.uploadJobStore.Fail(job.ID, "temp_file_missing")
				p.notifySubscribers(job)
				p.failedTotal.Add(1)
				slog.Warn("temp file missing, marking job as failed", "job_id", job.ID, "temp_path", job.TempPath)
				p.eventRecorder.Error("upload_error", "Upload failed: temp file missing", map[string]interface{}{
					"job_id":    job.ID,
					"filename":  job.Filename,
					"temp_path": job.TempPath,
				})
				continue
			}

			p.processingMu.Lock()
			p.processingJobs[job.ID] = job
			p.processingMu.Unlock()

			p.processJob(job)

			p.processingMu.Lock()
			delete(p.processingJobs, job.ID)
			p.processingMu.Unlock()
		}
	}
}

func (p *Pool) processJob(job *model.UploadJob) {
	if job.UploadMode == model.UploadModeChunked {
		stored, err := p.chunkStore.GetStoredChunkCount(job.ID)
		if err != nil || job.TotalChunks == nil || stored < *job.TotalChunks {
			if time.Since(job.CreatedAt) > 1*time.Hour {
				p.uploadJobStore.Fail(job.ID, "upload_expired")
				p.notifySubscribers(job)
				p.failedTotal.Add(1)
				return
			}
			p.uploadJobStore.SetStatus(job.ID, model.JobStatusQueued)
			return
		}
		p.processChunkedJob(job)
		return
	}

	stage := model.JobStageHashing
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.1)
	p.notifySubscribers(job)

	f, err := os.Open(job.TempPath)
	if err != nil {
		p.uploadJobStore.Fail(job.ID, "cannot_open_temp_file")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		return
	}

	if job.Reason != nil && *job.Reason == "reconcile" {
		p.processReconcileJob(job, f)
		return
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		f.Close()
		os.Remove(job.TempPath)
		p.uploadJobStore.Fail(job.ID, "hash_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: hash computation error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
		})
		return
	}
	sha256Hash := fmt.Sprintf("%x", hasher.Sum(nil))
	f.Seek(0, 0)

	mimeType := detectMimeTypeFromFile(f, job.Filename)
	f.Seek(0, 0)

	mediaType := detectMediaType(mimeType)

	stage = model.JobStageDedup
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.2)
	p.notifySubscribers(job)

	if !job.SkipNameSizeDedup && p.cfg.Media.AutoOrganize && mediaType == model.MediaTypePhoto {
		existing, _ := p.fileStore.FindByNameAndSize(job.UserID, job.Filename, job.SizeBytes)
		if existing != nil {
			f.Close()
			os.Remove(job.TempPath)
			p.uploadJobStore.Skip(job.ID, "duplicate_name_size", existing.ID)
			p.notifySubscribers(job)
			p.skippedTotal.Add(1)
			p.eventRecorder.Info("upload_skipped", "Upload skipped: duplicate name+size", map[string]interface{}{
				"filename": job.Filename,
				"reason":   "duplicate_name_size",
			})
			return
		}
	}

	if !job.SkipNameSizeDedup {
		existingHash, _ := p.fileStore.FindBySHA256(job.UserID, sha256Hash)
		if existingHash != nil {
			f.Close()
			os.Remove(job.TempPath)
			p.uploadJobStore.Skip(job.ID, "duplicate_content", existingHash.ID)
			p.notifySubscribers(job)
			p.skippedTotal.Add(1)
			p.eventRecorder.Info("upload_skipped", "Upload skipped: duplicate content", map[string]interface{}{
				"filename": job.Filename,
				"reason":   "duplicate_content",
			})
			return
		}
	}

	p.finishJob(job, f, mimeType, mediaType, sha256Hash)
}

func (p *Pool) processChunkedJob(job *model.UploadJob) {
	stage := model.JobStage("assembling")
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.0)
	p.notifySubscribers(job)

	tempDir := store.ChunkTempDir(p.cfg.OriginalsDir())
	destPath := filepath.Join(tempDir, "assembled-"+job.ID)

	sha256Hash, err := p.chunkStore.AssembleFile(job.ID, *job.TotalChunks, destPath)
	if err != nil {
		p.uploadJobStore.Fail(job.ID, "assembly_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: chunk assembly error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
			"error":    err.Error(),
		})
		return
	}

	p.chunkStore.DeleteChunks(job.ID)
	job.TempPath = destPath

	stage = model.JobStageDedup
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.2)
	p.notifySubscribers(job)

	f, err := os.Open(destPath)
	if err != nil {
		os.Remove(destPath)
		p.uploadJobStore.Fail(job.ID, "cannot_open_assembled")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		return
	}

	mimeType := detectMimeTypeFromFile(f, job.Filename)
	mediaType := detectMediaType(mimeType)
	f.Seek(0, 0)

	if !job.SkipNameSizeDedup && p.cfg.Media.AutoOrganize && mediaType == model.MediaTypePhoto {
		existing, _ := p.fileStore.FindByNameAndSize(job.UserID, job.Filename, job.SizeBytes)
		if existing != nil {
			os.Remove(destPath)
			p.uploadJobStore.Skip(job.ID, "duplicate_name_size", existing.ID)
			p.notifySubscribers(job)
			p.skippedTotal.Add(1)
			return
		}
	}

	if !job.SkipNameSizeDedup {
		existingHash, _ := p.fileStore.FindBySHA256(job.UserID, sha256Hash)
		if existingHash != nil {
			os.Remove(destPath)
			p.uploadJobStore.Skip(job.ID, "duplicate_content", existingHash.ID)
			p.notifySubscribers(job)
			p.skippedTotal.Add(1)
			return
		}
	}

	p.finishJob(job, f, mimeType, mediaType, sha256Hash)
}

func (p *Pool) finishJob(job *model.UploadJob, f *os.File, mimeType string, mediaType model.MediaType, sha256Hash string) {
	defer f.Close()
	f.Seek(0, 0)

	var exifData *model.ExifData
	var now = time.Now().UTC()

	if mediaType != model.MediaTypeFile {
		stage := model.JobStageExif
		job.Stage = &stage
		p.uploadJobStore.UpdateProgress(job.ID, stage, 0.3)
		p.notifySubscribers(job)
		exifData, _ = p.exifService.Extract(job.TempPath)
	} else {
		stage := model.JobStageStoring
		job.Stage = &stage
		p.uploadJobStore.UpdateProgress(job.ID, stage, 0.3)
		p.notifySubscribers(job)
	}

	yearMonth := now.Format("2006/01")

	if exifData != nil && exifData.DateTaken != nil {
		if t, err := time.Parse("2006:01:02 15:04:05", *exifData.DateTaken); err == nil {
			yearMonth = t.Format("2006/01")
			iso := t.Format(time.RFC3339)
			exifData.DateTaken = &iso
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", *exifData.DateTaken); err == nil {
			yearMonth = t.Format("2006/01")
		}
	}

	stage := model.JobStageStoring
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.5)
	p.notifySubscribers(job)

	pathPrefix := ""
	if mediaType == model.MediaTypeFile {
		pathPrefix = "files/"
	}
	destSubdir := filepath.Join(job.UserID, filepath.Join(pathPrefix, yearMonth))
	destDir := filepath.Join(p.cfg.OriginalsDir(), destSubdir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		os.Remove(job.TempPath)
		p.uploadJobStore.Fail(job.ID, "storage_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: storage error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
		})
		return
	}

	storedFilename := fmt.Sprintf("%s_%s", uuid.New().String(), job.Filename)
	destPath := filepath.Join(destDir, storedFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		os.Remove(job.TempPath)
		p.uploadJobStore.Fail(job.ID, "write_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: write error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
		})
		return
	}

	if _, err := io.Copy(destFile, f); err != nil {
		destFile.Close()
		os.Remove(job.TempPath)
		os.Remove(destPath)
		p.uploadJobStore.Fail(job.ID, "write_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: write error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
		})
		return
	}
	destFile.Close()

	var takenAt *string
	if exifData != nil && exifData.DateTaken != nil {
		takenAt = exifData.DateTaken
	} else {
		iso := now.Format(time.RFC3339)
		takenAt = &iso
	}

	filePath := yearMonth
	fileNamePath := filepath.Join(yearMonth, storedFilename)
	if mediaType == model.MediaTypeFile {
		filePath = filepath.Join("files", yearMonth)
		fileNamePath = filepath.Join("files", yearMonth, storedFilename)
	}

	fileRecord := &model.File{
		UserID:       job.UserID,
		Filename:     fileNamePath,
		OriginalName: job.Filename,
		Path:         filePath,
		SizeBytes:    job.SizeBytes,
		MimeType:     mimeType,
		SHA256:       sha256Hash,
		MediaType:    mediaType,
		TakenAt:      takenAt,
		FolderID:     job.FolderID,
	}

	if err := p.fileStore.Create(fileRecord); err != nil {
		os.Remove(job.TempPath)
		os.Remove(destPath)
		p.uploadJobStore.Fail(job.ID, "db_error")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		p.eventRecorder.Error("upload_error", "Upload failed: database error", map[string]interface{}{
			"job_id":   job.ID,
			"filename": job.Filename,
		})
		return
	}

	if exifData != nil {
		exifData.FileID = fileRecord.ID
		if err := p.exifStore.Create(exifData); err != nil {
			slog.Warn("failed to store exif", "file_id", fileRecord.ID, "error", err)
		}
	}

	var thumbs []*model.Thumbnail

	if mediaType != model.MediaTypeFile {
		stage = model.JobStageThumbnails
		job.Stage = &stage
		p.uploadJobStore.UpdateProgress(job.ID, stage, 0.8)
		p.notifySubscribers(job)
		thumbs, err = p.thumbnailService.GenerateAll(fileRecord.ID, destPath, mimeType)
		if err != nil {
			slog.Warn("thumbnail generation failed", "file_id", fileRecord.ID, "error", err)
			p.eventRecorder.Error("upload_error", "Upload failed: thumbnail generation error", map[string]interface{}{
				"file_id": fileRecord.ID,
				"error":   err.Error(),
			})
		} else {
			for _, t := range thumbs {
				if err := p.thumbnailStore.Create(t); err != nil {
					slog.Warn("failed to store thumbnail", "file_id", fileRecord.ID, "size", t.Size, "error", err)
				}
			}
		}
	}

	if p.storageService != nil {
		task := &s3Task{
			UserID:     job.UserID,
			Filename:   fileRecord.Filename,
			DestPath:   destPath,
			FileID:     fileRecord.ID,
			Thumbnails: thumbs,
		}
		select {
		case p.s3JobCh <- task:
		default:
			slog.Warn("s3 upload queue full, keeping local copy", "file_id", fileRecord.ID)
		}
	}

	os.Remove(job.TempPath)
	p.uploadJobStore.Complete(job.ID, fileRecord.ID)
	job.FileID = &fileRecord.ID
	job.Status = model.JobStatusCompleted
	job.Progress = 1.0
	p.notifySubscribers(job)
	p.completedTotal.Add(1)

	slog.Info("processed upload", "job_id", job.ID, "file_id", fileRecord.ID, "name", job.Filename)
}

func (p *Pool) processReconcileJob(job *model.UploadJob, f *os.File) {
	defer f.Close()

	if job.FileID == nil {
		p.uploadJobStore.Fail(job.ID, "reconcile_missing_file_id")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		return
	}

	fileRecord, err := p.fileStore.FindByID(*job.FileID)
	if err != nil || fileRecord == nil {
		p.uploadJobStore.Fail(job.ID, "reconcile_file_not_found")
		p.notifySubscribers(job)
		p.failedTotal.Add(1)
		return
	}

	f.Seek(0, 0)
	mimeType := detectMimeTypeFromFile(f, job.Filename)
	mediaType := detectMediaType(mimeType)

	if mediaType == model.MediaTypeFile {
		p.uploadJobStore.Complete(job.ID, fileRecord.ID)
		job.FileID = &fileRecord.ID
		job.Status = model.JobStatusCompleted
		job.Progress = 1.0
		p.notifySubscribers(job)
		p.completedTotal.Add(1)
		return
	}

	stage := model.JobStageThumbnails
	job.Stage = &stage
	p.uploadJobStore.UpdateProgress(job.ID, stage, 0.5)
	p.notifySubscribers(job)

	thumbs, err := p.thumbnailService.GenerateAll(fileRecord.ID, job.TempPath, mimeType)
	if err != nil {
		slog.Warn("reconcile thumbnail generation failed", "file_id", fileRecord.ID, "error", err)
	} else {
		for _, t := range thumbs {
			if err := p.thumbnailStore.Create(t); err != nil {
				slog.Warn("reconcile failed to store thumbnail", "file_id", fileRecord.ID, "size", t.Size, "error", err)
			}
		}
	}

	if p.storageService != nil && len(thumbs) > 0 {
		task := &s3Task{
			UserID:     job.UserID,
			Filename:   fileRecord.Filename,
			DestPath:   "",
			FileID:     fileRecord.ID,
			Thumbnails: thumbs,
		}
		select {
		case p.s3JobCh <- task:
		default:
			slog.Warn("s3 upload queue full, skipping reconcile thumb upload", "file_id", fileRecord.ID)
		}
	}

	p.uploadJobStore.Complete(job.ID, fileRecord.ID)
	job.FileID = &fileRecord.ID
	job.Status = model.JobStatusCompleted
	job.Progress = 1.0
	p.notifySubscribers(job)
	p.completedTotal.Add(1)

	slog.Info("processed reconcile", "job_id", job.ID, "file_id", fileRecord.ID, "name", job.Filename)
}

func (p *Pool) Shutdown() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *Pool) StartReconciler(interval time.Duration) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopCh:
				return
			case <-ticker.C:
				p.RunReconciliation()
			}
		}
	}()
}

func (p *Pool) RunReconciliation() map[string]interface{} {
	originalsDir := p.cfg.OriginalsDir()
	missingAll := 0
	missingPreview := 0
	created := 0
	batchID := "reconcile-" + time.Now().UTC().Format(time.RFC3339)

	files, err := p.fileStore.FindPhotosMissingThumbnails()
	if err != nil {
		slog.Warn("reconciler failed to find missing thumbnails", "error", err)
		p.eventRecorder.Error("reconciliation_error", "Reconciliation failed to find missing thumbnails", map[string]interface{}{
			"error": err.Error(),
		})
		return map[string]interface{}{"created": 0, "details": map[string]int{}}
	}

	for _, f := range files {
		thumbCount, _ := p.thumbnailStore.CountByFileID(f.ID)
		if thumbCount == 0 {
			missingAll++
		} else {
			missingPreview++
		}

		originalPath := filepath.Join(originalsDir, f.UserID, f.Filename)
		if _, err := os.Stat(originalPath); os.IsNotExist(err) {
			continue
		}

		job := &model.UploadJob{
			BatchID:           batchID,
			UserID:            f.UserID,
			Filename:          f.OriginalName,
			SizeBytes:         f.SizeBytes,
			TempPath:          originalPath,
			SkipNameSizeDedup: true,
			Status:            model.JobStatusQueued,
			Reason:            &[]string{"reconcile"}[0],
			FileID:            &f.ID,
		}
		if err := p.uploadJobStore.Create(job); err != nil {
			slog.Warn("reconciler failed to create job", "file_id", f.ID, "error", err)
			continue
		}
		created++
	}

	if created > 0 {
		slog.Info("reconciler created jobs", "created", created, "missing_all", missingAll, "missing_preview", missingPreview)
		p.eventRecorder.Info("reconciliation_run", "Thumbnail reconciliation completed", map[string]interface{}{
			"created":         created,
			"missing_all":     missingAll,
			"missing_preview": missingPreview,
		})
		p.NotifyJobsAvailable()
	}

	return map[string]interface{}{
		"created": created,
		"details": map[string]int{
			"missing_all_thumbnails": missingAll,
			"missing_preview_only":   missingPreview,
		},
	}
}

func (p *Pool) Subscribe(batchID string, listenerID string) chan *model.UploadJob {
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if p.subscribers[batchID] == nil {
		p.subscribers[batchID] = make(map[string]chan *model.UploadJob)
	}
	ch := make(chan *model.UploadJob, 20)
	p.subscribers[batchID][listenerID] = ch
	return ch
}

func (p *Pool) Unsubscribe(batchID, listenerID string) {
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if sub, ok := p.subscribers[batchID]; ok {
		if ch, ok := sub[listenerID]; ok {
			close(ch)
			delete(sub, listenerID)
		}
	}
}

func (p *Pool) SubscribeUser(userID string, listenerID string) chan *model.UploadJob {
	p.userSubMu.Lock()
	defer p.userSubMu.Unlock()
	if p.userSubscribers[userID] == nil {
		p.userSubscribers[userID] = make(map[string]chan *model.UploadJob)
	}
	ch := make(chan *model.UploadJob, 20)
	p.userSubscribers[userID][listenerID] = ch
	return ch
}

func (p *Pool) UnsubscribeUser(userID, listenerID string) {
	p.userSubMu.Lock()
	defer p.userSubMu.Unlock()
	if sub, ok := p.userSubscribers[userID]; ok {
		if ch, ok := sub[listenerID]; ok {
			close(ch)
			delete(sub, listenerID)
		}
	}
}

func (p *Pool) notifySubscribers(job *model.UploadJob) {
	p.subMu.RLock()
	defer p.subMu.RUnlock()
	if sub, ok := p.subscribers[job.BatchID]; ok {
		for _, ch := range sub {
			select {
			case ch <- job:
			default:
			}
		}
	}

	p.userSubMu.RLock()
	defer p.userSubMu.RUnlock()
	if sub, ok := p.userSubscribers[job.UserID]; ok {
		for _, ch := range sub {
			select {
			case ch <- job:
			default:
			}
		}
	}
}

func (p *Pool) Stats() PoolStats {
	p.processingMu.RLock()
	processing := make([]JobInfo, 0, len(p.processingJobs))
	for _, j := range p.processingJobs {
		stage := ""
		if j.Stage != nil {
			stage = string(*j.Stage)
		}
		processing = append(processing, JobInfo{
			JobID:    j.ID,
			Filename: j.Filename,
			Status:   string(j.Status),
			Stage:    stage,
			Progress: j.Progress,
		})
	}
	p.processingMu.RUnlock()

	return PoolStats{
		ActiveWorkers:  len(processing),
		TotalWorkers:   p.totalWorkers,
		ProcessingJobs: processing,
		CompletedTotal: p.completedTotal.Load(),
		FailedTotal:    p.failedTotal.Load(),
		SkippedTotal:   p.skippedTotal.Load(),
	}
}

func detectMimeTypeFromFile(f *os.File, filename string) string {
	head := make([]byte, 512)
	n, _ := f.Read(head)
	head = head[:n]

	if n >= 4 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF {
		return "image/jpeg"
	}
	if n >= 4 && head[0] == 0x89 && head[1] == 0x50 && head[2] == 0x4E && head[3] == 0x47 {
		return "image/png"
	}
	if n >= 4 && head[0] == 0x52 && head[1] == 0x49 && head[2] == 0x46 && head[3] == 0x46 {
		return "image/webp"
	}
	if n >= 8 && head[4] == 0x66 && head[5] == 0x74 && head[6] == 0x79 && head[7] == 0x70 {
		if n >= 12 {
			brand := string(head[8:12])
			if brand == "heic" || brand == "heix" || brand == "hevc" || brand == "hevx" {
				return "image/heic"
			}
		}
	}
	if n >= 12 && string(head[4:8]) == "ftyp" && string(head[8:12]) == "avif" {
		return "image/avif"
	}
	if n >= 4 && ((head[0] == 0x49 && head[1] == 0x49 && head[2] == 0x2A && head[3] == 0x00) ||
		(head[0] == 0x4D && head[1] == 0x4D && head[2] == 0x00 && head[3] == 0x2A)) {
		return "image/tiff"
	}
	if n >= 4 && head[0] == 0x00 && head[1] == 0x00 && head[2] == 0x00 && (head[3]&0xF0) == 0x10 {
		return "video/mp4"
	}
	if n >= 4 && head[0] == 0x1A && head[1] == 0x45 && head[2] == 0xDF && head[3] == 0xA3 {
		return "video/x-matroska"
	}
	if n >= 4 && head[0] == 0x25 && head[1] == 0x50 && head[2] == 0x44 && head[3] == 0x46 {
		return "application/pdf"
	}
	if n >= 4 && head[0] == 0x50 && head[1] == 0x4B && head[2] == 0x03 && head[3] == 0x04 {
		return "application/zip"
	}
	if n >= 10 && string(head[4:8]) == "ftyp" && head[8] == 'q' && head[9] == 't' {
		return "video/quicktime"
	}

	return detectMimeTypeByExtension(filename)
}

func detectMimeTypeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeMap := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".png": "image/png", ".webp": "image/webp",
		".heic": "image/heic", ".avif": "image/avif",
		".tiff": "image/tiff", ".tif": "image/tiff",
		".cr2": "image/x-canon-cr2", ".nef": "image/x-nikon-nef",
		".arw": "image/x-sony-arw", ".dng": "image/x-adobe-dng",
		".mp4": "video/mp4", ".mov": "video/quicktime",
		".avi": "video/x-msvideo", ".mkv": "video/x-matroska",
		".hevc": "video/hevc", ".pdf": "application/pdf",
		".zip": "application/zip",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func detectMediaType(mimeType string) model.MediaType {
	if strings.HasPrefix(mimeType, "image/") {
		return model.MediaTypePhoto
	}
	if strings.HasPrefix(mimeType, "video/") {
		return model.MediaTypeVideo
	}
	return model.MediaTypeFile
}
