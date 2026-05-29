package worker

import (
	"crypto/rand"
	"crypto/sha256"
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

	u, err := us.Create("workeruser_"+strings.ReplaceAll(t.Name(), "/", "_"), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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

	pool := NewPool(cfg, fs, es, ts, nil, ujs)
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
