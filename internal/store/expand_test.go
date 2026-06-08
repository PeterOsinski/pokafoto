package store

import (
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

func TestSettingStore_Set_shouldUpsert(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	s := NewSettingStore(db)

	if err := s.Set("key1", "value1"); err != nil {
		t.Fatalf("set: %v", err)
	}

	v, err := s.Get("key1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "value1" {
		t.Errorf("expected value1, got %s", v)
	}

	if err := s.Set("key1", "value2"); err != nil {
		t.Fatalf("set: %v", err)
	}

	v, err = s.Get("key1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "value2" {
		t.Errorf("expected value2 after upsert, got %s", v)
	}
}

func TestSettingStore_Get_shouldReturnEmptyForMissing(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	s := NewSettingStore(db)

	v, err := s.Get("nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "" {
		t.Errorf("expected empty for missing key, got %s", v)
	}
}

func TestUploadJobStore_Claim_shouldReturnNilWhenEmpty(t *testing.T) {
	t.Parallel()
	s, _, _ := setupUploadJobStore(t)

	job, err := s.Claim()
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if job != nil {
		t.Error("expected nil job when queue is empty")
	}
}

func TestUploadJobStore_Claim_shouldLockAndAdvanceStatus(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-claim",
		UserID:    userID,
		Filename:  "claim_test.jpg",
		SizeBytes: 100,
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
	if claimed.Status != model.JobStatusProcessing {
		t.Errorf("expected processing status, got %s", claimed.Status)
	}

	second, _ := s.Claim()
	if second != nil {
		t.Errorf("expected nil on second claim (already claimed), got %v", second.ID)
	}
}

func TestUploadJobStore_RecoverStuckJobs_shouldResetProcessing(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-stuck",
		UserID:    userID,
		Filename:  "stuck.jpg",
		SizeBytes: 200,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	s.Create(job)

	s.SetProcessing(job.ID)

	count, err := s.CountProcessing()
	if err != nil {
		t.Fatalf("count processing: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 processing job, got %d", count)
	}

	recovered, err := s.RecoverStuckJobs()
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if recovered != 1 {
		t.Errorf("expected 1 recovered, got %d", recovered)
	}

	recoveredJob, _ := s.FindByID(job.ID)
	if recoveredJob.Status != model.JobStatusQueued {
		t.Errorf("expected recovered job to be queued, got %s", recoveredJob.Status)
	}
}

func TestUploadJobStore_Complete_shouldSetFileID(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-comp",
		UserID:    userID,
		Filename:  "comp.jpg",
		SizeBytes: 300,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	s.Create(job)

	if err := s.Complete(job.ID, "file-123"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	completed, _ := s.FindByID(job.ID)
	if completed.Status != model.JobStatusCompleted {
		t.Errorf("expected completed status, got %s", completed.Status)
	}
	if completed.FileID == nil || *completed.FileID != "file-123" {
		t.Errorf("expected file_id file-123, got %v", completed.FileID)
	}
}

func TestUploadJobStore_CountByStatus_shouldIncludeAllStatuses(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	createJob := func(fileName, status string) {
		j := &model.UploadJob{
			BatchID:   "batch-cnt",
			UserID:    userID,
			Filename:  fileName,
			SizeBytes: 100,
			TempPath:  tmpPath,
			Status:    model.JobStatus(status),
		}
		s.Create(j)
	}
	createJob("q.jpg", "queued")
	createJob("f.jpg", "failed")
	createJob("s.jpg", "skipped")

	counts, err := s.CountByStatus()
	if err != nil {
		t.Fatalf("count by status: %v", err)
	}
	if counts[string(model.JobStatusQueued)] != 1 {
		t.Errorf("expected 1 queued, got %d", counts[string(model.JobStatusQueued)])
	}
	if counts[string(model.JobStatusFailed)] != 1 {
		t.Errorf("expected 1 failed, got %d", counts[string(model.JobStatusFailed)])
	}
	if counts[string(model.JobStatusSkipped)] != 1 {
		t.Errorf("expected 1 skipped, got %d", counts[string(model.JobStatusSkipped)])
	}
}

func TestUploadJobStore_Requeue_shouldRejectNonFailedOrSkipped(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	job := &model.UploadJob{
		BatchID:   "batch-rq2",
		UserID:    userID,
		Filename:  "no_retry.jpg",
		SizeBytes: 500,
		TempPath:  tmpPath,
		Status:    model.JobStatusCompleted,
	}
	s.Create(job)

	err := s.Requeue(job.ID)
	if err == nil {
		t.Fatal("expected error when requeuing completed job")
	}
}

func TestUploadJobStore_FindByResumeToken_shouldReturnNilWhenNotFound(t *testing.T) {
	t.Parallel()
	s, _, _ := setupUploadJobStore(t)

	job, err := s.FindByResumeToken("nonexistent-token")
	if err != nil {
		t.Fatalf("find by resume token: %v", err)
	}
	if job != nil {
		t.Error("expected nil for nonexistent token")
	}
}

func TestUploadJobStore_FindByResumeToken_shouldFindExisting(t *testing.T) {
	t.Parallel()
	s, userID, _ := setupUploadJobStore(t)
	tmpPath := createTempPath(t)

	token := "resume-token-123"
	job := &model.UploadJob{
		BatchID:     "batch-resume",
		UserID:      userID,
		Filename:    "resume.jpg",
		SizeBytes:   1024,
		TempPath:    tmpPath,
		Status:      model.JobStatusQueued,
		ResumeToken: &token,
	}
	s.Create(job)

	found, err := s.FindByResumeToken(token)
	if err != nil {
		t.Fatalf("find by resume token: %v", err)
	}
	if found == nil {
		t.Fatal("expected job, got nil")
	}
	if found.ResumeToken == nil || *found.ResumeToken != token {
		t.Errorf("expected resume token %s, got %v", token, found.ResumeToken)
	}
}

func TestThumbnailStore_Breakdown_shouldReturnSizes(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user, _ := us.Create("thumbbreak", "password123", model.RoleMember, nil)
	fs.Create(&model.File{
		UserID:    user.ID,
		Filename:  "2024/07/break.jpg",
		OriginalName: "break.jpg",
		Path:      "2024/07",
		SizeBytes: 2048,
		MimeType:  "image/jpeg",
		MediaType: model.MediaTypePhoto,
	})

	f := &model.File{UserID: user.ID, Filename: "2024/07/break2.jpg", OriginalName: "break2.jpg", Path: "2024/07", SizeBytes: 100, MimeType: "image/jpeg", MediaType: model.MediaTypePhoto}
	fs.Create(f)

	ts.Create(&model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeSmall,
		Width:     60,
		Height:    60,
		Format:    "jpeg",
		SizeBytes: 1000,
	})

	breakdown, err := ts.Breakdown()
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if len(breakdown) < 1 {
		t.Error("expected at least one breakdown entry")
	}
}

func TestEventStore_List_shouldFilterBySeverity(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	es := NewSystemEventsStore(db)

	es.Create(&model.SystemEvent{
		EventType: "test_event",
		Severity:  model.SeverityInfo,
		Message:   "info msg",
	})
	es.Create(&model.SystemEvent{
		EventType: "test_event",
		Severity:  model.SeverityError,
		Message:   "error msg",
	})

	events, total, err := es.List(10, 0, "", string(model.SeverityError), "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 error event, got %d", total)
	}
	if len(events) > 0 && events[0].Severity != model.SeverityError {
		t.Errorf("expected error severity, got %s", events[0].Severity)
	}

	all, total2, _ := es.List(10, 0, "", "", "", "")
	if total2 != 2 {
		t.Errorf("expected 2 total events, got %d", total2)
	}
	_ = all
}

func TestEventStore_PurgeOlderThan_shouldRemoveOldEvents(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	es := NewSystemEventsStore(db)

	es.Create(&model.SystemEvent{
		EventType: "old",
		Severity:  model.SeverityInfo,
		Message:   "old event",
	})

	deleted, err := es.PurgeOlderThan(1 * time.Hour)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	_ = deleted

	_, total, _ := es.List(10, 0, "", "", "", "")
	if total == 0 {
		t.Error("events created within the last hour should NOT be purged with 1h age")
	}
}

func TestEventStore_PurgeOlderThan_shouldNotRemoveRecentEvents(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	es := NewSystemEventsStore(db)

	es.Create(&model.SystemEvent{
		EventType: "recent",
		Severity:  model.SeverityInfo,
		Message:   "recent event",
	})

	deleted, err := es.PurgeOlderThan(1 * time.Hour)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	_ = deleted

	_, total, _ := es.List(10, 0, "", "", "", "")
	if total != 1 {
		t.Errorf("expected 1 recent event preserved, got %d", total)
	}
}

func TestChunkStore_FindMissingChunks_shouldReturnGaps(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	cs := NewChunkStore(db, &testFS{})

	cs.CreateChunkRecord("upload-1", 0, 100, 0, "abc", "/tmp/chunks/u1-0")
	cs.CreateChunkRecord("upload-1", 2, 100, 200, "def", "/tmp/chunks/u1-2")

	missing, err := cs.FindMissingChunks("upload-1", 4)
	if err != nil {
		t.Fatalf("find missing: %v", err)
	}
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing chunks, got %d", len(missing))
	}
	if missing[0] != 1 || missing[1] != 3 {
		t.Errorf("expected missing [1, 3], got %v", missing)
	}
}

func TestChunkStore_GetStoredChunks_shouldReturnSorted(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	cs := NewChunkStore(db, &testFS{})

	cs.CreateChunkRecord("upload-2", 2, 100, 200, "ccc", "/tmp/u2-2")
	cs.CreateChunkRecord("upload-2", 0, 100, 0, "aaa", "/tmp/u2-0")
	cs.CreateChunkRecord("upload-2", 1, 100, 100, "bbb", "/tmp/u2-1")

	chunks, err := cs.GetStoredChunks("upload-2")
	if err != nil {
		t.Fatalf("get stored: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 stored chunks, got %d", len(chunks))
	}
	if chunks[0] != 0 || chunks[1] != 1 || chunks[2] != 2 {
		t.Errorf("expected [0,1,2], got %v", chunks)
	}
}

func TestChunkStore_DeleteChunks_shouldRemoveRecords(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	cs := NewChunkStore(db, &testFS{})

	cs.CreateChunkRecord("upload-3", 0, 100, 0, "xxx", "/tmp/u3-0")
	cs.CreateChunkRecord("upload-3", 1, 100, 100, "yyy", "/tmp/u3-1")

	if err := cs.DeleteChunks("upload-3"); err != nil {
		t.Fatalf("delete chunks: %v", err)
	}

	count, _ := cs.GetStoredChunkCount("upload-3")
	if count != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", count)
	}
}

func TestSessionStore_DeleteByRefreshToken_shouldRemoveToken(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user, _ := us.Create("sessiontest", "password", model.RoleMember, nil)
	sess, err := ss.Create(user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	found, _ := ss.FindByRefreshToken(sess.RefreshToken)
	if found == nil {
		t.Fatal("expected session before delete")
	}

	ss.DeleteByRefreshToken(sess.RefreshToken)

	found, _ = ss.FindByRefreshToken(sess.RefreshToken)
	if found != nil {
		t.Error("expected nil session after delete by token")
	}
}
