package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

func setupChunkStore(t *testing.T) (*ChunkStore, *DB, string, func()) {
	t.Helper()
	db := OpenTestDB(t)
	cs := NewChunkStore(db)
	us := NewUserStore(db)

	user, err := us.Create("chunkuser_"+t.Name(), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	ujs := NewUploadJobStore(db)
	totalChunks := 3
	chunkSize := int64(1024)
	job := &model.UploadJob{
		BatchID:    "chunk-batch-" + t.Name(),
		UserID:     user.ID,
		Filename:   "test.dat",
		SizeBytes:  chunkSize * int64(totalChunks),
		TempPath:   os.TempDir(),
		Status:     model.JobStatusQueued,
		UploadMode: model.UploadModeChunked,
		TotalChunks: &totalChunks,
		ChunkSize:  &chunkSize,
	}
	if err := ujs.Create(job); err != nil {
		t.Fatalf("create upload job: %v", err)
	}

	return cs, db, job.ID, func() {}
}

func TestChunkStore_CreateChunkRecord_shouldPersist(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	if err := cs.CreateChunkRecord(uploadID, 0, 1024, 0, "abc123", "/tmp/chunk_0"); err != nil {
		t.Fatalf("create chunk record: %v", err)
	}

	count, err := cs.GetStoredChunkCount(uploadID)
	if err != nil {
		t.Fatalf("get stored chunk count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 stored chunk, got %d", count)
	}
}

func TestChunkStore_GetStoredChunks_shouldReturnSortedIndices(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	cs.CreateChunkRecord(uploadID, 2, 512, 1024, "sha2", "/tmp/c2")
	cs.CreateChunkRecord(uploadID, 0, 512, 0, "sha0", "/tmp/c0")
	cs.CreateChunkRecord(uploadID, 1, 512, 512, "sha1", "/tmp/c1")

	stored, err := cs.GetStoredChunks(uploadID)
	if err != nil {
		t.Fatalf("get stored chunks: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored chunks, got %d", len(stored))
	}
	if stored[0] != 0 || stored[1] != 1 || stored[2] != 2 {
		t.Errorf("expected sorted indices [0,1,2], got %v", stored)
	}
}

func TestChunkStore_FindMissingChunks_shouldReturnMissingIndices(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	cs.CreateChunkRecord(uploadID, 0, 512, 0, "", "/tmp/c0")
	cs.CreateChunkRecord(uploadID, 2, 512, 1024, "", "/tmp/c2")

	missing, err := cs.FindMissingChunks(uploadID, 3)
	if err != nil {
		t.Fatalf("find missing chunks: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing chunk, got %d: %v", len(missing), missing)
	}
	if missing[0] != 1 {
		t.Errorf("expected chunk 1 missing, got %d", missing[0])
	}
}

func TestChunkStore_GetChunkPath_shouldReturnPath(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	cs.CreateChunkRecord(uploadID, 0, 512, 0, "", "/tmp/my_chunk")

	path, err := cs.GetChunkPath(uploadID, 0)
	if err != nil {
		t.Fatalf("get chunk path: %v", err)
	}
	if path != "/tmp/my_chunk" {
		t.Errorf("expected /tmp/my_chunk, got %s", path)
	}
}

func TestChunkStore_GetChunkPath_shouldReturnEmptyForMissing(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	path, err := cs.GetChunkPath(uploadID, 99)
	if err != nil {
		t.Fatalf("get chunk path: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %s", path)
	}
}

func TestChunkStore_AssembleFile_shouldComputeSHA256(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	content := []byte("hello world chunked data")
	chunk1 := content[:5]
	chunk2 := content[5:]

	dir := t.TempDir()
	chunk1Path := filepath.Join(dir, "chunk_0")
	chunk2Path := filepath.Join(dir, "chunk_1")
	os.WriteFile(chunk1Path, chunk1, 0644)
	os.WriteFile(chunk2Path, chunk2, 0644)

	cs.CreateChunkRecord(uploadID, 0, int64(len(chunk1)), 0, "", chunk1Path)
	cs.CreateChunkRecord(uploadID, 1, int64(len(chunk2)), int64(len(chunk1)), "", chunk2Path)

	destPath := filepath.Join(dir, "assembled.dat")
	sha256Hex, err := cs.AssembleFile(uploadID, 2, destPath)
	if err != nil {
		t.Fatalf("assemble file: %v", err)
	}

	assembled, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read assembled file: %v", err)
	}
	if string(assembled) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(assembled))
	}
	if sha256Hex == "" {
		t.Error("expected non-empty SHA256 hash")
	}
}

func TestChunkStore_AssembleFile_shouldFailMissingChunk(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	dir := t.TempDir()
	chunkPath := filepath.Join(dir, "chunk_0")
	os.WriteFile(chunkPath, []byte("abc"), 0644)
	cs.CreateChunkRecord(uploadID, 0, 3, 0, "", chunkPath)

	destPath := filepath.Join(dir, "assembled.dat")
	_, err := cs.AssembleFile(uploadID, 3, destPath)
	if err == nil {
		t.Fatal("expected error for missing chunk, got nil")
	}
}

func TestChunkStore_DeleteChunks_shouldRemoveRecordsAndFiles(t *testing.T) {
	cs, _, uploadID, _ := setupChunkStore(t)

	dir := t.TempDir()
	chunkPath := filepath.Join(dir, "chunk_0")
	os.WriteFile(chunkPath, []byte("abc"), 0644)
	cs.CreateChunkRecord(uploadID, 0, 3, 0, "", chunkPath)

	if err := cs.DeleteChunks(uploadID); err != nil {
		t.Fatalf("delete chunks: %v", err)
	}

	count, err := cs.GetStoredChunkCount(uploadID)
	if err != nil {
		t.Fatalf("get stored chunk count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 chunks after delete, got %d", count)
	}

	if _, err := os.Stat(chunkPath); !os.IsNotExist(err) {
		t.Error("expected chunk file to be deleted")
	}
}

func TestChunkStore_DeleteAbandonedChunks_shouldCleanOld(t *testing.T) {
	cs, db, uploadID, _ := setupChunkStore(t)

	dir := t.TempDir()
	chunkPath := filepath.Join(dir, "chunk_0")
	os.WriteFile(chunkPath, []byte("old"), 0644)
	cs.CreateChunkRecord(uploadID, 0, 3, 0, "", chunkPath)

	db.Exec(`UPDATE upload_jobs SET status = 'completed' WHERE id = ?`, uploadID)

	time.Sleep(2 * time.Second)

	deleted, err := cs.DeleteAbandonedChunks(0)
	if err != nil {
		t.Fatalf("delete abandoned chunks: %v", err)
	}
	if deleted == 0 {
		t.Error("expected abandoned chunks to be deleted")
	}
}
