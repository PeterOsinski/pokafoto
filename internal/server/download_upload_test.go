package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestDownload_shouldReturn404WhenFileNotFound(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "GET", "/api/v1/download/nonexistent-id", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
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
