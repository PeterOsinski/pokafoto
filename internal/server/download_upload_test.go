package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestBatchDownload_shouldRejectTooManyFiles(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	ids := make([]string, 101)
	for i := range ids {
		ids[i] = "id"
	}
	body, _ := json.Marshal(map[string]interface{}{"file_ids": ids})

	w := testRequest(t, srv, "POST", "/api/v1/download/batch", string(body), map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for too many files, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchDownload_shouldReturnBadRequestForEmptyFileIDs(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "POST", "/api/v1/download/batch", `{"file_ids":[]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty file IDs, got %d", w.Code)
	}
}

func TestBatchDownload_shouldReturnBadRequestForInvalidJSON(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "POST", "/api/v1/download/batch", `{invalid}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
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

func TestVideoStream_shouldServeWithRangeSupport(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("videostream_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	userDir := filepath.Join(srv.cfg.OriginalsDir(), u.ID)
	os.MkdirAll(userDir, 0755)

	srcPath := "/tmp/test_video_1080p.mp4"
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		t.Skipf("test video not found: %v", err)
	}

	destPath := filepath.Join(userDir, "2024/07/test-video.mp4")
	os.MkdirAll(filepath.Dir(destPath), 0755)
	os.WriteFile(destPath, srcData, 0644)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/test-video.mp4",
		OriginalName: "test-video.mp4",
		Path:         "/2024",
		SizeBytes:    int64(len(srcData)),
		MimeType:     "video/mp4",
		SHA256:       makeHandlerSHA256("videostream"),
		MediaType:    model.MediaTypeVideo,
	}
	fs.Create(f)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/video/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("expected Accept-Ranges: bytes header")
	}
	if w.Header().Get("Content-Type") != "video/mp4" {
		t.Errorf("expected Content-Type video/mp4, got %s", w.Header().Get("Content-Type"))
	}

	rangeW := testRequest(t, srv, "GET", "/api/v1/video/"+f.ID, "", map[string]string{
		"Authorization": "Bearer " + token,
		"Range":         "bytes=0-999",
	})
	if rangeW.Code != http.StatusPartialContent {
		t.Errorf("expected 206 for range request, got %d", rangeW.Code)
	}
}

func TestVideoStream_shouldRejectNonVideoFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("notvideo_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	photo := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "/2024",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("notvideo"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(photo)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/video/"+photo.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-video, got %d", w.Code)
	}
}

func TestUploadStatus_shouldReturnBatchStatus(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	ujs := store.NewUploadJobStore(db)
	u, _ := us.Create("uploadstatus_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	job1 := &model.UploadJob{
		BatchID:  "batch-test-status",
		UserID:   u.ID,
		Filename: "photo1.jpg",
		Status:   model.JobStatusCompleted,
		TempPath: "/tmp/photo1.jpg",
	}
	ujs.Create(job1)

	job2 := &model.UploadJob{
		BatchID:  "batch-test-status",
		UserID:   u.ID,
		Filename: "photo2.jpg",
		Status:   model.JobStatusFailed,
		TempPath: "/tmp/photo2.jpg",
	}
	reason := "test_reason"
	job2.Reason = &reason
	ujs.Create(job2)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "GET", "/api/v1/upload/batch-test-status/status", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["batch_id"] != "batch-test-status" {
		t.Errorf("expected batch-test-status, got %v", resp["batch_id"])
	}
	if resp["total"].(float64) != 2 {
		t.Errorf("expected 2 total, got %v", resp["total"])
	}
	if resp["completed"].(float64) != 1 {
		t.Errorf("expected 1 completed, got %v", resp["completed"])
	}
	if resp["failed"].(float64) != 1 {
		t.Errorf("expected 1 failed, got %v", resp["failed"])
	}
}

func TestServeThumbnail_shouldReturnThumbnail(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	u, _ := us.Create("thumbnailview_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/test_thumb.jpg",
		OriginalName: "test_thumb.jpg",
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("testthumb"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	thumbDir := filepath.Join(srv.cfg.ThumbnailsDir(), f.ID)
	os.MkdirAll(thumbDir, 0755)
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
		0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C,
		0x14, 0x0C, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D,
		0x1A, 0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20,
		0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27,
		0x39, 0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34,
		0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4,
		0x00, 0x1F, 0x00, 0x00, 0x01, 0x05, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0xFF,
		0xC4, 0x00, 0xB5, 0x10, 0x00, 0x02, 0x01, 0x03,
		0x03, 0x02, 0x04, 0x03, 0x05, 0x05, 0x04, 0x04,
		0x00, 0x00, 0x01, 0x7D, 0x01, 0x02, 0x03, 0x00,
		0x04, 0x11, 0x05, 0x12, 0x21, 0x31, 0x41, 0x06,
		0x13, 0x51, 0x61, 0x07, 0x22, 0x71, 0x14, 0x32,
		0x81, 0x91, 0xA1, 0x08, 0x23, 0x42, 0xB1, 0xC1,
		0x15, 0x52, 0xD1, 0xF0, 0x24, 0x33, 0x62, 0x72,
		0x82, 0x09, 0x0A, 0x16, 0x17, 0x18, 0x19, 0x1A,
		0x25, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x34, 0x35,
		0x36, 0x37, 0x38, 0x39, 0x3A, 0x43, 0x44, 0x45,
		0x46, 0x47, 0x48, 0x49, 0x4A, 0x53, 0x54, 0x55,
		0x56, 0x57, 0x58, 0x59, 0x5A, 0x63, 0x64, 0x65,
		0x66, 0x67, 0x68, 0x69, 0x6A, 0x73, 0x74, 0x75,
		0x76, 0x77, 0x78, 0x79, 0x7A, 0x83, 0x84, 0x85,
		0x86, 0x87, 0x88, 0x89, 0x8A, 0x92, 0x93, 0x94,
		0x95, 0x96, 0x97, 0x98, 0x99, 0x9A, 0xA2, 0xA3,
		0xA4, 0xA5, 0xA6, 0xA7, 0xA8, 0xA9, 0xAA, 0xB2,
		0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9, 0xBA,
		0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9,
		0xCA, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7, 0xD8,
		0xD9, 0xDA, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6,
		0xE7, 0xE8, 0xE9, 0xEA, 0xF1, 0xF2, 0xF3, 0xF4,
		0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFF, 0xD9,
	}
	os.WriteFile(filepath.Join(thumbDir, "sm"), jpegData, 0644)

	thumb := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeSmall,
		Width:     60,
		Height:    40,
		Format:    "jpeg",
		LocalPath: filepath.Join(thumbDir, "sm"),
		SizeBytes: int64(len(jpegData)),
	}
	ts.Create(thumb)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "GET", "/api/v1/thumb/"+f.ID+"/sm", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", ct)
	}
}

func TestServeThumbnail_shouldFallbackToMedium(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	u, _ := us.Create("thumbfallback_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/fallback_thumb.jpg",
		OriginalName: "fallback_thumb.jpg",
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("fallbackthumb"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	thumbDir := filepath.Join(srv.cfg.ThumbnailsDir(), f.ID)
	os.MkdirAll(thumbDir, 0755)
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	os.WriteFile(filepath.Join(thumbDir, "sm.jpg"), jpegData, 0644)

	thumb := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeSmall,
		Width:     60,
		Height:    40,
		Format:    "jpeg",
		LocalPath: filepath.Join(thumbDir, "sm.jpg"),
		SizeBytes: int64(len(jpegData)),
	}
	ts.Create(thumb)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "GET", "/api/v1/thumb/"+f.ID+"/md.jpg", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for md.jpg fallback to sm.jpg, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServeThumbnail_shouldReturn404WhenNoThumbnail(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("thumbmissing_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f := &model.File{
		UserID:       u.ID,
		Filename:     "2024/07/nothumb.jpg",
		OriginalName: "nothumb.jpg",
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256("nothumb"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "GET", "/api/v1/thumb/"+f.ID+"/xl", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUploadStatus_emptyBatch_shouldReturnEmpty(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("emptystatus_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "GET", "/api/v1/upload/nonexistent-batch/status", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty batch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBatchDownload_withFiles_shouldReturnZip(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("batchdl_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	originalsDir := filepath.Join(srv.cfg.Storage.Local.Path, "originals", u.ID)
	os.MkdirAll(originalsDir, 0755)

	fileIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		filename := "batchtest" + string(rune('a'+i)) + ".txt"
		file := &model.File{
			UserID:       u.ID,
			Filename:     filename,
			OriginalName: filename,
			Path:         filepath.Join(originalsDir, filename),
			SizeBytes:    4,
			MimeType:     "text/plain",
			SHA256:       "batchsha" + string(rune('a'+i)),
			MediaType:    model.MediaTypeFile,
		}
		fs.Create(file)
		fileIDs[i] = file.ID
		os.WriteFile(filepath.Join(originalsDir, filename), []byte("test"), 0644)
	}

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	body, _ := json.Marshal(map[string]interface{}{"file_ids": fileIDs})
	w := testRequest(t, srv, "POST", "/api/v1/download/batch", string(body), map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected Content-Type application/zip, got %s", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty zip body")
	}
}

func TestSoftDelete_shouldMoveToTrash(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("softdel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "My Folder", nil)
	folderID := f.ID

	file := &model.File{
		UserID:       u.ID,
		Filename:     "todelete.jpg",
		OriginalName: "todelete.jpg",
		Path:         "/uploads/todelete.jpg",
		SizeBytes:    512,
		MimeType:     "image/jpeg",
		SHA256:       "softdelsha",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+file.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	trashed, _ := fs.FindByID(file.ID)
	if trashed == nil || !trashed.IsDeleted {
		t.Error("expected file to be soft-deleted (IsDeleted=true)")
	}
}

func TestRestoreTrash_shouldRestoreFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("restore_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "My Folder", nil)
	folderID := f.ID

	file := &model.File{
		UserID:       u.ID,
		Filename:     "torestore.jpg",
		OriginalName: "torestore.jpg",
		Path:         "/uploads/torestore.jpg",
		SizeBytes:    512,
		MimeType:     "image/jpeg",
		SHA256:       "restoresha",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "DELETE", "/api/v1/files/"+file.ID, "", map[string]string{"Authorization": "Bearer " + token})

	w := testRequest(t, srv, "POST", "/api/v1/trash/"+file.ID+"/restore", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	restored, _ := fs.FindByID(file.ID)
	if restored == nil || restored.IsDeleted {
		t.Error("expected file to be restored (IsDeleted=false)")
	}
}
