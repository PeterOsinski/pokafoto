package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestTags_ListTags_shouldReturnTags(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewTagStore(db)
	u, _ := us.Create("taglist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "tagged.jpg")
	tag, _ := ts.FindOrCreate("vacation")
	ts.AddToFile(f.ID, tag.ID, u.ID)

	w := testRequest(t, srv, "GET", "/api/v1/tags", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTags_GetFileTags_shouldReturnFileTags(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewTagStore(db)
	u, _ := us.Create("tagfile_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "tagged-file.jpg")
	tag, _ := ts.FindOrCreate("landscape")
	ts.AddToFile(f.ID, tag.ID, u.ID)

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID+"/tags", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	tags := resp["tags"].([]interface{})
	if len(tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(tags))
	}
}

func TestTags_AddFileTags_shouldAddTags(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("tagadd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "add-tag.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/files/"+f.ID+"/tags", `{"tags":["sunset","beach"]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTags_RemoveFileTag_shouldRemoveTag(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewTagStore(db)
	u, _ := us.Create("tagrem_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "remove-tag.jpg")
	tag, _ := ts.FindOrCreate("vacation")
	ts.AddToFile(f.ID, tag.ID, u.ID)

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID+"/tags/"+tag.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTags_GetFileAlbums_shouldReturnFileAlbums(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	as := store.NewAlbumStore(db)
	ais := store.NewAlbumItemStore(db)
	u, _ := us.Create("tagalbum_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "file-in-album.jpg")
	album, _ := as.Create(u.ID, "My Album", nil)
	ais.Add(album.ID, f.ID, u.ID)

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID+"/albums", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
