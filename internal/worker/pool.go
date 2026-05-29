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

type JobStatus string

const (
	JobQueued     JobStatus = "queued"
	JobProcessing JobStatus = "processing"
	JobCompleted  JobStatus = "completed"
	JobSkipped    JobStatus = "skipped"
	JobFailed     JobStatus = "failed"
)

type JobStage string

const (
	StageHashing      JobStage = "hashing"
	StageDedup        JobStage = "dedup"
	StageStoring      JobStage = "storing"
	StageExif         JobStage = "exif"
	StageThumbnails   JobStage = "thumbnails"
)

type UploadJob struct {
	JobID             string
	BatchID           string
	UserID            string
	Filename          string
	OriginalName      string
	Size              int64
	TempPath          string
	FolderID          *string
	SkipNameSizeDedup bool
	Status            JobStatus
	Stage             JobStage
	Progress          float64
	Error             string
	FileID            string
	Reason            string

	mu sync.Mutex
}

func (j *UploadJob) SetProgress(stage JobStage, progress float64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Stage = stage
	j.Progress = progress
}

func (j *UploadJob) SetCompleted(fileID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobCompleted
	j.FileID = fileID
	j.Progress = 1.0
}

func (j *UploadJob) SetSkipped(reason, existingFileID string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobSkipped
	j.Reason = reason
	j.FileID = existingFileID
	j.Progress = 1.0
}

func (j *UploadJob) SetFailed(errStr string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobFailed
	j.Error = errStr
}

type Pool struct {
	cfg              *config.Config
	fileStore        *store.FileStore
	exifStore        *store.ExifStore
	thumbnailStore   *store.ThumbnailStore
	exifService      *service.ExifService
	thumbnailService *service.ThumbnailService
	storageService   *service.StorageService
	jobChan          chan *UploadJob
	totalWorkers     int
	wg               sync.WaitGroup
	batches          map[string]*Batch
	batchesMu        sync.RWMutex
	subscribers      map[string]map[string]chan *UploadJob
	subMu            sync.RWMutex
	userSubscribers  map[string]map[string]chan *UploadJob
	userSubMu        sync.RWMutex
	processingJobs   map[string]*UploadJob
	processingMu     sync.RWMutex
	completedTotal   atomic.Int64
	failedTotal      atomic.Int64
	skippedTotal     atomic.Int64
}

type Batch struct {
	ID    string
	Jobs  []*UploadJob
	Total int
}

type UploadInput struct {
	BatchID string
	UserID  string
	Job     *UploadJob
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

func NewPool(cfg *config.Config, fileStore *store.FileStore, exifStore *store.ExifStore, thumbnailStore *store.ThumbnailStore, storageService *service.StorageService) *Pool {
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
		jobChan:          make(chan *UploadJob, workers*2),
		totalWorkers:     workers,
		batches:          make(map[string]*Batch),
		subscribers:      make(map[string]map[string]chan *UploadJob),
		userSubscribers:  make(map[string]map[string]chan *UploadJob),
		processingJobs:   make(map[string]*UploadJob),
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return p
}

func (p *Pool) Enqueue(batchID, userID string, originalName string, size int64, tempPath string, folderID *string, skipNameSizeDedup bool) *UploadJob {
	job := &UploadJob{
		JobID:             uuid.New().String(),
		BatchID:           batchID,
		UserID:            userID,
		Filename:          originalName,
		OriginalName:      originalName,
		Size:              size,
		TempPath:          tempPath,
		FolderID:          folderID,
		SkipNameSizeDedup: skipNameSizeDedup,
		Status:            JobQueued,
	}

	p.batchesMu.Lock()
	b, ok := p.batches[batchID]
	if !ok {
		b = &Batch{ID: batchID}
		p.batches[batchID] = b
	}
	b.Jobs = append(b.Jobs, job)
	b.Total++
	p.batchesMu.Unlock()

	p.jobChan <- job
	p.notifySubscribers(job)
	return job
}

func (p *Pool) GetBatch(batchID string) *Batch {
	p.batchesMu.RLock()
	defer p.batchesMu.RUnlock()
	return p.batches[batchID]
}

func (p *Pool) Subscribe(batchID string, listenerID string) chan *UploadJob {
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if p.subscribers[batchID] == nil {
		p.subscribers[batchID] = make(map[string]chan *UploadJob)
	}
	ch := make(chan *UploadJob, 20)
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

func (p *Pool) notifySubscribers(job *UploadJob) {
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

func (p *Pool) SubscribeUser(userID string, listenerID string) chan *UploadJob {
	p.userSubMu.Lock()
	defer p.userSubMu.Unlock()
	if p.userSubscribers[userID] == nil {
		p.userSubscribers[userID] = make(map[string]chan *UploadJob)
	}
	ch := make(chan *UploadJob, 20)
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

func (p *Pool) Stats() PoolStats {
	queueLen := len(p.jobChan)

	p.processingMu.RLock()
	processing := make([]JobInfo, 0, len(p.processingJobs))
	for _, j := range p.processingJobs {
		j.mu.Lock()
		processing = append(processing, JobInfo{
			JobID:    j.JobID,
			Filename: j.Filename,
			Status:   string(j.Status),
			Stage:    string(j.Stage),
			Progress: j.Progress,
		})
		j.mu.Unlock()
	}
	p.processingMu.RUnlock()

	return PoolStats{
		QueueLength:    queueLen,
		ActiveWorkers:  len(processing),
		TotalWorkers:   p.totalWorkers,
		ProcessingJobs: processing,
		CompletedTotal: p.completedTotal.Load(),
		FailedTotal:    p.failedTotal.Load(),
		SkippedTotal:   p.skippedTotal.Load(),
	}
}

func (p *Pool) Shutdown() {
	close(p.jobChan)
	p.wg.Wait()
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for job := range p.jobChan {
		func() {
			defer func() {
				if r := recover(); r != nil {
					job.SetFailed(fmt.Sprintf("worker panic: %v", r))
					p.notifySubscribers(job)
					slog.Error("worker panic recovered", "worker_id", id, "job_id", job.JobID, "panic", r)
				}
			}()
			p.processJob(job)
		}()
	}
}

func (p *Pool) processJob(job *UploadJob) {
	job.Status = JobProcessing
	p.notifySubscribers(job)

	p.processingMu.Lock()
	p.processingJobs[job.JobID] = job
	p.processingMu.Unlock()

	defer func() {
		p.processingMu.Lock()
		delete(p.processingJobs, job.JobID)
		p.processingMu.Unlock()
		switch job.Status {
		case JobCompleted:
			p.completedTotal.Add(1)
		case JobSkipped:
			p.skippedTotal.Add(1)
		case JobFailed:
			p.failedTotal.Add(1)
		}
	}()

	f, err := os.Open(job.TempPath)
	if err != nil {
		job.SetFailed("cannot_open_temp_file")
		p.notifySubscribers(job)
		return
	}

	job.SetProgress(StageHashing, 0.1)
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		f.Close()
		os.Remove(job.TempPath)
		job.SetFailed("hash_error")
		p.notifySubscribers(job)
		return
	}
	sha256Hash := fmt.Sprintf("%x", hasher.Sum(nil))
	f.Seek(0, 0)

	mimeType := detectMimeTypeFromFile(f, job.OriginalName)
	f.Seek(0, 0)

	mediaType := detectMediaType(mimeType)

	job.SetProgress(StageDedup, 0.2)

	if !job.SkipNameSizeDedup && p.cfg.Media.AutoOrganize && mediaType == model.MediaTypePhoto {
		existing, _ := p.fileStore.FindByNameAndSize(job.OriginalName, job.Size)
		if existing != nil {
			f.Close()
			os.Remove(job.TempPath)
			job.SetSkipped("duplicate_name_size", existing.ID)
			p.notifySubscribers(job)
			return
		}
	}

	if !job.SkipNameSizeDedup {
		existingHash, _ := p.fileStore.FindBySHA256(sha256Hash)
		if existingHash != nil {
			f.Close()
			os.Remove(job.TempPath)
			job.SetSkipped("duplicate_content", existingHash.ID)
			p.notifySubscribers(job)
			return
		}
	}

	var exifData *model.ExifData
	var now = time.Now().UTC()

	if mediaType != model.MediaTypeFile {
		job.SetProgress(StageExif, 0.3)
		exifData, _ = p.exifService.Extract(job.TempPath)
	} else {
		job.SetProgress(StageStoring, 0.3)
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

	job.SetProgress(StageStoring, 0.5)
	pathPrefix := ""
	if mediaType == model.MediaTypeFile {
		pathPrefix = "files/"
	}
	destSubdir := filepath.Join(job.UserID, filepath.Join(pathPrefix, yearMonth))
	destDir := filepath.Join(p.cfg.OriginalsDir(), destSubdir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		f.Close()
		os.Remove(job.TempPath)
		job.SetFailed("storage_error")
		p.notifySubscribers(job)
		return
	}

	storedFilename := fmt.Sprintf("%s_%s", uuid.New().String(), job.OriginalName)
	destPath := filepath.Join(destDir, storedFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		f.Close()
		os.Remove(job.TempPath)
		job.SetFailed("write_error")
		p.notifySubscribers(job)
		return
	}

	if _, err := io.Copy(destFile, f); err != nil {
		f.Close()
		destFile.Close()
		os.Remove(job.TempPath)
		os.Remove(destPath)
		job.SetFailed("write_error")
		p.notifySubscribers(job)
		return
	}
	f.Close()
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
		OriginalName: job.OriginalName,
		Path:         filePath,
		SizeBytes:    job.Size,
		MimeType:     mimeType,
		SHA256:       sha256Hash,
		MediaType:    mediaType,
		TakenAt:      takenAt,
		FolderID:     job.FolderID,
	}

	if err := p.fileStore.Create(fileRecord); err != nil {
		os.Remove(destPath)
		job.SetFailed("db_error")
		p.notifySubscribers(job)
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
		job.SetProgress(StageThumbnails, 0.8)
		thumbs, err = p.thumbnailService.GenerateAll(fileRecord.ID, destPath, mimeType)
		if err != nil {
			slog.Warn("thumbnail generation failed", "file_id", fileRecord.ID, "error", err)
		} else {
			for _, t := range thumbs {
				if err := p.thumbnailStore.Create(t); err != nil {
					slog.Warn("failed to store thumbnail", "file_id", fileRecord.ID, "size", t.Size, "error", err)
				}
			}
		}
	}

	if p.storageService != nil {
		if err := p.storageService.PutOriginals(job.UserID, fileRecord.Filename, destPath); err != nil {
			slog.Warn("s3 upload failed for original", "file_id", fileRecord.ID, "error", err)
		}
		for _, t := range thumbs {
			ext := ".jpg"
			format := "jpg"
			if t.Size == model.ThumbSizePreview {
				ext = ".webp"
				format = "webp"
			}
			if err := p.storageService.PutThumbnail(fileRecord.ID, string(t.Size), format, t.LocalPath); err != nil {
				slog.Warn("s3 upload failed for thumbnail", "file_id", fileRecord.ID, "size", t.Size, "error", err)
			}
			_ = ext
		}
	}

	os.Remove(job.TempPath)
	job.SetCompleted(fileRecord.ID)
	job.SetProgress(StageStoring, 1.0)
	p.notifySubscribers(job)

	slog.Info("processed upload", "job_id", job.JobID, "file_id", fileRecord.ID, "name", job.OriginalName)
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
