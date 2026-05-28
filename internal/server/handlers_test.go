package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func createTestFileForHandler(t *testing.T, fs *store.FileStore, userID, name string) *model.File {
	t.Helper()
	f := &model.File{
		UserID:       userID,
		Filename:     "2024/07/" + name,
		OriginalName: name,
		Path:         "/2024",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeHandlerSHA256(name),
		MediaType:    model.MediaTypePhoto,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create test file: %v", err)
	}
	return f
}

func makeHandlerSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

func TestHandlers_ListFiles_shouldReturnUserFiles(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("listfiles_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	createTestFileForHandler(t, fs, u.ID, "photo1.jpg")
	createTestFileForHandler(t, fs, u.ID, "photo2.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/files", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestHandlers_ListFiles_shouldFilterByPath(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("filterpath_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f1 := &model.File{
		UserID: u.ID, Filename: "2024/jan/a.jpg", OriginalName: "a.jpg", Path: "/2024",
		SizeBytes: 100, MimeType: "image/jpeg", SHA256: makeHandlerSHA256("a"), MediaType: model.MediaTypePhoto,
	}
	f2 := &model.File{
		UserID: u.ID, Filename: "2025/b.jpg", OriginalName: "b.jpg", Path: "/2025",
		SizeBytes: 200, MimeType: "image/jpeg", SHA256: makeHandlerSHA256("b"), MediaType: model.MediaTypePhoto,
	}
	fs.Create(f1)
	fs.Create(f2)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/files?path=/2024", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestHandlers_ListFiles_shouldEnforceUserIsolation(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u1, _ := us.Create("isolate1_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("isolate2_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	createTestFileForHandler(t, fs, u1.ID, "a.jpg")
	createTestFileForHandler(t, fs, u2.ID, "b.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/files", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestHandlers_GetFile_shouldReturnFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("getfile_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := createTestFileForHandler(t, fs, u.ID, "photo.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["id"] != f.ID {
		t.Errorf("expected id %s, got %v", f.ID, resp["id"])
	}
}

func TestHandlers_GetFile_shouldReturn404ForOtherUser(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u1, _ := us.Create("otheruser1_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("otheruser2_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := createTestFileForHandler(t, fs, u1.ID, "photo.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u2.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlers_Search_shouldReturnMatches(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("search_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	createTestFileForHandler(t, fs, u.ID, "sunset_beach.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/search?q=sunset", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 result, got %d", len(items))
	}
}

func TestHandlers_Search_shouldReturnEmptyForNoMatch(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("nosearch_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	createTestFileForHandler(t, fs, u.ID, "important.pdf")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/search?q=zzznotfound", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("expected 0 results, got %d", len(items))
	}
}

func TestHandlers_ListDirs_shouldReturnDirectories(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("dirs_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	f1 := &model.File{
		UserID: u.ID, Filename: "jan/a.jpg", OriginalName: "a.jpg", Path: "/2024",
		SizeBytes: 100, MimeType: "image/jpeg", SHA256: makeHandlerSHA256("a"), MediaType: model.MediaTypePhoto,
	}
	f2 := &model.File{
		UserID: u.ID, Filename: "feb/b.jpg", OriginalName: "b.jpg", Path: "/2025",
		SizeBytes: 200, MimeType: "image/jpeg", SHA256: makeHandlerSHA256("b"), MediaType: model.MediaTypePhoto,
	}
	fs.Create(f1)
	fs.Create(f2)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/dirs", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandlers_Stats_shouldReturnUserStats(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("stats_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	createTestFileForHandler(t, fs, u.ID, "photo.jpg")
	createTestFileForHandler(t, fs, u.ID, "photo2.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/stats", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total_files"] == nil {
		t.Errorf("expected total_files, got %v", resp)
	}
}

func TestHandlers_Timeline_shouldReturnGroups(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("timeline_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	takenAt := "2024-06-15T10:00:00Z"
	f := &model.File{
		UserID: u.ID, Filename: "june/td.jpg", OriginalName: "td.jpg", Path: "/2024/june",
		SizeBytes: 1024, MimeType: "image/jpeg", SHA256: makeHandlerSHA256("td"),
		MediaType: model.MediaTypePhoto, TakenAt: &takenAt,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/timeline", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandlers_GeoPoints_shouldReturnPointsInBbox(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	es := store.NewExifStore(db)
	u, _ := us.Create("geo_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := createTestFileForHandler(t, fs, u.ID, "gps_photo.jpg")

	lat, lon := 37.7749, -122.4194
	es.Create(&model.ExifData{
		FileID: f.ID, GPSLatitude: &lat, GPSLongitude: &lon,
	})

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/geo/points?lat_min=37.0&lat_max=38.0&lon_min=-123.0&lon_max=-122.0", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	points := resp["points"].([]interface{})
	if len(points) != 1 {
		t.Errorf("expected 1 point, got %d", len(points))
	}
}

func TestHandlers_SoftDelete_shouldReturn204(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("softdel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := createTestFileForHandler(t, fs, u.ID, "photo.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestHandlers_PermanentDelete_shouldReturn204(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("permadel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := createTestFileForHandler(t, fs, u.ID, "photo.jpg")

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID+"/permanent", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}

	exists, _ := fs.FindByID(f.ID)
	if exists != nil {
		t.Error("file still exists after permanent delete")
	}
}

func TestHandlers_Health_shouldReturnOK(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/health", "", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
