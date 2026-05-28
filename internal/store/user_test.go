package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestUserStore_Create_shouldReturnUser(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	name := "testuser"
	u, err := s.Create(name, "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.Username != name {
		t.Errorf("expected username %q, got %q", name, u.Username)
	}
	if u.Role != model.RoleMember {
		t.Errorf("expected role member, got %q", u.Role)
	}
	if u.PasswordHash == "password123" {
		t.Error("password should be hashed, not plaintext")
	}
}

func TestUserStore_Create_shouldHashPassword(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	u, _ := s.Create("pwuser", "securepassword", model.RoleMember, nil)
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("securepassword")); err != nil {
		t.Error("password hash does not match original password")
	}
}

func TestUserStore_Create_shouldReturnErrorOnDuplicate(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	s.Create("dupe", "pass1", model.RoleMember, nil)
	_, err := s.Create("dupe", "pass2", model.RoleMember, nil)
	if err == nil {
		t.Error("expected error on duplicate username")
	}
}

func TestUserStore_Create_shouldSetDisplayName(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	dn := "Display Name"
	u, err := s.Create("disp", "pass", model.RoleMember, &dn)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.DisplayName == nil || *u.DisplayName != dn {
		t.Errorf("expected display name %q, got %v", dn, u.DisplayName)
	}
}

func TestUserStore_FindByUsername_shouldReturnUser(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	s.Create("findme", "pass", model.RoleAdmin, nil)
	u, err := s.FindByUsername("findme")
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if u == nil {
		t.Fatal("expected user, got nil")
	}
	if u.Username != "findme" {
		t.Errorf("expected findme, got %q", u.Username)
	}
	if u.Role != model.RoleAdmin {
		t.Errorf("expected admin, got %q", u.Role)
	}
}

func TestUserStore_FindByUsername_shouldReturnNil(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	u, err := s.FindByUsername("nonexistent")
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if u != nil {
		t.Errorf("expected nil for nonexistent user, got %v", u)
	}
}

func TestUserStore_FindByID_shouldReturnUser(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	created, _ := s.Create("byid", "pass", model.RoleMember, nil)
	found, err := s.FindByID(created.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, found.ID)
	}
}

func TestUserStore_FindByID_shouldReturnNil(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	u, err := s.FindByID("nonexistent-id")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if u != nil {
		t.Error("expected nil for nonexistent id")
	}
}

func TestUserStore_List_shouldReturnAllUsers(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	s.Create("a", "pass", model.RoleMember, nil)
	s.Create("b", "pass", model.RoleAdmin, nil)

	users, err := s.List()
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUserStore_List_shouldReturnEmpty(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	users, err := s.List()
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestUserStore_UpdateRole_shouldChangeRole(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	u, _ := s.Create("roleuser", "pass", model.RoleMember, nil)
	if err := s.UpdateRole(u.ID, model.RoleAdmin); err != nil {
		t.Fatalf("update role: %v", err)
	}
	found, _ := s.FindByID(u.ID)
	if found.Role != model.RoleAdmin {
		t.Errorf("expected admin, got %q", found.Role)
	}
}

func TestUserStore_Delete_shouldRemoveUser(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	u, _ := s.Create("delme", "pass", model.RoleMember, nil)
	if err := s.Delete(u.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	found, _ := s.FindByID(u.ID)
	if found != nil {
		t.Error("expected nil after delete")
	}
}

func TestUserStore_Count_shouldReturnCorrectCount(t *testing.T) {
	db := OpenTestDB(t)
	s := NewUserStore(db)

	c1, _ := s.Count()
	if c1 != 0 {
		t.Errorf("expected 0, got %d", c1)
	}

	s.Create("c1", "pass", model.RoleMember, nil)
	s.Create("c2", "pass", model.RoleAdmin, nil)

	c2, _ := s.Count()
	if c2 != 2 {
		t.Errorf("expected 2, got %d", c2)
	}
}
