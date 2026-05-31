package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func multipartUploadBody(t *testing.T, files map[string][]byte) (string, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for name, data := range files {
		part, err := w.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("create form file %s: %v", name, err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("write form file %s: %v", name, err)
		}
	}
	boundary := w.Boundary()
	w.Close()
	return buf.String(), boundary
}

func TestDownload_shouldReturn404WhenFileNotOnDisk(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("download_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/nonexistent.jpg",
		OriginalName: "nonexistent.jpg",
		Path:         "/2024",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("download"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/download/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent file on disk, got %d", w.Code)
	}
}

func TestDownload_shouldRejectUnauthenticated(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/download/some-id", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated, got %d", w.Code)
	}
}

func TestBatchDownload_shouldReturnBadRequestWhenEmpty(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "POST", "/api/v1/download/batch", "{}", map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdmin_ListUsers_shouldReturnUsers(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	us.Create("adminlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	us.Create("adminlist2_"+uuid.NewString()[:8], "password123", model.RoleAdmin, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "admin-user", "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/users", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	users := resp["users"].([]interface{})
	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}
}

func TestAdmin_DeleteUser_shouldReturn204(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("admindelete_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "admin-user", "admin")

	w := testRequest(t, srv, "DELETE", "/api/v1/admin/users/"+u.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	exists, _ := us.FindByID(u.ID)
	if exists != nil {
		t.Error("user still exists after admin delete")
	}
}

func TestAdmin_UpdateRole_shouldChangeRole(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("adminrole_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "admin-user", "admin")

	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+u.ID+"/role", `{"role":"admin"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	updated, _ := us.FindByID(u.ID)
	if updated.Role != model.RoleAdmin {
		t.Errorf("expected admin role, got %s", updated.Role)
	}
}

func TestAdmin_UpdateRole_shouldRejectInvalidRole(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("adminbadrole_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "admin-user", "admin")

	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+u.ID+"/role", `{"role":"superadmin"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdmin_Endpoints_shouldRejectNonAdmin(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "GET", "/api/v1/admin/users", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestUpload_shouldRejectUnauthenticated(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/upload", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUploadCheck_shouldReturnDuplicates(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("dupcheck_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/existing.jpg",
		OriginalName: "existing.jpg",
		Path:         "2024/07",
		SizeBytes:    2048,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("existing"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	body := `[{"filename":"existing.jpg","size":2048},{"filename":"new.jpg","size":4096}]`
	w := testRequest(t, srv, "POST", "/api/v1/upload/check", body, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	dupes := resp["duplicates"].([]interface{})
	if len(dupes) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(dupes))
	}

	d := dupes[0].(map[string]interface{})
	if d["filename"] != "existing.jpg" {
		t.Errorf("expected existing.jpg, got %v", d["filename"])
	}
}

func TestUploadCheck_shouldReturnEmptyWhenNoDuplicates(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	body := `[{"filename":"brand_new.jpg","size":1024}]`
	w := testRequest(t, srv, "POST", "/api/v1/upload/check", body, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	dupes := resp["duplicates"].([]interface{})
	if len(dupes) != 0 {
		t.Errorf("expected 0 duplicates, got %d", len(dupes))
	}
}

func TestUploadCheck_shouldRejectUnauthenticated(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/upload/check", "[]", jsonHeaders())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUploadActiveJobs_shouldReturnActiveJobs(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)
	u, _ := us.Create("activejobsu_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	tmpDir := t.TempDir()
	tmpPath := tmpDir + "/test-active.bin"
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(tmpPath, []byte("test"), 0644)

	j1 := &model.UploadJob{
		BatchID:   "batch-active-test",
		UserID:    u.ID,
		Filename:  "active.jpg",
		SizeBytes: 1024,
		TempPath:  tmpPath,
		Status:    model.JobStatusQueued,
	}
	ujs.Create(j1)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/upload/active", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	jobs := resp["jobs"].([]interface{})
	if len(jobs) < 1 {
		t.Fatalf("expected at least 1 job, got %d", len(jobs))
	}

	first := jobs[0].(map[string]interface{})
	if first["filename"] != "active.jpg" {
		t.Errorf("expected active.jpg, got %v", first["filename"])
	}
	if first["status"] != "queued" {
		t.Errorf("expected queued status, got %v", first["status"])
	}
	if _, ok := first["batch_id"]; !ok {
		t.Error("expected batch_id in response")
	}
}

func TestUploadActiveJobs_shouldRejectUnauthenticated(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/upload/active", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpload_shouldRejectQuotaExceeded(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("quota_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	quota := int64(50)
	us.UpdateSpaceQuota(u.ID, &quota)

	data := []byte("this file content is more than 50 bytes long so the quota check should reject it")
	body, boundary := multipartUploadBody(t, map[string][]byte{"big.jpg": data})

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	req := httptest.NewRequest("POST", "/api/v1/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
}

func TestUpload_shouldAllowWhenUnderQuota(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("quota2_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	quota := int64(1024 * 1024)
	us.UpdateSpaceQuota(u.ID, &quota)

	data := []byte("small file")
	body, boundary := multipartUploadBody(t, map[string][]byte{"small.jpg": data})

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	req := httptest.NewRequest("POST", "/api/v1/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestUpload_shouldAllowWhenUnlimited(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("quota3_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	bigData := make([]byte, 1024*1024)
	body, boundary := multipartUploadBody(t, map[string][]byte{"large.jpg": bigData})

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	req := httptest.NewRequest("POST", "/api/v1/upload", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}
