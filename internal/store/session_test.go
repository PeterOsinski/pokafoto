package store

import (
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

func createTestUser(t *testing.T, s *UserStore) *model.User {
	t.Helper()
	u, err := s.Create("testuser_"+t.Name(), "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

func TestSessionStore_Create_shouldReturnSession(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user := createTestUser(t, us)
	expires := time.Now().UTC().Add(72 * time.Hour)

	sess, err := ss.Create(user.ID, expires)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if sess.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sess.UserID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, sess.UserID)
	}
	if sess.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestSessionStore_FindByRefreshToken_shouldReturnSession(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user := createTestUser(t, us)
	created, _ := ss.Create(user.ID, time.Now().UTC().Add(72*time.Hour))

	found, err := ss.FindByRefreshToken(created.RefreshToken)
	if err != nil {
		t.Fatalf("find by refresh token: %v", err)
	}
	if found == nil {
		t.Fatal("expected session, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, found.ID)
	}
}

func TestSessionStore_FindByRefreshToken_shouldReturnNil(t *testing.T) {
	db := OpenTestDB(t)
	ss := NewSessionStore(db)

	sess, err := ss.FindByRefreshToken("nonexistent-token")
	if err != nil {
		t.Fatalf("find by refresh token: %v", err)
	}
	if sess != nil {
		t.Error("expected nil for nonexistent token")
	}
}

func TestSessionStore_Delete_shouldRemoveSession(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user := createTestUser(t, us)
	sess, _ := ss.Create(user.ID, time.Now().UTC().Add(72*time.Hour))

	if err := ss.Delete(sess.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	found, _ := ss.FindByRefreshToken(sess.RefreshToken)
	if found != nil {
		t.Error("expected nil after delete")
	}
}

func TestSessionStore_DeleteByRefreshToken_shouldRemoveSession(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user := createTestUser(t, us)
	sess, _ := ss.Create(user.ID, time.Now().UTC().Add(72*time.Hour))

	if err := ss.DeleteByRefreshToken(sess.RefreshToken); err != nil {
		t.Fatalf("delete by refresh token: %v", err)
	}
	found, _ := ss.FindByRefreshToken(sess.RefreshToken)
	if found != nil {
		t.Error("expected nil after delete by token")
	}
}

func TestSessionStore_DeleteByUserID_shouldRemoveAllSessions(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	ss := NewSessionStore(db)

	user := createTestUser(t, us)
	s1, _ := ss.Create(user.ID, time.Now().UTC().Add(72*time.Hour))
	s2, _ := ss.Create(user.ID, time.Now().UTC().Add(48*time.Hour))

	if err := ss.DeleteByUserID(user.ID); err != nil {
		t.Fatalf("delete by user id: %v", err)
	}

	f1, _ := ss.FindByRefreshToken(s1.RefreshToken)
	f2, _ := ss.FindByRefreshToken(s2.RefreshToken)
	if f1 != nil || f2 != nil {
		t.Error("expected all sessions deleted for user")
	}
}
