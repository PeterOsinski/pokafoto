package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestComments_ListComments_shouldReturnComments(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	u, _ := us.Create("commentlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "comment-photo.jpg")
	cs.Create(f.ID, u.ID, "First comment")
	cs.Create(f.ID, u.ID, "Second comment")

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID+"/comments", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	comments := resp["comments"].([]interface{})
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

func TestComments_AddComment_shouldCreateComment(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("commentadd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "comment-add.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/files/"+f.ID+"/comments", `{"content":"Great photo!"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["content"] != "Great photo!" {
		t.Errorf("expected 'Great photo!', got %v", resp["content"])
	}
}

func TestComments_AddComment_shouldRejectEmptyContent(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("commentempty_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "comment-empty.jpg")

	w := testRequest(t, srv, "POST", "/api/v1/files/"+f.ID+"/comments", `{"content":""}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestComments_UpdateComment_shouldUpdateContent(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	u, _ := us.Create("commentupd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "comment-update.jpg")
	c, _ := cs.Create(f.ID, u.ID, "Old text")

	w := testRequest(t, srv, "PUT", "/api/v1/files/"+f.ID+"/comments/"+c.ID, `{"content":"New text"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestComments_DeleteComment_shouldDelete(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	u, _ := us.Create("commentdel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "comment-delete.jpg")
	c, _ := cs.Create(f.ID, u.ID, "Delete me")

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID+"/comments/"+c.ID, "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReactions_ToggleReaction_shouldAdd(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	u, _ := us.Create("reacttoggle_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "react-photo.jpg")
	c, _ := cs.Create(f.ID, u.ID, "Nice photo")

	w := testRequest(t, srv, "POST", "/api/v1/files/"+f.ID+"/comments/"+c.ID+"/reactions", `{"emoji":"👍"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReactions_GetReactions_shouldReturnReactions(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	rs := store.NewReactionStore(db)
	u, _ := us.Create("reactget_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "react-get.jpg")
	c, _ := cs.Create(f.ID, u.ID, "Nice")
	rs.Toggle(c.ID, u.ID, "👍")

	w := testRequest(t, srv, "GET", "/api/v1/files/"+f.ID+"/comments/"+c.ID+"/reactions", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	reactions := resp["reactions"].([]interface{})
	if len(reactions) != 1 {
		t.Errorf("expected 1 reaction, got %d", len(reactions))
	}
}

func TestReactions_RemoveReaction_shouldRemove(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	cs := store.NewCommentStore(db)
	rs := store.NewReactionStore(db)
	u, _ := us.Create("reactrem_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	f := createTestFileForHandler(t, fs, u.ID, "react-rem.jpg")
	c, _ := cs.Create(f.ID, u.ID, "Nice")
	rs.Toggle(c.ID, u.ID, "🔥")

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID+"/comments/"+c.ID+"/reactions/%F0%9F%94%A5", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReactions_RemoveReaction_shouldReturn404OnNoAccess(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u1, _ := us.Create("reactremowner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("reactremother_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u2.ID, "member")

	f := createTestFileForHandler(t, fs, u1.ID, "react-noaccess.jpg")

	w := testRequest(t, srv, "DELETE", "/api/v1/files/"+f.ID+"/comments/any-id/reactions/👍", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
