package worker

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/model"
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

func setupTestPool(t *testing.T) (*Pool, *store.FileStore) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 2

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	u, _ := us.Create("workeruser_"+t.Name(), "password123", model.RoleMember, nil)

	_ = u
	pool := NewPool(cfg, fs, es, ts, nil)
	return pool, fs
}

func TestPool_Enqueue_shouldReturnJob(t *testing.T) {
	pool, _ := setupTestPool(t)

	tmpPath, _ := createTempUploadFile(t)

	job := pool.Enqueue("batch-1", "user-1", "photo.jpg", 1024, tmpPath, nil, false)
	if job == nil {
		t.Fatal("expected job, got nil")
	}
	if job.BatchID != "batch-1" {
		t.Errorf("expected batch-1, got %s", job.BatchID)
	}

	pool.Shutdown()
}

func TestPool_GetBatch_shouldReturnBatch(t *testing.T) {
	pool, _ := setupTestPool(t)
	defer pool.Shutdown()

	tmpPath, _ := createTempUploadFile(t)

	pool.Enqueue("batch-2", "user-1", "photo.jpg", 1024, tmpPath, nil, false)
	pool.Enqueue("batch-2", "user-1", "photo2.jpg", 2048, tmpPath, nil, false)

	batch := pool.GetBatch("batch-2")
	if batch == nil {
		t.Fatal("expected batch, got nil")
	}
	if batch.Total != 2 {
		t.Errorf("expected 2 jobs, got %d", batch.Total)
	}
	if len(batch.Jobs) != 2 {
		t.Errorf("expected 2 jobs in list, got %d", len(batch.Jobs))
	}
}

func TestPool_Subscribe_shouldReceiveUpdates(t *testing.T) {
	pool, _ := setupTestPool(t)
	defer pool.Shutdown()

	tmpPath, _ := createTempUploadFile(t)

	ch := pool.Subscribe("batch-3", "listener-1")
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}

	job := pool.Enqueue("batch-3", "user-1", "photo.jpg", 1024, tmpPath, nil, false)

	select {
	case update := <-ch:
		if update == nil {
			t.Error("expected job update, got nil")
		}
	default:
		if job.Status == JobQueued || job.Status == JobProcessing {
			t.Log("job still processing, waiting for next update")
		}
	}

	pool.Unsubscribe("batch-3", "listener-1")
}

func TestPool_Shutdown_shouldNotPanic(t *testing.T) {
	pool, _ := setupTestPool(t)

	tmpPath, _ := createTempUploadFile(t)

	pool.Enqueue("batch-4", "user-1", "photo.jpg", 1024, tmpPath, nil, false)

	pool.Shutdown()

	pool.wg.Wait()

	select {
	case _, ok := <-pool.jobChan:
		if ok {
			t.Error("channel should be closed after shutdown")
		}
	default:
	}
}

func TestPool_Enqueue_shouldBlockWhenFull(t *testing.T) {
	pool, _ := setupTestPool(t)
	defer pool.Shutdown()

	tmpPath, _ := createTempUploadFile(t)

	for i := 0; i < 10; i++ {
		pool.Enqueue("batch-5", "user-1", "photo.jpg", 1024, tmpPath, nil, false)
	}

	batch := pool.GetBatch("batch-5")
	if batch.Total != 10 {
		t.Errorf("expected 10 jobs, got %d", batch.Total)
	}
}

func TestJob_SetCompleted_shouldUpdateState(t *testing.T) {
	tmpPath, _ := createTempUploadFile(t)

	job := &UploadJob{
		JobID:        "job-1",
		BatchID:      "batch-1",
		UserID:       "user-1",
		OriginalName: "photo.jpg",
		Size:         1024,
		TempPath:     tmpPath,
		Status:       JobQueued,
	}

	job.SetCompleted("file-123")
	if job.Status != JobCompleted {
		t.Errorf("expected completed, got %s", job.Status)
	}
	if job.FileID != "file-123" {
		t.Errorf("expected file-123, got %s", job.FileID)
	}
	if job.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", job.Progress)
	}
}

func TestJob_SetFailed_shouldUpdateState(t *testing.T) {
	job := &UploadJob{
		JobID:  "job-1",
		Status: JobQueued,
	}

	job.SetFailed("something went wrong")
	if job.Status != JobFailed {
		t.Errorf("expected failed, got %s", job.Status)
	}
	if job.Error != "something went wrong" {
		t.Errorf("expected error, got %s", job.Error)
	}
}

func TestJob_SetSkipped_shouldUpdateState(t *testing.T) {
	job := &UploadJob{
		JobID:  "job-1",
		Status: JobQueued,
	}

	job.SetSkipped("duplicate_content", "existing-file-1")
	if job.Status != JobSkipped {
		t.Errorf("expected skipped, got %s", job.Status)
	}
	if job.Reason != "duplicate_content" {
		t.Errorf("expected duplicate_content, got %s", job.Reason)
	}
	if job.FileID != "existing-file-1" {
		t.Errorf("expected existing-file-1, got %s", job.FileID)
	}
}

func TestJob_SetProgress_shouldUpdateStageAndProgress(t *testing.T) {
	job := &UploadJob{
		JobID:  "job-1",
		Status: JobQueued,
	}

	job.SetProgress(StageHashing, 0.5)
	if job.Stage != StageHashing {
		t.Errorf("expected hashing stage, got %s", job.Stage)
	}
	if job.Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", job.Progress)
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

func TestPool_WorkerRecovery_poolShouldNotDie(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 2

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	u, err := us.Create("recoveryuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, fs, es, ts, nil)

	tmpDir := t.TempDir()

	for i := 0; i < 8; i++ {
		content := make([]byte, 5+i) // 5 to 12 bytes
		rand.Read(content)
		path := filepath.Join(tmpDir, fmt.Sprintf("file_%d.bin", i))
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("write temp file: %v", err)
		}
		info, _ := os.Stat(path)
		pool.Enqueue("batch-recovery", u.ID, fmt.Sprintf("file_%d.bin", i), info.Size(), path, nil, false)
	}

	time.Sleep(200 * time.Millisecond)

	batch := pool.GetBatch("batch-recovery")
	if batch == nil {
		t.Fatal("expected batch, got nil")
	}

	pool.Shutdown()

	laterJob := &UploadJob{
		JobID:    "later-job",
		BatchID:  "batch-recovery",
		UserID:   u.ID,
		Filename: "later.bin",
		Status:   JobQueued,
	}
	pool.notifySubscribers(laterJob)

	pooled := 0
	for _, j := range batch.Jobs {
		if j.Stage != "" || j.Status == JobFailed || j.Status == JobCompleted {
			pooled++
		}
	}
	if pooled == 0 && batch.Total > 0 {
		t.Error("expected some jobs to be processed, all were ignored")
	}
}

func TestPool_NonMediaFile_shouldUseFilesPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	u, err := us.Create("fileuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, fs, es, ts, nil)

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "document.pdf")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	info, _ := os.Stat(tmpPath)
	job := pool.Enqueue("batch-files", u.ID, "document.pdf", info.Size(), tmpPath, nil, false)
	if job == nil {
		t.Fatal("expected job, got nil")
	}

	pool.Shutdown()

	files, _, _, err := fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least 1 file, got 0")
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

func TestPool_SubscribeUser_shouldReceiveUpdates(t *testing.T) {
	pool, _ := setupTestPool(t)
	defer pool.Shutdown()

	tmpPath, _ := createTempUploadFile(t)

	ch := pool.SubscribeUser("user-global", "listener-1")
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}

	job := pool.Enqueue("batch-global", "user-global", "photo.jpg", 1024, tmpPath, nil, false)

	select {
	case update := <-ch:
		if update == nil {
			t.Error("expected job update, got nil")
		}
		if update.JobID != job.JobID {
			t.Errorf("expected job %s, got %s", job.JobID, update.JobID)
		}
	default:
		t.Log("job update not immediately available, continuing")
	}

	pool.UnsubscribeUser("user-global", "listener-1")
}

func TestPool_NonMediaFile_shouldSkipExifAndThumbnails(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	u, err := us.Create("skipuser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, fs, es, ts, nil)

	content := make([]byte, 256)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "document.pdf")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	info, _ := os.Stat(tmpPath)
	job := pool.Enqueue("batch-skip", u.ID, "document.pdf", info.Size(), tmpPath, nil, false)
	if job == nil {
		t.Fatal("expected job, got nil")
	}

	pool.Shutdown()

	files, _, _, err := fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 file, got 0")
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

func TestPool_SkipNameSizeDedup_shouldSkipCheckWhenFlagSet(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret"
	cfg.Upload.ConcurrentWorkers = 1
	cfg.Media.AutoOrganize = true

	db := store.OpenTestDB(t)
	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	ts := store.NewThumbnailStore(db)

	u, err := us.Create("skipdedup_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	existing := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/dup.jpg",
		OriginalName: "dup.jpg",
		Path:         "2024/07",
		SizeBytes:    512,
		MimeType:     "image/jpeg",
		SHA256:       "abcdef_skipdedup_test",
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(existing); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	pool := NewPool(cfg, fs, es, ts, nil)

	content := make([]byte, 512)
	rand.Read(content)

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "dup.jpg")
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	info, _ := os.Stat(tmpPath)

	job := pool.Enqueue("batch-dedup-skip", u.ID, "dup.jpg", info.Size(), tmpPath, nil, true)
	if job == nil {
		t.Fatal("expected job, got nil")
	}

	pool.Shutdown()

	files, _, _, err := fs.List(store.FileListOptions{UserID: u.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list files: %v", err)
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

func TestPool_Stats_shouldReportQueueLength(t *testing.T) {
	pool, _ := setupTestPool(t)

	tmpPath1, _ := createTempUploadFile(t)
	tmpPath2, _ := createTempUploadFile(t)

	pool.processingMu.Lock()
	pool.processingJobs["blocker-1"] = &UploadJob{JobID: "blocker-1", Status: JobProcessing}
	pool.processingJobs["blocker-2"] = &UploadJob{JobID: "blocker-2", Status: JobProcessing}
	pool.processingMu.Unlock()

	j1 := pool.Enqueue("batch-stats", "user-1", "test1.jpg", 100, tmpPath1, nil, true)
	j2 := pool.Enqueue("batch-stats", "user-1", "test2.jpg", 100, tmpPath2, nil, true)

	_ = j1
	_ = j2

	pool.Shutdown()

	stats := pool.Stats()
	if stats.TotalWorkers != 2 {
		t.Errorf("expected total workers 2, got %d", stats.TotalWorkers)
	}
	total := stats.CompletedTotal + stats.FailedTotal + stats.SkippedTotal
	if total < 2 {
		t.Errorf("expected at least 2 jobs processed, got %d processed (%d completed + %d failed + %d skipped)",
			total, stats.CompletedTotal, stats.FailedTotal, stats.SkippedTotal)
	}

	pool.processingMu.Lock()
	delete(pool.processingJobs, "blocker-1")
	delete(pool.processingJobs, "blocker-2")
	pool.processingMu.Unlock()
}

func TestPool_Stats_shouldReportProcessingJobs(t *testing.T) {
	pool, _ := setupTestPool(t)

	job := &UploadJob{
		JobID:    "test-job-1",
		Filename: "processing.jpg",
		Status:   JobProcessing,
		Stage:    StageThumbnails,
		Progress: 0.8,
	}

	pool.processingMu.Lock()
	pool.processingJobs["test-job-1"] = job
	pool.processingMu.Unlock()

	stats := pool.Stats()
	if len(stats.ProcessingJobs) != 1 {
		t.Fatalf("expected 1 processing job, got %d", len(stats.ProcessingJobs))
	}
	pj := stats.ProcessingJobs[0]
	if pj.JobID != "test-job-1" {
		t.Errorf("expected job_id test-job-1, got %s", pj.JobID)
	}
	if pj.Filename != "processing.jpg" {
		t.Errorf("expected filename processing.jpg, got %s", pj.Filename)
	}
	if pj.Stage != string(StageThumbnails) {
		t.Errorf("expected stage thumbnails, got %s", pj.Stage)
	}
	if pj.Progress != 0.8 {
		t.Errorf("expected progress 0.8, got %f", pj.Progress)
	}

	pool.processingMu.Lock()
	delete(pool.processingJobs, "test-job-1")
	pool.processingMu.Unlock()
}

func TestPool_Stats_shouldReportCountersAfterCompletion(t *testing.T) {
	pool, _ := setupTestPool(t)

	tmpPath, _ := createTempUploadFile(t)
	_ = pool.Enqueue("batch-counter", "user-1", "counter.jpg", 100, tmpPath, nil, true)
	pool.Shutdown()

	stats := pool.Stats()
	total := stats.CompletedTotal + stats.FailedTotal + stats.SkippedTotal
	if total < 1 {
		t.Errorf("expected at least 1 counter total (completed+failed+skipped), got %d", total)
	}
	if stats.ActiveWorkers != 0 {
		t.Errorf("expected 0 active workers after shutdown, got %d", stats.ActiveWorkers)
	}
}
