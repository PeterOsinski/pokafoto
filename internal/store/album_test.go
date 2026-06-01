package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestAlbumStore_Create_shouldCreateAlbum(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	albumStore := NewAlbumStore(db)

	desc := "Test album description"
	album, err := albumStore.Create(u.ID, "My Album", &desc)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if album.Name != "My Album" {
		t.Errorf("expected name 'My Album', got %q", album.Name)
	}
	if album.UserID != u.ID {
		t.Errorf("expected userID %q, got %q", u.ID, album.UserID)
	}
	if album.Description == nil || *album.Description != "Test album description" {
		t.Error("expected description to be set")
	}

	found, err := albumStore.FindByID(album.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found == nil {
		t.Fatal("expected album to be found")
	}
}

func TestAlbumStore_ListByUser_shouldListOwnAlbums(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	albumStore := NewAlbumStore(db)

	albumStore.Create(u.ID, "Album 1", nil)
	albumStore.Create(u.ID, "Album 2", nil)

	albums, err := albumStore.ListByUser(u.ID)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(albums) != 2 {
		t.Errorf("expected 2 albums, got %d", len(albums))
	}
}

func TestAlbumStore_CheckAccess_shouldDetectOwner(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	other, _ := us.Create("otheruser_checkaccess_owner", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)

	album, _ := albumStore.Create(owner.ID, "Shared Album", nil)

	perm, found, err := albumStore.CheckAccess(album.ID, owner.ID)
	if err != nil {
		t.Fatalf("CheckAccess() error = %v", err)
	}
	if !found || perm != "edit" {
		t.Errorf("expected owner to have edit access, got found=%v perm=%q", found, perm)
	}

	perm, found, err = albumStore.CheckAccess(album.ID, other.ID)
	if err != nil {
		t.Fatalf("CheckAccess() error = %v", err)
	}
	if found {
		t.Error("expected other user to not have access before sharing")
	}
}

func TestAlbumStore_CheckAccess_shouldDetectShare(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("vieweruser_checkaccess_share", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Shared Album", nil)
	shareStore.Add(album.ID, viewer.ID, "view")

	perm, found, err := albumStore.CheckAccess(album.ID, viewer.ID)
	if err != nil {
		t.Fatalf("CheckAccess() error = %v", err)
	}
	if !found || perm != "view" {
		t.Errorf("expected viewer to have view access, got found=%v perm=%q", found, perm)
	}
}

func TestAlbumStore_Delete_shouldRemoveAlbum(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	albumStore := NewAlbumStore(db)

	album, _ := albumStore.Create(u.ID, "To Delete", nil)

	if err := albumStore.Delete(album.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := albumStore.FindByID(album.ID)
	if err == nil {
		t.Error("expected album to be deleted")
	}
}
