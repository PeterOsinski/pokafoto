package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestAlbum_ListAlbums_shouldReturnEmpty(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(srv.admin.DB)
	u, _ := us.Create("albumlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/albums", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	myAlbums := resp["myAlbums"].([]interface{})
	if len(myAlbums) != 0 {
		t.Errorf("expected 0 myAlbums, got %d", len(myAlbums))
	}
}

func TestAlbum_CreateAlbum_shouldCreateAlbum(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(srv.admin.DB)
	u, _ := us.Create("albumcreate_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/albums", `{"name":"My Vacation"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "My Vacation" {
		t.Errorf("expected 'My Vacation', got %v", resp["name"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected album ID")
	}
}

func TestAlbum_CreateAlbum_shouldRejectEmptyName(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(srv.admin.DB)
	u, _ := us.Create("albempty_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/albums", `{"name":""}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAlbum_GetAlbum_shouldReturnAlbum(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u, _ := us.Create("albumget_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Test Album", nil)

	w := testRequest(t, srv, "GET", "/api/v1/albums/"+album.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "Test Album" {
		t.Errorf("expected 'Test Album', got %v", resp["name"])
	}
}

func TestAlbum_GetAlbum_shouldReturn404ForNonexistent(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(srv.admin.DB)
	u, _ := us.Create("album404_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/albums/nonexistent-id", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAlbum_UpdateAlbum_shouldUpdateName(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u, _ := us.Create("albumupd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Original", nil)

	w := testRequest(t, srv, "PUT", "/api/v1/albums/"+album.ID, `{"name":"Updated Name"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlbum_UpdateAlbum_shouldRejectNonOwner(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u1, _ := us.Create("albowner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("albother_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u2.ID, "member")

	album, _ := as.Create(u1.ID, "Owned Album", nil)

	w := testRequest(t, srv, "PUT", "/api/v1/albums/"+album.ID, `{"name":"Hacked"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAlbum_DeleteAlbum_shouldDelete(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u, _ := us.Create("albdel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Delete Me", nil)

	w := testRequest(t, srv, "DELETE", "/api/v1/albums/"+album.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestAlbum_DeleteAlbum_shouldRejectNonOwner(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u1, _ := us.Create("albdelowner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("albdelother_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u2.ID, "member")

	album, _ := as.Create(u1.ID, "Owned Album", nil)

	w := testRequest(t, srv, "DELETE", "/api/v1/albums/"+album.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAlbum_ListAlbumItems_shouldReturnFiles(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	ais := store.NewAlbumItemStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("albitems_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Album", nil)
	f := createTestFileForHandler(t, fs, u.ID, "album-photo.jpg")
	ais.Add(album.ID, f.ID, u.ID)

	w := testRequest(t, srv, "GET", "/api/v1/albums/"+album.ID+"/items", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestAlbum_AddAlbumItems_shouldAddFiles(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("albadd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Album", nil)
	f := createTestFileForHandler(t, fs, u.ID, "add-photo.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/albums/"+album.ID+"/items", `{"file_ids":["`+f.ID+`"]}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlbum_RemoveAlbumItem_shouldRemoveFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	ais := store.NewAlbumItemStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("albrem_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	album, _ := as.Create(u.ID, "Album", nil)
	f := createTestFileForHandler(t, fs, u.ID, "remove-photo.jpg")
	ais.Add(album.ID, f.ID, u.ID)

	w := testRequest(t, srv, "DELETE", "/api/v1/albums/"+album.ID+"/items/"+f.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlbum_ShareAlbum_shouldCreateShare(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u1, _ := us.Create("albshareowner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	target, _ := us.Create("albsharetarget_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")

	album, _ := as.Create(u1.ID, "Shared Album", nil)

	w := testRequest(t, srv, "POST", "/api/v1/albums/"+album.ID+"/shares", `{"username":"`+target.Username+`","permission":"edit"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlbum_RemoveShare_shouldRemoveShare(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	ass := store.NewAlbumShareStore(db)
	u1, _ := us.Create("albremowner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("albremtarget_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")

	album, _ := as.Create(u1.ID, "Album", nil)
	share, _ := ass.Add(album.ID, u2.ID, "edit")

	w := testRequest(t, srv, "DELETE", "/api/v1/albums/"+album.ID+"/shares/"+share.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAlbum_ShareAlbum_shouldRejectSelf(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u1, _ := us.Create("albself_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")

	album, _ := as.Create(u1.ID, "My Album", nil)

	w := testRequest(t, srv, "POST", "/api/v1/albums/"+album.ID+"/shares", `{"username":"`+u1.Username+`","permission":"edit"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code >= 200 && w.Code < 300 {
		t.Errorf("expected error when sharing to self, got %d", w.Code)
	}
}

func TestAlbum_ShareAlbum_shouldRejectInvalidUser(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	u1, _ := us.Create("albinvusr_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")

	album, _ := as.Create(u1.ID, "My Album", nil)

	w := testRequest(t, srv, "POST", "/api/v1/albums/"+album.ID+"/shares", `{"username":"nonexistent_user_999","permission":"edit"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code < 400 {
		t.Errorf("expected 4xx for invalid user, got %d", w.Code)
	}
}

func TestAlbum_RemoveShare_shouldRejectNonOwner(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	as := store.NewAlbumStore(db)
	ass := store.NewAlbumShareStore(db)
	u1, _ := us.Create("albnonown1_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("albnonown2_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u3, _ := us.Create("albnonown3_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	album, _ := as.Create(u1.ID, "Album", nil)
	share, _ := ass.Add(album.ID, u2.ID, "edit")

	token3 := generateTestToken(srv.cfg.Auth.JWTSecret, u3.ID, "member")
	w := testRequest(t, srv, "DELETE", "/api/v1/albums/"+album.ID+"/shares/"+share.ID, "", map[string]string{"Authorization": "Bearer " + token3})
	if w.Code < 400 {
		t.Errorf("expected 4xx for non-owner, got %d", w.Code)
	}
}
