package server

import (
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestFolders_CreateFolder_shouldCreate(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("foldcreate_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/folders", `{"name":"Test Folder"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFolders_ListFolders_shouldReturnFolders(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFolderStore(db)
	u, _ := us.Create("foldlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	fs.Create(u.ID, "Folder A", nil)
	fs.Create(u.ID, "Folder B", nil)

	w := testRequest(t, srv, "GET", "/api/v1/folders", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchOps_SoftDelete_shouldDeleteMultiple(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("batchsoft_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f1 := createTestFileForHandler(t, fs, u.ID, "batch1.jpg")
	f2 := createTestFileForHandler(t, fs, u.ID, "batch2.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/files/batch-delete", `{"ids":["`+f1.ID+`","`+f2.ID+`"]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchOps_Move_shouldMoveToFolder(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	folderStore := store.NewFolderStore(db)
	u, _ := us.Create("batchmove_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "to-move.jpg")
	folder, _ := folderStore.Create(u.ID, "Dest", nil)

	w := testRequest(t, srv, "POST", "/api/v1/files/batch-move", `{"ids":["`+f.ID+`"],"folder_id":"`+folder.ID+`"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchOps_Copy_shouldCopyFiles(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("batchcopy_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "source.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/files/batch-copy", `{"ids":["`+f.ID+`"]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTrash_BatchPermanentDelete_shouldDeleteFiles(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("trashbatch_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f1 := createTestFileForHandler(t, fs, u.ID, "trash1.jpg")
	f2 := createTestFileForHandler(t, fs, u.ID, "trash2.jpg")
	fs.SoftDelete(f1.ID)
	fs.SoftDelete(f2.ID)

	w := testRequest(t, srv, "POST", "/api/v1/trash/batch-permanent-delete", `{"ids":["`+f1.ID+`","`+f2.ID+`"]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdmin_RetryJob_shouldRequeueFailedJob(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("adminretry_"+uuid.NewString()[:8], "password123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "admin")

	f := createTestFileForHandler(t, fs, u.ID, "retry-photo.jpg")
	job := &model.UploadJob{
		BatchID:   "admin-retry",
		UserID:    u.ID,
		Filename:  "retry-photo.jpg",
		SizeBytes: f.SizeBytes,
		Status:    model.JobStatusFailed,
	}
	ujs.Create(job)
	ujs.Fail(job.ID, "test error")

	w := testRequest(t, srv, "POST", "/api/v1/admin/jobs/"+job.ID+"/retry", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdmin_TriggerBackup_shouldReturnUnavailable(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("adminbackup_"+uuid.NewString()[:8], "password123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "admin")

	w := testRequest(t, srv, "POST", "/api/v1/admin/backup", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
