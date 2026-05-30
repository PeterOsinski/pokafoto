package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

func setupUploadJobStore(t *testing.T) (*UploadJobStore, string, func()) {
	t.Helper()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	u, err := us.Create("jobuser_"+t.Name(), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return NewUploadJobStore(db), u.ID, func() {}
}

func createTempPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test_upload.bin")
	if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestUploadJobStore_Create_shouldInsertJob(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-1",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}

	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected job ID, got empty")
	}

	fetched, err := s.FindByID(job.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected job, got nil")
	}
	if fetched.Filename != "photo.jpg" {
		t.Errorf("expected photo.jpg, got %s", fetched.Filename)
	}
}

func TestUploadJobStore_Claim_shouldReturnAndLockJob(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-claim",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	claimed, err := s.Claim()
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected claimed job, got nil")
	}
	if claimed.ID != job.ID {
		t.Errorf("expected job %s, got %s", job.ID, claimed.ID)
	}
	if claimed.Status != model.JobStatusProcessing {
		t.Errorf("expected processing, got %s", claimed.Status)
	}

	second, err := s.Claim()
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if second != nil {
		t.Errorf("expected nil on empty queue, got %+v", second)
	}
}

func TestUploadJobStore_Claim_shouldReturnNilWhenQueueEmpty(t *testing.T) {
	s, _, _ := setupUploadJobStore(t)

	claimed, err := s.Claim()
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed != nil {
		t.Errorf("expected nil, got %+v", claimed)
	}
}

func TestUploadJobStore_Claim_shouldExhaustQueue(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	var jobIDs []string
	for i := 0; i < 3; i++ {
		job := &model.UploadJob{
			BatchID:   "batch-exhaust",
			UserID:    userID,
			Filename:  "photo.jpg",
			SizeBytes: 1024,
			TempPath:  tmpPath,
			Status:    model.JobStatusQueued,
		}
		if err := s.Create(job); err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
		jobIDs = append(jobIDs, job.ID)
	}

	var claimedIDs []string
	for i := 0; i < 3; i++ {
		job, _ := s.Claim()
		if job == nil {
			t.Fatalf("expected job %d, got nil", i)
		}
		claimedIDs = append(claimedIDs, job.ID)
	}

	for _, cid := range claimedIDs {
		found := false
		for _, jid := range jobIDs {
			if cid == jid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("claimed unknown job %s", cid)
		}
	}

	empty, _ := s.Claim()
	if empty != nil {
		t.Errorf("expected empty queue after consuming all jobs, got %+v", empty)
	}
}

func TestUploadJobStore_UpdateProgress_shouldUpdateStageAndPercent(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-progress",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := s.SetProcessing(job.ID); err != nil {
		t.Fatalf("set processing: %v", err)
	}

	if err := s.UpdateProgress(job.ID, model.JobStageHashing, 0.5); err != nil {
		t.Fatalf("update progress: %v", err)
	}

	fetched, _ := s.FindByID(job.ID)
	if fetched.Stage == nil || *fetched.Stage != model.JobStageHashing {
		t.Errorf("expected stage hashing, got %v", fetched.Stage)
	}
	if fetched.Progress != 0.5 {
		t.Errorf("expected progress 0.5, got %f", fetched.Progress)
	}
}

func TestUploadJobStore_Complete_shouldSetStatusAndFileID(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-complete",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	s.SetProcessing(job.ID)

	if err := s.Complete(job.ID, "file-uuid-123"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	fetched, _ := s.FindByID(job.ID)
	if fetched.Status != model.JobStatusCompleted {
		t.Errorf("expected completed, got %s", fetched.Status)
	}
	if fetched.FileID == nil || *fetched.FileID != "file-uuid-123" {
		t.Errorf("expected file_id file-uuid-123, got %v", fetched.FileID)
	}
	if fetched.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", fetched.Progress)
	}
}

func TestUploadJobStore_Fail_shouldSetStatusAndError(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-fail",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	s.SetProcessing(job.ID)

	if err := s.Fail(job.ID, "something went wrong"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	fetched, _ := s.FindByID(job.ID)
	if fetched.Status != model.JobStatusFailed {
		t.Errorf("expected failed, got %s", fetched.Status)
	}
	if fetched.Error == nil || *fetched.Error != "something went wrong" {
		t.Errorf("expected error 'something went wrong', got %v", fetched.Error)
	}
}

func TestUploadJobStore_Skip_shouldSetStatusAndReason(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-skip",
		UserID:    userID,
		Filename:  "photo.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	s.SetProcessing(job.ID)

	if err := s.Skip(job.ID, "duplicate_content", "existing-file-1"); err != nil {
		t.Fatalf("skip: %v", err)
	}

	fetched, _ := s.FindByID(job.ID)
	if fetched.Status != model.JobStatusSkipped {
		t.Errorf("expected skipped, got %s", fetched.Status)
	}
	if fetched.Reason == nil || *fetched.Reason != "duplicate_content" {
		t.Errorf("expected reason 'duplicate_content', got %v", fetched.Reason)
	}
	if fetched.FileID == nil || *fetched.FileID != "existing-file-1" {
		t.Errorf("expected file_id existing-file-1, got %v", fetched.FileID)
	}
	if fetched.Progress != 1.0 {
		t.Errorf("expected progress 1.0, got %f", fetched.Progress)
	}
}

func TestUploadJobStore_ListByBatch_shouldReturnAllJobsForBatch(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	for i := 0; i < 3; i++ {
		job := &model.UploadJob{
			BatchID:   "batch-list",
			UserID:    userID,
			Filename:  "photo.jpg",
			SizeBytes: 1024,
			TempPath:  tmpPath,
			Status:    model.JobStatusQueued,
		}
		if err := s.Create(job); err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
	}

	jobs, err := s.ListByBatch("batch-list")
	if err != nil {
		t.Fatalf("list by batch: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestUploadJobStore_ListByBatch_shouldReturnEmptyForUnknownBatch(t *testing.T) {
	s, _, _ := setupUploadJobStore(t)

	jobs, err := s.ListByBatch("unknown-batch")
	if err != nil {
		t.Fatalf("list by batch: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestUploadJobStore_RecoverStuckJobs_shouldResetProcessingJobs(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	j1 := &model.UploadJob{
		BatchID:   "batch-recover",
		UserID:    userID,
		Filename:  "photo1.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	j2 := &model.UploadJob{
		BatchID:   "batch-recover",
		UserID:    userID,
		Filename:  "photo2.jpg",
		SizeBytes: 2048,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	if err := s.Create(j1); err != nil {
		t.Fatalf("create j1: %v", err)
	}
	s.SetProcessing(j1.ID)
	if err := s.Create(j2); err != nil {
		t.Fatalf("create j2: %v", err)
	}

	count, err := s.RecoverStuckJobs()
	if err != nil {
		t.Fatalf("recover stuck jobs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 recovered job, got %d", count)
	}

	f1, _ := s.FindByID(j1.ID)
	if f1.Status != model.JobStatusQueued {
		t.Errorf("expected j1 status queued, got %s", f1.Status)
	}

	f2, _ := s.FindByID(j2.ID)
	if f2.Status != model.JobStatusQueued {
		t.Errorf("expected j2 status queued, got %s", f2.Status)
	}
}

func TestUploadJobStore_FindByID_shouldReturnNilForUnknownID(t *testing.T) {
	s, _, _ := setupUploadJobStore(t)

	job, err := s.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if job != nil {
		t.Errorf("expected nil, got %+v", job)
	}
}

func TestUploadJobStore_Create_shouldStoreSkipNameSizeDedup(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:           "batch-dedup",
		UserID:            userID,
		Filename:          "photo.jpg",
		SizeBytes:         1024,
		TempPath:          tmpPath,
		SkipNameSizeDedup: true,
		Status:            model.JobStatusQueued,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fetched, _ := s.FindByID(job.ID)
	if !fetched.SkipNameSizeDedup {
		t.Error("expected skip_name_size_dedup true")
	}
}

func TestUploadJobStore_ListActiveByUser_shouldReturnActiveAndRecentJobs(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	j1 := &model.UploadJob{
		BatchID:   "batch-active",
		UserID:    userID,
		Filename:  "queued.jpg",
		SizeBytes: 100,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	j2 := &model.UploadJob{
		BatchID:   "batch-active",
		UserID:    userID,
		Filename:  "processing.jpg",
		SizeBytes: 200,
		TempPath:  tmpPath,
		Status:    model.JobStatusProcessing,
	}
	j3 := &model.UploadJob{
		BatchID:   "batch-active",
		UserID:    userID,
		Filename:  "completed.jpg",
		SizeBytes: 300,
		TempPath:  tmpPath,
		Status:    model.JobStatusCompleted,
	}
	if err := s.Create(j1); err != nil {
		t.Fatalf("create j1: %v", err)
	}
	if err := s.Create(j2); err != nil {
		t.Fatalf("create j2: %v", err)
	}
	s.SetProcessing(j2.ID)
	if err := s.Create(j3); err != nil {
		t.Fatalf("create j3: %v", err)
	}
	s.Complete(j3.ID, "file-1")

	jobs, err := s.ListActiveByUser(userID)
	if err != nil {
		t.Fatalf("list active by user: %v", err)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 active/recent jobs, got %d", len(jobs))
	}

	found := make(map[string]bool)
	for _, j := range jobs {
		found[j.Filename] = true
	}
	if !found["queued.jpg"] {
		t.Error("expected queued.jpg in results")
	}
	if !found["processing.jpg"] {
		t.Error("expected processing.jpg in results")
	}
	if !found["completed.jpg"] {
		t.Error("expected completed.jpg in results (within 1 hour)")
	}
}

func TestUploadJobStore_ListActiveByUser_shouldExcludeOtherUserJobs(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	s := NewUploadJobStore(db)

	user1, _ := us.Create("user1_"+t.Name(), "password123", model.RoleMember, nil)
	user2, _ := us.Create("user2_"+t.Name(), "password123", model.RoleMember, nil)

	tmpPath := createTempPath(t)

	j1 := &model.UploadJob{
		BatchID:   "batch-active",
		UserID:    user1.ID,
		Filename:  "mine.jpg",
		SizeBytes: 100,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	j2 := &model.UploadJob{
		BatchID:   "batch-active",
		UserID:    user2.ID,
		Filename:  "theirs.jpg",
		SizeBytes: 200,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	if err := s.Create(j1); err != nil {
		t.Fatalf("create j1: %v", err)
	}
	if err := s.Create(j2); err != nil {
		t.Fatalf("create j2: %v", err)
	}

	jobs, err := s.ListActiveByUser(user1.ID)
	if err != nil {
		t.Fatalf("list active by user: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 job for user1, got %d", len(jobs))
	}
	if jobs[0].Filename != "mine.jpg" {
		t.Errorf("expected mine.jpg, got %s", jobs[0].Filename)
	}
}

func TestUploadJobStore_ListActiveByUser_shouldExcludeOldCompletedJobs(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	j := &model.UploadJob{
		BatchID:   "batch-old",
		UserID:    userID,
		Filename:  "old.jpg",
		SizeBytes: 100,
		TempPath:  tmpPath,
		Status:    model.JobStatusCompleted,
	}
	if err := s.Create(j); err != nil {
		t.Fatalf("create job: %v", err)
	}
	s.Complete(j.ID, "file-old")

	s.db.Exec(
		`UPDATE upload_jobs SET updated_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-2*time.Hour).Format(time.RFC3339),
		j.ID,
	)

	jobs, err := s.ListActiveByUser(userID)
	if err != nil {
		t.Fatalf("list active by user: %v", err)
	}

	for _, job := range jobs {
		if job.ID == j.ID {
			t.Error("expected old completed job to be excluded")
		}
	}
}

func TestUploadJobStore_ListActiveByUser_shouldReturnEmptyForUnknownUser(t *testing.T) {
	s, _, _ := setupUploadJobStore(t)

	jobs, err := s.ListActiveByUser("nonexistent-user")
	if err != nil {
		t.Fatalf("list active by user: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestUploadJobStore_ListAll_shouldPaginate(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	for i := 0; i < 5; i++ {
		job := &model.UploadJob{
			BatchID:   "batch-list-paginate",
			UserID:    userID,
			Filename:  "photo" + string(rune('a'+i)) + ".jpg",
			SizeBytes: int64(100 + i),
			TempPath:  tmpPath,
			Status:    model.JobStatusCompleted,
		}
		if err := s.Create(job); err != nil {
			t.Fatalf("create job: %v", err)
		}
	}

	jobs, total, err := s.ListAll(3, 0, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestUploadJobStore_ListAll_shouldFilterByStatus(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	for i := 0; i < 3; i++ {
		job := &model.UploadJob{
			BatchID:   "batch-filter-status",
			UserID:    userID,
			Filename:  "photo" + string(rune('a'+i)) + ".jpg",
			SizeBytes: 100,
			TempPath:  tmpPath,
			Status:    model.JobStatusCompleted,
		}
		if err := s.Create(job); err != nil {
			t.Fatalf("create job: %v", err)
		}
	}
	failedJob := &model.UploadJob{
		BatchID:   "batch-filter-failed",
		UserID:    userID,
		Filename:  "bad.jpg",
		SizeBytes: 50,
		TempPath:  tmpPath,
		Status:    model.JobStatusFailed,
	}
	if err := s.Create(failedJob); err != nil {
		t.Fatalf("create failed job: %v", err)
	}
	s.Fail(failedJob.ID, "test_error")

	jobs, total, err := s.ListAll(10, 0, string(model.JobStatusFailed))
	if err != nil {
		t.Fatalf("list all with filter: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for failed filter, got %d", total)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != model.JobStatusFailed {
		t.Errorf("expected failed status, got %s", jobs[0].Status)
	}
}

func TestUploadJobStore_CountByStatus_shouldReturnAllStatuses(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	for _, status := range []model.JobStatus{model.JobStatusCompleted, model.JobStatusCompleted, model.JobStatusFailed} {
		job := &model.UploadJob{
			BatchID:   "batch-cnt-status",
			UserID:    userID,
			Filename:  "file.jpg",
			SizeBytes: 100,
			TempPath:  tmpPath,
			Status:    status,
		}
		if err := s.Create(job); err != nil {
			t.Fatalf("create job: %v", err)
		}
	}

	counts, err := s.CountByStatus()
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	if counts["completed"] != 2 {
		t.Errorf("expected 2 completed, got %d", counts["completed"])
	}
	if counts["failed"] != 1 {
		t.Errorf("expected 1 failed, got %d", counts["failed"])
	}
	if counts["queued"] != 0 {
		t.Errorf("expected 0 queued, got %d", counts["queued"])
	}
}

func TestUploadJobStore_Requeue_shouldResetToQueued(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-requeue",
		UserID:    userID,
		Filename:  "failed.jpg",
		SizeBytes: 100,
		TempPath:  tmpPath,
		Status:    model.JobStatusFailed,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	s.Fail(job.ID, "test_error")

	if err := s.Requeue(job.ID); err != nil {
		t.Fatalf("requeue failed: %v", err)
	}

	fetched, err := s.FindByID(job.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if fetched.Status != model.JobStatusQueued {
		t.Errorf("expected queued after requeue, got %s", fetched.Status)
	}
	if fetched.Error != nil {
		t.Errorf("expected nil error after requeue, got %v", fetched.Error)
	}
}

func TestUploadJobStore_Requeue_shouldFailForNonFailedJob(t *testing.T) {
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-nonretry",
		UserID:    userID,
		Filename:  "completed.jpg",
		SizeBytes: 100,
		TempPath:  tmpPath,
		Status:    model.JobStatusCompleted,
	}
	if err := s.Create(job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := s.Requeue(job.ID); err == nil {
		t.Error("expected error requeueing completed job, got nil")
	}
}
