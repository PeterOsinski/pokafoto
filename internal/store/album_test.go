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

func TestAlbumStore_FindByIDWithOwner_shouldReturnAlbumWithDetails(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	albumStore := NewAlbumStore(db)

	album, _ := albumStore.Create(u.ID, "Detailable", nil)

	awd, err := albumStore.FindByIDWithOwner(album.ID)
	if err != nil {
		t.Fatalf("FindByIDWithOwner() error = %v", err)
	}
	if awd.Album.ID != album.ID {
		t.Errorf("expected album ID %q, got %q", album.ID, awd.Album.ID)
	}
	if awd.OwnerName != u.Username {
		t.Errorf("expected owner %q, got %q", u.Username, awd.OwnerName)
	}
}

func TestAlbumStore_ListSharedWithUser_shouldListSharedAlbums(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("vieweruser_listshared", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	a1, _ := albumStore.Create(owner.ID, "Shared 1", nil)
	a2, _ := albumStore.Create(owner.ID, "Shared 2", nil)
	albumStore.Create(owner.ID, "Private", nil)

	shareStore.Add(a1.ID, viewer.ID, "view")
	shareStore.Add(a2.ID, viewer.ID, "comment")

	albums, err := albumStore.ListSharedWithUser(viewer.ID)
	if err != nil {
		t.Fatalf("ListSharedWithUser() error = %v", err)
	}
	if len(albums) != 2 {
		t.Errorf("expected 2 shared albums, got %d", len(albums))
	}
	for _, a := range albums {
		if a.OwnerName != owner.Username {
			t.Errorf("expected owner %q, got %q", owner.Username, a.OwnerName)
		}
	}
}

func TestAlbumStore_Update_shouldUpdateAlbum(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	albumStore := NewAlbumStore(db)

	album, _ := albumStore.Create(u.ID, "Original", nil)
	desc := "Updated desc"

	err := albumStore.Update(album.ID, "Updated", &desc)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	found, _ := albumStore.FindByID(album.ID)
	if found.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", found.Name)
	}
	if found.Description == nil || *found.Description != "Updated desc" {
		t.Error("expected description to be updated")
	}
}

func TestAlbumStore_ListShares_shouldListShares(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	v1, _ := us.Create("v1_listshares", "password123", model.RoleMember, nil)
	v2, _ := us.Create("v2_listshares", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Shared Album", nil)
	shareStore.Add(album.ID, v1.ID, "view")
	shareStore.Add(album.ID, v2.ID, "comment")

	shares, err := albumStore.ListShares(album.ID)
	if err != nil {
		t.Fatalf("ListShares() error = %v", err)
	}
	if len(shares) != 2 {
		t.Errorf("expected 2 shares, got %d", len(shares))
	}
}

func TestAlbumStore_ItemCount_shouldReturnCount(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "Counting Album", nil)

	file := createTestFile(t, fs, u.ID, "photo1.jpg")
	itemStore.Add(album.ID, file.ID, u.ID)

	count := albumStore.ItemCount(album.ID)
	if count != 1 {
		t.Errorf("expected item count 1, got %d", count)
	}
}

func TestAlbumStore_HasShares_shouldDetectShares(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	viewer, _ := us.Create("has_share_user", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(u.ID, "Check Shares", nil)
	if albumStore.HasShares(album.ID) {
		t.Error("expected no shares initially")
	}

	shareStore.Add(album.ID, viewer.ID, "view")
	if !albumStore.HasShares(album.ID) {
		t.Error("expected HasShares to return true after sharing")
	}
}

func TestAlbumItemStore_Add_shouldAddItem(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "Test Album", nil)
	file := createTestFile(t, fs, u.ID, "photo.jpg")

	item, err := itemStore.Add(album.ID, file.ID, u.ID)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if item.AlbumID != album.ID {
		t.Errorf("expected albumID %q, got %q", album.ID, item.AlbumID)
	}
	if item.FileID != file.ID {
		t.Errorf("expected fileID %q, got %q", file.ID, item.FileID)
	}
}

func TestAlbumItemStore_FindByAlbumAndFile_shouldFindItem(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "Find Me", nil)
	file := createTestFile(t, fs, u.ID, "target.jpg")

	itemStore.Add(album.ID, file.ID, u.ID)

	found, err := itemStore.FindByAlbumAndFile(album.ID, file.ID)
	if err != nil {
		t.Fatalf("FindByAlbumAndFile() error = %v", err)
	}
	if found.FileID != file.ID {
		t.Errorf("expected fileID %q, got %q", file.ID, found.FileID)
	}

	_, err = itemStore.FindByAlbumAndFile(album.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent item")
	}
}

func TestAlbumItemStore_Remove_shouldRemoveItem(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "Remove Test", nil)
	file := createTestFile(t, fs, u.ID, "gone.jpg")

	itemStore.Add(album.ID, file.ID, u.ID)

	if err := itemStore.Remove(album.ID, file.ID); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	_, err := itemStore.FindByAlbumAndFile(album.ID, file.ID)
	if err == nil {
		t.Error("expected item to be removed")
	}
}

func TestAlbumItemStore_RemoveByID_shouldRemoveItem(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "RemoveByID Test", nil)
	file := createTestFile(t, fs, u.ID, "byid.jpg")

	item, _ := itemStore.Add(album.ID, file.ID, u.ID)

	if err := itemStore.RemoveByID(item.ID); err != nil {
		t.Fatalf("RemoveByID() error = %v", err)
	}

	_, err := itemStore.FindByAlbumAndFile(album.ID, file.ID)
	if err == nil {
		t.Error("expected item to be removed by ID")
	}
}

func TestAlbumItemStore_ListFileIDs_shouldListFiles(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)

	album, _ := albumStore.Create(u.ID, "FileList Test", nil)

	f1 := createTestFile(t, fs, u.ID, "first.jpg")
	f2 := createTestFile(t, fs, u.ID, "second.jpg")

	itemStore.Add(album.ID, f1.ID, u.ID)
	itemStore.Add(album.ID, f2.ID, u.ID)

	ids, total, err := itemStore.ListFileIDs(album.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListFileIDs() error = %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
}

func TestAlbumItemStore_HasSharedAccess_shouldDetectAccess(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("sharedaccess_viewer", "password123", model.RoleMember, nil)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Shared Access", nil)
	file := createTestFile(t, fs, owner.ID, "sharedfile.jpg")

	itemStore.Add(album.ID, file.ID, owner.ID)
	shareStore.Add(album.ID, viewer.ID, "view")

	has, err := itemStore.HasSharedAccess(file.ID, viewer.ID)
	if err != nil {
		t.Fatalf("HasSharedAccess() error = %v", err)
	}
	if !has {
		t.Error("expected shared access to be true")
	}

	has, err = itemStore.HasSharedAccess(file.ID, "nonexistent-user")
	if err != nil {
		t.Fatalf("HasSharedAccess() error = %v", err)
	}
	if has {
		t.Error("expected no shared access for nonexistent user")
	}
}

func TestAlbumItemStore_GetSharedPermission_shouldGetPermission(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("perm_viewer", "password123", model.RoleMember, nil)
	commenter, _ := us.Create("perm_commenter", "password123", model.RoleMember, nil)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Permission Test", nil)
	file := createTestFile(t, fs, owner.ID, "permfile.jpg")

	itemStore.Add(album.ID, file.ID, owner.ID)
	shareStore.Add(album.ID, viewer.ID, "view")
	shareStore.Add(album.ID, commenter.ID, "comment")

	perm, err := itemStore.GetSharedPermission(file.ID, viewer.ID)
	if err != nil {
		t.Fatalf("GetSharedPermission() error = %v", err)
	}
	if perm != "view" {
		t.Errorf("expected permission 'view', got %q", perm)
	}

	perm, err = itemStore.GetSharedPermission(file.ID, commenter.ID)
	if err != nil {
		t.Fatalf("GetSharedPermission() error = %v", err)
	}
	if perm != "comment" {
		t.Errorf("expected permission 'comment', got %q", perm)
	}
}

func TestAlbumItemStore_ListAlbumsByFile_shouldListAlbums(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("albumsbyfile_viewer", "password123", model.RoleMember, nil)
	fs := NewFileStore(db)
	albumStore := NewAlbumStore(db)
	itemStore := NewAlbumItemStore(db)
	shareStore := NewAlbumShareStore(db)

	a1, _ := albumStore.Create(owner.ID, "Album One", nil)
	a2, _ := albumStore.Create(owner.ID, "Album Two", nil)
	file := createTestFile(t, fs, owner.ID, "crossfile.jpg")

	itemStore.Add(a1.ID, file.ID, owner.ID)
	itemStore.Add(a2.ID, file.ID, owner.ID)
	shareStore.Add(a2.ID, viewer.ID, "view")

	albums, err := itemStore.ListAlbumsByFile(file.ID, viewer.ID)
	if err != nil {
		t.Fatalf("ListAlbumsByFile() error = %v", err)
	}
	if len(albums) != 1 {
		t.Errorf("expected 1 visible album for viewer, got %d", len(albums))
	}
	if albums[0].ID != a2.ID {
		t.Errorf("expected album %q, got %q", a2.ID, albums[0].ID)
	}

	albums, err = itemStore.ListAlbumsByFile(file.ID, owner.ID)
	if err != nil {
		t.Fatalf("ListAlbumsByFile() error = %v", err)
	}
	if len(albums) != 2 {
		t.Errorf("expected 2 albums for owner, got %d", len(albums))
	}
}

func TestAlbumShareStore_Remove_shouldRemoveShare(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("remove_share_user", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Unshare Me", nil)
	share, _ := shareStore.Add(album.ID, viewer.ID, "view")

	if err := shareStore.Remove(share.ID); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	_, err := shareStore.FindByAlbumAndUser(album.ID, viewer.ID)
	if err == nil {
		t.Error("expected share to be removed")
	}
}

func TestAlbumShareStore_FindByAlbumAndUser_shouldFindShare(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("find_share_user", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Find Share", nil)
	shareStore.Add(album.ID, viewer.ID, "edit")

	found, err := shareStore.FindByAlbumAndUser(album.ID, viewer.ID)
	if err != nil {
		t.Fatalf("FindByAlbumAndUser() error = %v", err)
	}
	if found.AlbumID != album.ID {
		t.Errorf("expected albumID %q, got %q", album.ID, found.AlbumID)
	}
	if found.Permission != "edit" {
		t.Errorf("expected permission 'edit', got %q", found.Permission)
	}

	_, err = shareStore.FindByAlbumAndUser(album.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent share")
	}
}

func TestAlbumShareStore_Add_shouldBeIdempotent(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	owner := createTestUser(t, us)
	viewer, _ := us.Create("dedup_share", "password123", model.RoleMember, nil)
	albumStore := NewAlbumStore(db)
	shareStore := NewAlbumShareStore(db)

	album, _ := albumStore.Create(owner.ID, "Dedup Share", nil)

	s1, err := shareStore.Add(album.ID, viewer.ID, "view")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	s2, err := shareStore.Add(album.ID, viewer.ID, "comment")
	if err != nil {
		t.Fatalf("Add() second call error = %v", err)
	}
	if s1.AlbumID != s2.AlbumID || s1.SharedWithUserID != s2.SharedWithUserID {
		t.Error("expected same album/share target on second Add")
	}

	found, _ := shareStore.FindByAlbumAndUser(album.ID, viewer.ID)
	if found.Permission != "view" {
		t.Errorf("expected permission 'view' preserved (INSERT OR IGNORE), got %q", found.Permission)
	}
}
