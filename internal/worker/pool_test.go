package worker

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
)

func createTempUploadFile(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	tmpPath := filepath.Join(dir, "upload_test.jpg")
	if err := os.WriteFile(tmpPath, []byte("test image content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return tmpPath, func() {}
}

func setupTestPool(t *testing.T) (*Pool, *store.FileStore, *store.UploadJobStore, string, func()) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 2

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)
	mockfs := service.NewRealFS()

	u, err := us.Create("workeruser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	return pool, fs, ujs, u.ID, func() {
		pool.Shutdown()
	}
}

func enqueueJob(t *testing.T, ujs *store.UploadJobStore, batchID, userID, filename string, size int64, tmpPath string, folderID *string, skipDedup bool) *model.UploadJob {
	t.Helper()
	job := &model.UploadJob{
		BatchID:           batchID,
		UserID:            userID,
		Filename:          filename,
		SizeBytes:         size,
		TempPath:          tmpPath,
		FolderID:          folderID,
		SkipNameSizeDedup: skipDedup,
		Status:            model.JobStatusQueued,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}
	return job
}

func TestPool_DBProcessed_shouldCompleteJob(t *testing.T) {
	_, fs, ujs, userID, cleanup := setupTestPool(t)
	defer cleanup()

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	enqueueJob(t, ujs, "batch-db", userID, "photo.jpg", info.Size(), tmpPath, nil, true)

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for job to complete")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}
	if files[0].OriginalName != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %s", files[0].OriginalName)
	}
}

func TestPool_DBProcessed_shouldSkipDuplicateContent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = false

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("dupuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	content := []byte("duplicate-content-hash-test")
	sha256Hash := fmt.Sprintf("%x", sha256.Sum256(content))

	existing := &model.File{
		UserID:       u.ID,
		Filename:     "2026/01/dup.jpg",
		OriginalName: "dup.jpg",
		Path:         "2026/01",
		SizeBytes:    int64(len(content)),
		MimeType:     "image/jpeg",
		SHA256:       sha256Hash,
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "dup.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	job := enqueueJob(t, ujs, "batch-dup-content", u.ID, "dup.jpg", info.Size(), tmpPath, nil, false)

	deadline := time.After(5 * time.Second)
	for {
		fetched, _ := ujs.FindByID(job.ID)
		if fetched != nil && fetched.Status != model.JobStatusQueued && fetched.Status != model.JobStatusProcessing {
			if fetched.Status != model.JobStatusSkipped {
				t.Errorf("expected skipped, got %s", fetched.Status)
			}
			if fetched.Reason == nil || *fetched.Reason != "duplicate_content" {
				t.Errorf("expected reason duplicate_content, got %v", fetched.Reason)
			}
			return
		}
		select {
		case <-deadline:
			fetched, _ := ujs.FindByID(job.ID)
			t.Fatalf("timed out waiting for duplicate detection, status=%s", fetched.Status)
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestPool_Recovery_shouldResetStuckProcessingJobs(t *testing.T) {
	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("recovuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	tmpPath, _ := createTempUploadFile(t)

	job := enqueueJob(t, ujs, "batch-recovery", u.ID, "stuck.jpg", 100, tmpPath, nil, true)
	if err := ujs.SetProcessing(job.ID); err != nil {
		t.Fatalf("set processing: %v", err)
	}

	fetched, _ := ujs.FindByID(job.ID)
	if fetched.Status != model.JobStatusProcessing {
		t.Fatalf("expected processing, got %s", fetched.Status)
	}

	cfg := config.DefaultConfig()
	cfg.Upload.ConcurrentWorkers = 1

	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	fetched, _ = ujs.FindByID(job.ID)
	if fetched.Status != model.JobStatusQueued {
		t.Errorf("expected queued after recovery, got %s", fetched.Status)
	}
}

func TestPool_NonMediaFile_shouldUseFilesPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = false

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("fileuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "document.pdf")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	enqueueJob(t, ujs, "batch-files", u.ID, "document.pdf", info.Size(), tmpPath, nil, true)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for job to complete")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	for _, f := range files {
		if f.MediaType == model.MediaTypeFile {
			if !strings.Contains(f.Path, "files") {
				t.Errorf("expected 'files' in path for non-media file, got %q", f.Path)
			}
			if !strings.Contains(f.Filename, "files") {
				t.Errorf("expected 'files' in filename for non-media file, got %q", f.Filename)
			}
		}
	}
}

func TestPool_NonMediaFile_shouldSkipExifAndThumbnails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = false

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("skipuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "document.pdf")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	enqueueJob(t, ujs, "batch-skip", u.ID, "document.pdf", info.Size(), tmpPath, nil, true)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for job to complete")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	for _, f := range files {
		exifData, _ := es.FindByFileID(f.ID)
		if exifData != nil {
			t.Errorf("expected no exif data for non-media file %q, got camera %v", f.OriginalName, exifData.CameraModel)
		}
		for _, size := range []model.ThumbnailSize{model.ThumbSizeSmall, model.ThumbSizeLarge, model.ThumbSizeMedium, model.ThumbSizePreview} {
			thumb, _ := ts.FindByFileIDAndSize(f.ID, size)
			if thumb != nil {
				t.Errorf("expected no thumbnails for non-media file %q at size %s, got %+v", f.OriginalName, size, thumb)
			}
		}
	}
}

func TestPool_Subscribe_shouldReceiveUpdates(t *testing.T) {
	pool, _, ujs, userID, cleanup := setupTestPool(t)
	defer cleanup()

	tmpPath, _ := createTempUploadFile(t)

	ch := pool.Subscribe("batch-sub", "listener-1")
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
	defer pool.Unsubscribe("batch-sub", "listener-1")

	job := enqueueJob(t, ujs, "batch-sub", userID, "photo.jpg", 1024, tmpPath, nil, true)

	timeout := time.After(2 * time.Second)
	received := false
	for !received {
		select {
		case update := <-ch:
			if update.ID == job.ID {
				received = true
			}
		case <-timeout:
			t.Error("timed out waiting for subscriber update")
			received = true
		}
	}
}

func TestPool_SubscribeUser_shouldReceiveUpdates(t *testing.T) {
	pool, _, ujs, userID, cleanup := setupTestPool(t)
	defer cleanup()

	tmpPath, _ := createTempUploadFile(t)

	ch := pool.SubscribeUser(userID, "listener-1")
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
	defer pool.UnsubscribeUser(userID, "listener-1")

	enqueueJob(t, ujs, "batch-global-sub", userID, "photo.jpg", 1024, tmpPath, nil, true)

	timeout := time.After(2 * time.Second)
	received := false
	for !received {
		select {
		case <-ch:
			received = true
		case <-timeout:
			t.Error("timed out waiting for user subscriber update")
			received = true
		}
	}
}

func TestPool_SkipNameSizeDedup_shouldSkipChecksWhenFlagSet(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = false
	cfg.Media.AutoOrganize = true

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("skipdedup_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	content := []byte("same-content-for-sha256-test")
	sha256Hash := fmt.Sprintf("%x", sha256.Sum256(content))

	existing := &model.File{
		UserID:       u.ID,
		Filename:     "other-folder/dup.jpg",
		OriginalName: "dup.jpg",
		Path:         "other-folder",
		SizeBytes:    int64(len(content)),
		MimeType:     "image/jpeg",
		SHA256:       sha256Hash,
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "dup.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	enqueueJob(t, ujs, "batch-dedup-skip", u.ID, "dup.jpg", info.Size(), tmpPath, nil, true)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for {
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
		if len(files) >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting, got %d files", len(files))
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}

	count := 0
	for _, f := range files {
		if f.OriginalName == "dup.jpg" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 files named dup.jpg (existing + new upload with dedup skipped), got %d", count)
	}
}

func TestPool_Stats_shouldReportCompleted(t *testing.T) {
	pool, _, ujs, userID, cleanup := setupTestPool(t)
	defer cleanup()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(path, []byte("stats test content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(path)

	enqueueJob(t, ujs, "batch-stats", userID, "test.bin", info.Size(), path, nil, true)

	deadline := time.After(5 * time.Second)
	for {
		stats := pool.Stats()
		total := stats.CompletedTotal + stats.FailedTotal + stats.SkippedTotal
		if total >= 1 {
			if stats.TotalWorkers != 2 {
				t.Errorf("expected total workers 2, got %d", stats.TotalWorkers)
			}
			return
		}
		select {
		case <-deadline:
			t.Error("timed out waiting for job to complete")
			return
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestPool_Stats_shouldReportActiveWorkers(t *testing.T) {
	pool, _, _, _, cleanup := setupTestPool(t)
	defer cleanup()

	pool.processingMu.Lock()
	pool.processingJobs["test-job-1"] = &model.UploadJob{
		ID:       "test-job-1",
		Filename: "processing.jpg",
		Status:   model.JobStatusProcessing,
		Progress: 0.8,
	}
	stage := model.JobStageThumbnails
	pool.processingJobs["test-job-1"].Stage = &stage
	pool.processingMu.Unlock()

	stats := pool.Stats()
	if len(stats.ProcessingJobs) != 1 {
		t.Fatalf("expected 1 processing job, got %d", len(stats.ProcessingJobs))
	}
	pj := stats.ProcessingJobs[0]
	if pj.JobID != "test-job-1" {
		t.Errorf("expected job_id test-job-1, got %s", pj.JobID)
	}
	if pj.Stage != string(model.JobStageThumbnails) {
		t.Errorf("expected stage thumbnails, got %s", pj.Stage)
	}

	pool.processingMu.Lock()
	delete(pool.processingJobs, "test-job-1")
	pool.processingMu.Unlock()
}

func TestPool_Shutdown_shouldStopWorkers(t *testing.T) {
	pool, _, _, _, cleanup := setupTestPool(t)
	cleanup()

	stats := pool.Stats()
	if stats.ActiveWorkers != 0 {
		t.Errorf("expected 0 active workers after shutdown, got %d", stats.ActiveWorkers)
	}
}

func TestPool_NotifyJobsAvailable_shouldWakeClaimerImmediately(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("notifyuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "notify-test.bin")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	job := enqueueJob(t, ujs, "batch-notify", u.ID, "notify-test.bin", info.Size(), tmpPath, nil, true)
	pool.NotifyJobsAvailable()

	pickedUp := false
	deadline := time.After(100 * time.Millisecond)
	for !pickedUp {
		select {
		case <-deadline:
			t.Fatal("claimer did not pick up job within 100ms after NotifyJobsAvailable")
		default:
		}
		fetched, _ := ujs.FindByID(job.ID)
		if fetched != nil && fetched.Status != model.JobStatusQueued {
			pickedUp = true
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDetectMimeTypeByExtension_known(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"photo.png", "image/png"},
		{"photo.webp", "image/webp"},
		{"photo.heic", "image/heic"},
		{"video.mp4", "video/mp4"},
		{"video.mov", "video/quicktime"},
		{"doc.pdf", "application/pdf"},
		{"archive.zip", "application/zip"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectMimeTypeByExtension(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected model.MediaType
	}{
		{"image/jpeg", model.MediaTypePhoto},
		{"image/png", model.MediaTypePhoto},
		{"image/heic", model.MediaTypePhoto},
		{"video/mp4", model.MediaTypeVideo},
		{"video/quicktime", model.MediaTypeVideo},
		{"application/pdf", model.MediaTypeFile},
		{"text/plain", model.MediaTypeFile},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := detectMediaType(tt.mimeType)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestPool_Claimer_shouldNotClaimWhenChannelFull(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cap := cfg.Upload.ConcurrentWorkers * 2

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("capacityuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	pool.Shutdown()

	for i := 0; i < cap; i++ {
		pool.jobCh <- &model.UploadJob{
			ID:       "sentinel-" + strconv.Itoa(i),
			TempPath: "/nonexistent/" + strconv.Itoa(i),
		}
	}

	if len(pool.jobCh) != cap {
		t.Fatalf("expected channel full (%d), got %d", cap, len(pool.jobCh))
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "capacity-test.jpg")
	if err := os.WriteFile(tmpPath, []byte("capacity test content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	jobIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		job := enqueueJob(t, ujs, "batch-capacity", u.ID, fmt.Sprintf("photo%d.jpg", i), info.Size(), tmpPath, nil, true)
		jobIDs[i] = job.ID
	}

	exifSvc := service.NewExifService(mockfs)
	thumbSvc := service.NewThumbnailService(cfg.ThumbnailsDir(), mockfs)
	p := &Pool{
		cfg:              cfg,
		fs:               mockfs,
		fileStore:        fs,
		exifStore:        es,
		thumbnailStore:   ts,
		exifService:      exifSvc,
		thumbnailService: thumbSvc,
		storageService:   nil,
		uploadJobStore:   ujs,
		totalWorkers:     1,
		stopCh:           make(chan struct{}),
		jobCh:            pool.jobCh,
		s3JobCh:          make(chan *s3Task, 4),
		s3Workers:        2,
		subscribers:      make(map[string]map[string]chan *model.UploadJob),
		userSubscribers:  make(map[string]map[string]chan *model.UploadJob),
		processingJobs:   make(map[string]*model.UploadJob),
		notifyCh:         make(chan struct{}, 1),
	}

	p.wg.Add(1)
	p.NotifyJobsAvailable()
	go p.claimer()

	time.Sleep(pollInterval + 500*time.Millisecond)

	for _, id := range jobIDs {
		job, _ := ujs.FindByID(id)
		if job == nil {
			t.Fatalf("job %s not found", id)
		}
		if job.Status == model.JobStatusFailed {
			t.Errorf("job %s failed with reason %v", id, job.Reason)
		}
		if job.Status == model.JobStatusProcessing {
			t.Errorf("job %s claimed despite full channel", id)
		}
		if job.Status != model.JobStatusQueued {
			t.Errorf("job %s expected queued, got %s", id, job.Status)
		}
	}

	for i := 0; i < cap; i++ {
		select {
		case <-p.jobCh:
		default:
		}
	}

	close(p.stopCh)
	p.wg.Wait()
}

func TestDetectMimeTypeFromFile_tinyFilesShouldNotPanic(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 15, 20, 50}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d_bytes", size), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "tiny.bin")
			content := make([]byte, size)
			if size > 0 {
				for i := range content {
					content[i] = byte(i % 256)
				}
			}
			if err := os.WriteFile(path, content, 0644); err != nil {
				t.Fatalf("write temp file: %v", err)
			}
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("open temp file: %v", err)
			}
			defer f.Close()

			result := detectMimeTypeFromFile(f, "tiny.bin")
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
}
}

func setupChunkedTestPool(t *testing.T) (*Pool, *config.Config, *store.FileStore, *store.UploadJobStore, *store.ChunkStore, *store.DB, string, func()) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 2

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)
	mockfs := service.NewRealFS()
	cs := store.NewChunkStore(db, mockfs)

	u, err := us.Create("chunkedworker_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, cs, nil)
	return pool, cfg, fs, ujs, cs, db, u.ID, func() {
		pool.Shutdown()
	}
}

func createChunkedJob(t *testing.T, cfg *config.Config, ujs *store.UploadJobStore, cs *store.ChunkStore, userID, filename string, totalChunks int, chunkData []byte) (*model.UploadJob, string) {
	t.Helper()

	chunkSize := len(chunkData)
	totalSize := int64(chunkSize * totalChunks)
	tc := totalChunks
	csInt := int64(chunkSize)

	chunkDir := store.ChunkTempDir(cfg.OriginalsDir())
	os.MkdirAll(chunkDir, 0755)

	job := &model.UploadJob{
		BatchID:           "chunked-batch-" + strings.ReplaceAll(t.Name(), "/", "_"),
		UserID:            userID,
		Filename:          filename,
		SizeBytes:         totalSize,
		TempPath:          chunkDir,
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       &tc,
		ChunkSize:         &csInt,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", i))
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
		if err := cs.CreateChunkRecord(job.ID, i, int64(chunkSize), int64(i*chunkSize), "", chunkPath); err != nil {
			t.Fatalf("create chunk record %d: %v", i, err)
		}
	}

	return job, job.ID
}

func TestPool_ChunkedJob_shouldAssembleAndComplete(t *testing.T) {
	_, cfg, fs, ujs, cs, _, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	chunkData := []byte("hello world chunked test data for assembly verification")
	job, _ := createChunkedJob(t, cfg, ujs, cs, userID, "chunked_test.dat", 2, chunkData)

	var files []*model.File
	deadline := time.After(10 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			jobStatus, _ := ujs.FindByID(job.ID)
			if jobStatus != nil {
				t.Fatalf("timed out waiting for chunked job to complete, status=%s error=%v", jobStatus.Status, jobStatus.Error)
			}
			t.Fatal("timed out waiting for chunked job to complete, job not found")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	if files[0].OriginalName != "chunked_test.dat" {
		t.Errorf("expected chunked_test.dat, got %s", files[0].OriginalName)
	}
	if files[0].SizeBytes != int64(len(chunkData)*2) {
		t.Errorf("expected size %d, got %d", len(chunkData)*2, files[0].SizeBytes)
	}

	storedCount, err := cs.GetStoredChunkCount(job.ID)
	if err != nil {
		t.Fatalf("get stored chunk count: %v", err)
	}
	if storedCount != 0 {
		t.Errorf("expected 0 chunks after assembly, got %d", storedCount)
	}
}

func TestPool_ChunkedJob_singleChunk_shouldComplete(t *testing.T) {
	_, cfg, fs, ujs, cs, _, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	chunkData := []byte("single chunk file data")
	job, _ := createChunkedJob(t, cfg, ujs, cs, userID, "single_chunk.dat", 1, chunkData)

	var files []*model.File
	deadline := time.After(10 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			jobStatus, _ := ujs.FindByID(job.ID)
			if jobStatus != nil {
				t.Fatalf("timed out, status=%s error=%v", jobStatus.Status, jobStatus.Error)
			}
			t.Fatal("timed out, job not found")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	if len(files) == 0 {
		t.Fatal("expected at least 1 file")
	}
}

func TestPool_ChunkedJob_incompleteChunks_shouldStayQueued(t *testing.T) {
	_, _, _, ujs, _, _, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	job := &model.UploadJob{
		BatchID:           "incomplete-batch",
		UserID:            userID,
		Filename:          "incomplete.dat",
		SizeBytes:         2048,
		TempPath:          os.TempDir(),
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       intPtr(2),
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	time.Sleep(5 * time.Second)

	var finalStatus model.JobStatus
	for i := 0; i < 10; i++ {
		j, err := ujs.FindByID(job.ID)
		if err != nil {
			t.Fatalf("find job: %v", err)
		}
		if j == nil {
			t.Fatal("job not found")
		}
		finalStatus = j.Status
		if finalStatus == model.JobStatusQueued {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Errorf("expected status queued after retries, got %s", finalStatus)
}

func TestPool_ChunkedJob_oldIncomplete_shouldExpire(t *testing.T) {
	_, _, _, ujs, cs, db, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	chunkSize := len([]byte("data"))
	tc := 2
	csInt := int64(chunkSize)

	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	oldTimeStr := oldTime.Format(time.RFC3339)

	job := &model.UploadJob{
		BatchID:           "expired-batch",
		UserID:            userID,
		Filename:          "expired.dat",
		SizeBytes:         int64(chunkSize * tc),
		TempPath:          os.TempDir(),
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       &tc,
		ChunkSize:         &csInt,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	cs.CreateChunkRecord(job.ID, 0, int64(chunkSize), 0, "", "/tmp/expired_c0")

	db.Exec(`UPDATE upload_jobs SET created_at = ? WHERE id = ?`, oldTimeStr, job.ID)

	var finalStatus model.JobStatus
	var finalError string
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for expiry, status=%s error=%s", finalStatus, finalError)
		default:
		}
		j, err := ujs.FindByID(job.ID)
		if err != nil {
			t.Fatalf("find job: %v", err)
		}
		if j == nil {
			t.Fatal("job not found")
		}
		finalStatus = j.Status
		if j.Error != nil {
			finalError = *j.Error
		}
		if finalStatus == model.JobStatusFailed {
			if finalError != "upload_expired" {
				t.Errorf("expected upload_expired, got %s", finalError)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestPool_ChunkedJob_shouldPreserveContentIntegrity(t *testing.T) {
	_, cfg, fs, ujs, cs, _, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	prefix := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 1)
	middle := bytes.Repeat([]byte{0xAB}, 800)
	suffix := []byte{0xFF, 0xD9}
	content := make([]byte, 0, len(prefix)+len(middle)+len(suffix))
	content = append(content, prefix...)
	content = append(content, middle...)
	content = append(content, suffix...)
	expectSHA := fmt.Sprintf("%x", sha256.Sum256(content))

	half := len(content) / 2
	chunk0 := content[:half]
	chunk1 := content[half:]

	totalChunks := 2
	totalSize := int64(len(content))
	tc := totalChunks
	csInt := int64(half)

	chunkDir := store.ChunkTempDir(cfg.OriginalsDir())
	os.MkdirAll(chunkDir, 0755)

	job := &model.UploadJob{
		BatchID:           "chunked-integrity-" + strings.ReplaceAll(t.Name(), "/", "_"),
		UserID:            userID,
		Filename:          "photo.jpg",
		SizeBytes:         totalSize,
		TempPath:          chunkDir,
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       &tc,
		ChunkSize:         &csInt,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}

	for i, data := range [][]byte{chunk0, chunk1} {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%d", i))
		if err := os.WriteFile(chunkPath, data, 0644); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
		if err := cs.CreateChunkRecord(job.ID, i, int64(len(data)), int64(i*half), fmt.Sprintf("%x", sha256.Sum256(data)), chunkPath); err != nil {
			t.Fatalf("create chunk record %d: %v", i, err)
		}
	}

	var files []*model.File
	deadline := time.After(15 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			jobStatus, _ := ujs.FindByID(job.ID)
			if jobStatus != nil {
				t.Fatalf("timed out, status=%s error=%v", jobStatus.Status, jobStatus.Error)
			}
			t.Fatal("timed out, job not found")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	storedPath := filepath.Join(cfg.OriginalsDir(), userID, files[0].Filename)
	storedBytes, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored file %q: %v", storedPath, err)
	}

	if len(storedBytes) != len(content) {
		t.Fatalf("stored file size %d != original %d", len(storedBytes), len(content))
	}

	if !bytes.Equal(storedBytes[:len(prefix)], prefix) {
		t.Errorf("JPEG magic bytes corrupted: expected %x, got %x", prefix, storedBytes[:len(prefix)])
	}

	if !bytes.Equal(storedBytes[len(content)-len(suffix):], suffix) {
		t.Errorf("JPEG EOI marker corrupted: expected %x, got %x", suffix, storedBytes[len(content)-len(suffix):])
	}

	storedSHA := fmt.Sprintf("%x", sha256.Sum256(storedBytes))
	if storedSHA != expectSHA {
		t.Errorf("SHA-256 mismatch: expected %s, got %s", expectSHA, storedSHA)
	}

	if files[0].SHA256 != expectSHA {
		t.Errorf("stored SHA-256 in DB mismatch: expected %s, got %s", expectSHA, files[0].SHA256)
	}
}

func TestPool_ChunkedJob_shouldGenerateThumbnailsForPhoto(t *testing.T) {
	pool, cfg, fs, ujs, cs, db, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	ts := store.NewThumbnailStore(db)

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	var jpgBuf bytes.Buffer
	if err := jpeg.Encode(&jpgBuf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}
	content := jpgBuf.Bytes()

	totalChunks := 2
	totalSize := int64(len(content))
	half := totalSize / 2
	tc := totalChunks
	csInt := half

	chunkDir := store.ChunkTempDir(cfg.OriginalsDir())
	os.MkdirAll(chunkDir, 0755)

	job := &model.UploadJob{
		BatchID:           "chunked-thumb-" + strings.ReplaceAll(t.Name(), "/", "_"),
		UserID:            userID,
		Filename:          "thumb_test.jpg",
		SizeBytes:         totalSize,
		TempPath:          chunkDir,
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       &tc,
		ChunkSize:         &csInt,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}

	chunk0 := content[:half]
	chunk1 := content[half:]
	for i, data := range [][]byte{chunk0, chunk1} {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("thumb_chunk_%d", i))
		if err := os.WriteFile(chunkPath, data, 0644); err != nil {
			t.Fatalf("write chunk %d: %v", i, err)
		}
		if err := cs.CreateChunkRecord(job.ID, i, int64(len(data)), int64(i)*half, fmt.Sprintf("%x", sha256.Sum256(data)), chunkPath); err != nil {
			t.Fatalf("create chunk record %d: %v", i, err)
		}
	}

	var files []*model.File
	deadline := time.After(15 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			jobStatus, _ := ujs.FindByID(job.ID)
			if jobStatus != nil {
				t.Fatalf("timed out, status=%s error=%v", jobStatus.Status, jobStatus.Error)
			}
			t.Fatal("timed out, job not found")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	fileID := files[0].ID

	sizes := []model.ThumbnailSize{model.ThumbSizeSmall, model.ThumbSizeLarge, model.ThumbSizeMedium, model.ThumbSizeXL, model.ThumbSizePreview}
	deadline2 := time.After(10 * time.Second)
	for _, size := range sizes {
		var thumb *model.Thumbnail
		var err error
		for thumb == nil {
			select {
			case <-deadline2:
				t.Fatalf("timed out waiting for thumbnail %s", size)
			default:
			}
			thumb, err = ts.FindByFileIDAndSize(fileID, size)
			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				t.Errorf("FindByFileIDAndSize(%q, %s): %v", fileID, size, err)
				break
			}
			if thumb.LocalPath == "" {
				t.Errorf("thumbnail %s has empty LocalPath", size)
				break
			}
			st, stErr := os.Stat(thumb.LocalPath)
			if stErr != nil {
				t.Errorf("thumbnail file %s not found: %v", thumb.LocalPath, stErr)
				break
			}
			if st.Size() == 0 {
				t.Errorf("thumbnail file %s is empty", thumb.LocalPath)
			}
		}
	}

	_ = pool
}

func TestPool_ChunkedJob_smallFile_shouldNotBeTruncated(t *testing.T) {
	_, cfg, fs, ujs, cs, _, userID, cleanup := setupChunkedTestPool(t)
	defer cleanup()

	content := bytes.Repeat([]byte("X"), 200)

	totalChunks := 1
	totalSize := int64(len(content))
	tc := totalChunks
	csInt := int64(1024 * 1024)

	chunkDir := store.ChunkTempDir(cfg.OriginalsDir())
	os.MkdirAll(chunkDir, 0755)

	job := &model.UploadJob{
		BatchID:           "chunked-small-" + strings.ReplaceAll(t.Name(), "/", "_"),
		UserID:            userID,
		Filename:          "small.bin",
		SizeBytes:         totalSize,
		TempPath:          chunkDir,
		Status:            model.JobStatusQueued,
		UploadMode:        model.UploadModeChunked,
		TotalChunks:       &tc,
		ChunkSize:         &csInt,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}

	chunkPath := filepath.Join(chunkDir, "small_chunk_0")
	if err := os.WriteFile(chunkPath, content, 0644); err != nil {
		t.Fatalf("write chunk: %v", err)
	}
	if err := cs.CreateChunkRecord(job.ID, 0, int64(len(content)), 0, fmt.Sprintf("%x", sha256.Sum256(content)), chunkPath); err != nil {
		t.Fatalf("create chunk record: %v", err)
	}

	var files []*model.File
	deadline := time.After(15 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			jobStatus, _ := ujs.FindByID(job.ID)
			if jobStatus != nil {
				t.Fatalf("timed out, status=%s error=%v", jobStatus.Status, jobStatus.Error)
			}
			t.Fatal("timed out, job not found")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		var err error
		files, _, _, err = fs.List(store.FileListOptions{UserID: userID, Limit: 10})
		if err != nil {
			t.Fatalf("list files: %v", err)
		}
	}

	storedPath := filepath.Join(cfg.OriginalsDir(), userID, files[0].Filename)
	storedBytes, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read stored file %q: %v", storedPath, err)
	}

	if len(storedBytes) != len(content) {
		t.Fatalf("stored file size %d != original %d", len(storedBytes), len(content))
	}

	if !bytes.Equal(storedBytes, content) {
		t.Errorf("stored file content does not match original")
	}

	if files[0].SizeBytes != totalSize {
		t.Errorf("SizeBytes in DB %d != expected %d", files[0].SizeBytes, totalSize)
	}
}

func TestPool_s3Worker_shouldSkipWithNilStorageService(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)

	u, err := us.Create("s3worker_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	f := &model.File{
		UserID:       u.ID,
		Filename:     "videos/test.mp4",
		OriginalName: "test.mp4",
		Path:         "videos",
		SizeBytes:    5000000,
		MimeType:     "video/mp4",
		SHA256:       "abc123",
		MediaType:    model.MediaTypeVideo,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	proxy := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeVideoProxy,
		Width:     720,
		Height:    405,
		Format:    "mp4",
		LocalPath: "/tmp/thumb-video_proxy.mp4",
		SizeBytes: 2000000,
	}
	if err := ts.Create(proxy); err != nil {
		t.Fatalf("create thumbnail: %v", err)
	}

	still := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeVideoStill,
		Width:     600,
		Height:    338,
		Format:    "jpeg",
		LocalPath: "/tmp/thumb-video_still.jpg",
		SizeBytes: 50000,
	}
	if err := ts.Create(still); err != nil {
		t.Fatalf("create thumbnail: %v", err)
	}

	p := &Pool{
		cfg:            cfg,
		thumbnailStore: ts,
		storageService: nil,
		stopCh:         make(chan struct{}),
		s3JobCh:        make(chan *s3Task, 4),
		s3Workers:      1,
	}

	p.wg.Add(1)
	go p.s3Worker(0)

	task := &s3Task{
		UserID:   u.ID,
		Filename: "test/file.mp4",
		FileID:   f.ID,
		Thumbnails: []*model.Thumbnail{
			proxy, still,
		},
	}

	p.s3JobCh <- task
	time.Sleep(100 * time.Millisecond)

	close(p.stopCh)
	p.wg.Wait()

	found, err := ts.FindByFileIDAndSize(f.ID, model.ThumbSizeVideoProxy)
	if err != nil {
		t.Fatalf("FindByFileIDAndSize: %v", err)
	}
	if found.S3Key != nil {
		t.Error("expected S3Key to remain nil with nil storageService")
	}
}

func TestPool_Reconciliation_shouldRunAndReport(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("reconuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	userDir := filepath.Join(cfg.OriginalsDir(), u.ID, "2024/07")
	os.MkdirAll(userDir, 0755)
	realPath := filepath.Join(userDir, "recon_photo.jpg")
	os.WriteFile(realPath, []byte("recon test data"), 0644)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/recon_photo.jpg",
		OriginalName: "recon_photo.jpg",
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       fmt.Sprintf("%x", sha256.Sum256([]byte("recon test data"))),
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, store.NewExifStore(db), ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	result := pool.RunReconciliation()
	if created, ok := result["created"].(int); !ok || created == 0 {
		t.Errorf("expected at least 1 reconciliation job created, got %v", result["created"])
	}
}

func TestPool_StandardJob_tempFileRemovedBeforeProcessing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("tmpfileuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent.jpg")
	job := &model.UploadJob{
		BatchID:           "batch-missing-temp",
		UserID:            u.ID,
		Filename:          "missing.jpg",
		SizeBytes:         100,
		TempPath:          nonexistentPath,
		Status:            model.JobStatusQueued,
		SkipNameSizeDedup: true,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, store.NewFileStore(db), store.NewExifStore(db), store.NewThumbnailStore(db), nil, ujs, nil, nil)
	defer pool.Shutdown()

	deadline := time.After(5 * time.Second)
	for {
		fetched, _ := ujs.FindByID(job.ID)
		if fetched != nil && fetched.Status == model.JobStatusFailed {
			if fetched.Error == nil || *fetched.Error != "temp_file_missing" {
				t.Errorf("expected temp_file_missing error, got %v", fetched.Error)
			}
			return
		}
		select {
		case <-deadline:
		fetched, _ := ujs.FindByID(job.ID)
		t.Fatalf("timed out, status=%s error=%v", fetched.Status, fetched.Error)
	default:
	}
	}
}

func TestPool_ProcessReconcileJob_noFileID_fails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("reconnofile_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := &Pool{
		cfg:            cfg,
		fs:             service.NewRealFS(),
		fileStore:      store.NewFileStore(db),
		uploadJobStore: ujs,
	}

	f, _ := os.CreateTemp(t.TempDir(), "recon-*.jpg")
	defer f.Close()
	job := &model.UploadJob{Filename: "test.jpg", UserID: u.ID, TempPath: f.Name(), Status: model.JobStatusProcessing, BatchID: "batch-nofileid"}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	pool.processReconcileJob(job, f)

	found, err := ujs.FindByID(job.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil {
		t.Fatal("job should exist in DB")
	}
	if found.Status != model.JobStatusFailed {
		t.Errorf("expected failed status, got %s", found.Status)
	}
	if found.Error == nil || *found.Error != "reconcile_missing_file_id" {
		t.Errorf("expected reconcile_missing_file_id error, got %v", found.Error)
	}
}

func TestPool_ProcessReconcileJob_fileNotFound_fails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("reconnofile_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := &Pool{
		cfg:            cfg,
		fs:             service.NewRealFS(),
		fileStore:      store.NewFileStore(db),
		uploadJobStore: ujs,
	}

	fileID := "nonexistent-file"
	job := &model.UploadJob{Filename: "test.jpg", UserID: u.ID, FileID: &fileID, Status: model.JobStatusProcessing, BatchID: "batch-nofile"}
	f, _ := os.CreateTemp(t.TempDir(), "recon-*.jpg")
	defer f.Close()

	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	pool.processReconcileJob(job, f)

	fetched, err := ujs.FindByID(job.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("job should exist")
	}
	if fetched.Status != model.JobStatusFailed {
		t.Errorf("expected failed, got %s", fetched.Status)
	}
	if fetched.Error == nil || *fetched.Error != "reconcile_file_not_found" {
		t.Errorf("expected reconcile_file_not_found, got %v", fetched.Error)
	}
}

func TestPool_StartReconciler_stopsOnShutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"

	db := store.OpenTestDB(t)
	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, store.NewFileStore(db), store.NewExifStore(db), store.NewThumbnailStore(db), nil, store.NewUploadJobStore(db), nil, nil)

	pool.StartReconciler(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	pool.Shutdown()
}

func intPtr(n int) *int { return &n }

func TestPool_StandardJob_nameSizeDedup_sameUser(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = true

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("namededup_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	content := []byte("unique photo content for name dedup test")
	existing := &model.File{
		UserID:       u.ID,
		Filename:     "2026/01/photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "2026/01",
		SizeBytes:    int64(len(content)),
		MimeType:     "image/jpeg",
		SHA256:       "different-hash-for-name-dedup",
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	job := enqueueJob(t, ujs, "batch-name-dedup", u.ID, "photo.jpg", info.Size(), tmpPath, nil, false)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	deadline := time.After(5 * time.Second)
	for {
		fetched, _ := ujs.FindByID(job.ID)
		if fetched != nil && fetched.Status != model.JobStatusQueued && fetched.Status != model.JobStatusProcessing {
			if fetched.Status != model.JobStatusSkipped {
				t.Errorf("expected skipped, got %s", fetched.Status)
			}
			if fetched.Reason == nil || *fetched.Reason != "duplicate_name_size" {
				t.Errorf("expected duplicate_name_size, got %v", fetched.Reason)
			}
			return
		}
		select {
		case <-deadline:
			fetched, _ := ujs.FindByID(job.ID)
			t.Fatalf("timed out, status=%s", fetched.Status)
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestPool_StandardJob_nameSizeDedup_differentUser(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = true

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u1, err := us.Create("userA_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create userA: %v", err)
	}
	u2, err := us.Create("userB_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create userB: %v", err)
	}

	content := []byte("different user dedup test data")
	existing := &model.File{
		UserID:       u1.ID,
		Filename:     "2026/01/diffuser.jpg",
		OriginalName: "diffuser.jpg",
		Path:         "2026/01",
		SizeBytes:    int64(len(content)),
		MimeType:     "image/jpeg",
		SHA256:       fmt.Sprintf("%x", sha256.Sum256(content)),
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "diffuser.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	enqueueJob(t, ujs, "batch-diffuser", u2.ID, "diffuser.jpg", info.Size(), tmpPath, nil, false)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out — job should complete for different user")
		default:
		}
		time.Sleep(200 * time.Millisecond)
		files, _, _, _ = fs.List(store.FileListOptions{UserID: u2.ID, Limit: 10})
	}

	if files[0].OriginalName != "diffuser.jpg" {
		t.Errorf("expected diffuser.jpg, got %s", files[0].OriginalName)
	}
}

func TestPool_Reconciliation_successPath_mediaFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	tmpDir := t.TempDir()
	cfg.Storage.Local.Path = tmpDir

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("reconsuccess_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	var jpgBuf bytes.Buffer
	if err := jpeg.Encode(&jpgBuf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}
	jpgContent := jpgBuf.Bytes()

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/01/recon_success.jpg",
		OriginalName: "recon_success.jpg",
		Path:         "2024/01",
		SizeBytes:    int64(len(jpgContent)),
		MimeType:     "image/jpeg",
		SHA256:       fmt.Sprintf("%x", sha256.Sum256(jpgContent)),
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	tmpDir2 := t.TempDir()
	tmpPath := filepath.Join(tmpDir2, "recon_success.jpg")
	if err := os.WriteFile(tmpPath, jpgContent, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := &Pool{
		cfg:              cfg,
		fs:               mockfs,
		fileStore:        fs,
		thumbnailStore:   ts,
		uploadJobStore:   ujs,
		thumbnailService: service.NewThumbnailService(cfg.ThumbnailsDir(), mockfs),
	}

	tf, err := os.Open(tmpPath)
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}

	job := &model.UploadJob{Filename: "recon_success.jpg", UserID: u.ID, FileID: &f.ID, Status: model.JobStatusProcessing, BatchID: "batch-recon-success", TempPath: tmpPath}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	pool.processReconcileJob(job, tf)

	fetched, err := ujs.FindByID(job.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("job should exist")
	}
	if fetched.Status != model.JobStatusCompleted {
		t.Errorf("expected completed, got %s (error=%v)", fetched.Status, fetched.Error)
	}

	thumbsFound := 0
	for _, size := range []model.ThumbnailSize{model.ThumbSizeSmall, model.ThumbSizeLarge, model.ThumbSizeMedium, model.ThumbSizeXL} {
		thumb, _ := ts.FindByFileIDAndSize(f.ID, size)
		if thumb != nil {
			thumbsFound++
		}
	}
	if thumbsFound == 0 {
		t.Error("expected at least one thumbnail to be generated")
	}
}

func TestPool_Reconciliation_successPath_nonMediaFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ujs := store.NewUploadJobStore(db)

	u, err := us.Create("reconfile_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	f := &model.File{
		UserID:       u.ID,
		Filename:     "files/2024/01/recon_doc.pdf",
		OriginalName: "recon_doc.pdf",
		Path:         "files/2024/01",
		SizeBytes:    100,
		MimeType:     "application/pdf",
		SHA256:       "recondoc-hash",
		MediaType:    model.MediaTypeFile,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "recon_doc.pdf")
	if err := os.WriteFile(tmpPath, []byte("pdf content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	mockfs := service.NewRealFS()
	pool := &Pool{
		cfg:            cfg,
		fs:             mockfs,
		fileStore:      fs,
		uploadJobStore: ujs,
	}

	tf, _ := os.Open(tmpPath)
	job := &model.UploadJob{Filename: "recon_doc.pdf", UserID: u.ID, FileID: &f.ID, Status: model.JobStatusProcessing, BatchID: "batch-recon-nonmedia"}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	pool.processReconcileJob(job, tf)

	fetched, _ := ujs.FindByID(job.ID)
	if fetched == nil || fetched.Status != model.JobStatusCompleted {
		t.Errorf("expected completed, got %v", fetched)
	}
}

func TestPool_StandardJob_duplicateContent_differentUser_shouldUpload(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = false

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)
	ujs := store.NewUploadJobStore(db)

	u1, err := us.Create("ctuserA_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create userA: %v", err)
	}
	u2, err := us.Create("ctuserB_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create userB: %v", err)
	}

	content := []byte("same-content-across-users-for-dedup-test")
	sha256Hash := fmt.Sprintf("%x", sha256.Sum256(content))

	existing := &model.File{
		UserID:       u1.ID,
		Filename:     "2026/01/shared-content.jpg",
		OriginalName: "shared-content.jpg",
		Path:         "2026/01",
		SizeBytes:    int64(len(content)),
		MimeType:     "image/jpeg",
		SHA256:       sha256Hash,
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "shared-content.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	job := enqueueJob(t, ujs, "batch-ct-diffuser", u2.ID, "shared-content.jpg", info.Size(), tmpPath, nil, false)

	mockfs := service.NewRealFS()
	pool := NewPool(cfg, mockfs, fs, es, ts, nil, ujs, nil, nil)
	defer pool.Shutdown()

	var files []*model.File
	deadline := time.After(5 * time.Second)
	for len(files) == 0 {
		select {
		case <-deadline:
			fetched, _ := ujs.FindByID(job.ID)
			t.Fatalf("timed out, status=%s", fetched.Status)
		default:
		}
		time.Sleep(200 * time.Millisecond)
		files, _, _, _ = fs.List(store.FileListOptions{UserID: u2.ID, Limit: 10})
	}

	if files[0].OriginalName != "shared-content.jpg" {
		t.Errorf("expected shared-content.jpg, got %s", files[0].OriginalName)
	}
}
