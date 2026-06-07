package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestShare_GetFile_shouldReturnFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shget_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "/uploads/photo.jpg",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       "abc123",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 creating share, got %d: %s", w.Code, w.Body.String())
	}
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	if wu.Code != http.StatusOK {
		t.Fatalf("expected 200 unlock, got %d: %s", wu.Code, wu.Body.String())
	}
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wg := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files/"+file.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wg.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wg.Code, wg.Body.String())
	}
	var fileResp map[string]interface{}
	json.Unmarshal(wg.Body.Bytes(), &fileResp)
	if fileResp["id"] != file.ID {
		t.Errorf("expected file ID %s, got %v", file.ID, fileResp["id"])
	}
	if fileResp["original_name"] != "photo.jpg" {
		t.Errorf("expected original_name photo.jpg, got %v", fileResp["original_name"])
	}
}

func TestShare_GetFile_shouldReturn404ForUnknownShare(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/share/bad-token/files/someid", "", map[string]string{})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestShare_GetFile_shouldReturn403WithoutSessionToken(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shnoauth_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wg := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files/someid", "", map[string]string{})
	if wg.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", wg.Code)
	}
}

func TestShare_GetFile_shouldReturn404ForFileNotInShare(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shnotree_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	sharedFolder, _ := folders.Create(u.ID, "Shared", nil)
	otherFolder, _ := folders.Create(u.ID, "Other", nil)

	sharedFolderID := sharedFolder.ID
	otherFolderID := otherFolder.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "secret.jpg",
		OriginalName: "secret.jpg",
		Path:         "/uploads/secret.jpg",
		SizeBytes:    2048,
		MimeType:     "image/jpeg",
		SHA256:       "def456",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &otherFolderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+sharedFolder.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wg := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files/"+file.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wg.Code != http.StatusNotFound {
		t.Errorf("expected 404 for file not in shared folder, got %d", wg.Code)
	}

	_ = sharedFolderID
}

func TestShare_Thumbnail_shouldReturnThumbnail(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	u, _ := us.Create("shthumb_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "/uploads/photo.jpg",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       "ghi789",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	thumbDir := filepath.Join(srv.cfg.ThumbnailsDir(), file.ID)
	os.MkdirAll(thumbDir, 0755)
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
	os.WriteFile(filepath.Join(thumbDir, "sm.jpg"), jpegData, 0644)

	thumb := &model.Thumbnail{
		FileID:    file.ID,
		Size:      model.ThumbSizeSmall,
		Width:     60,
		Height:    40,
		Format:    "jpeg",
		LocalPath: filepath.Join(thumbDir, "sm.jpg"),
		SizeBytes: int64(len(jpegData)),
	}
	ts.Create(thumb)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wt := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/thumb/"+file.ID+"/sm.jpg", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wt.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wt.Code, wt.Body.String())
	}
	ct := wt.Header().Get("Content-Type")
	if ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
}

func TestShare_Thumbnail_shouldFallback(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	ts := store.NewThumbnailStore(db)
	u, _ := us.Create("shfallbk_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "/uploads/photo.jpg",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       "jkl012",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	thumbDir := filepath.Join(srv.cfg.ThumbnailsDir(), file.ID)
	os.MkdirAll(thumbDir, 0755)
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
	os.WriteFile(filepath.Join(thumbDir, "sm.jpg"), jpegData, 0644)

	thumb := &model.Thumbnail{
		FileID:    file.ID,
		Size:      model.ThumbSizeSmall,
		Width:     60,
		Height:    40,
		Format:    "jpeg",
		LocalPath: filepath.Join(thumbDir, "sm.jpg"),
		SizeBytes: int64(len(jpegData)),
	}
	ts.Create(thumb)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wt := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/thumb/"+file.ID+"/md.jpg", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wt.Code != http.StatusOK {
		t.Fatalf("expected 200 for md.jpg fallback to sm.jpg, got %d: %s", wt.Code, wt.Body.String())
	}
}

func TestShare_Thumbnail_shouldReturn404WhenNoThumbnail(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shnothumb_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "/uploads/photo.jpg",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       "mno345",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wt := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/thumb/"+file.ID+"/sm.jpg", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wt.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", wt.Code, wt.Body.String())
	}
}

func TestShare_Thumbnail_shouldReturn404ForUnknownShare(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/share/bad-token/thumb/somefile/sm.jpg", "", map[string]string{})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestShare_GetFile_shouldReturn404ForNonexistentFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shnf_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wg := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files/nonexistent-id", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wg.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", wg.Code)
	}
}

func TestShare_ListFiles_shouldReturnFiles(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	folderID := f.ID
	for i := 0; i < 3; i++ {
		file := &model.File{
			UserID:       u.ID,
			Filename:     "photo" + string(rune('a'+i)) + ".jpg",
			OriginalName: "photo" + string(rune('a'+i)) + ".jpg",
			Path:         "/uploads/photo.jpg",
			SizeBytes:    1024,
			MimeType:     "image/jpeg",
			SHA256:       "sha" + string(rune('a'+i)),
			MediaType:    model.MediaTypePhoto,
			FolderID:     &folderID,
		}
		fs.Create(file)
	}

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wl := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files?limit=50", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wl.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wl.Code, wl.Body.String())
	}
	var listResp map[string]interface{}
	json.Unmarshal(wl.Body.Bytes(), &listResp)
	items := listResp["items"].([]interface{})
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestShare_ListFiles_shouldReturn403WithoutSessionToken(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shlnosess_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wl := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files", "", map[string]string{})
	if wl.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", wl.Code)
	}
}

func TestShare_ListFolders_shouldReturnFolders(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shlistfd_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	parent, _ := folders.Create(u.ID, "Parent", nil)
	parentID := parent.ID
	folders.Create(u.ID, "Subfolder", &parentID)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+parent.ID+"/shares", `{"permissions":"read","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wl := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/folders", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wl.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", wl.Code, wl.Body.String())
	}
	var listResp map[string]interface{}
	json.Unmarshal(wl.Body.Bytes(), &listResp)
	items := listResp["folders"].([]interface{})
	if len(items) == 0 {
		t.Error("expected at least 1 folder")
	}
}

func TestShare_DeleteFile_shouldDeleteFile(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shdelfile_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "todelete.jpg",
		OriginalName: "todelete.jpg",
		Path:         "/uploads/todelete.jpg",
		SizeBytes:    512,
		MimeType:     "image/jpeg",
		SHA256:       "shdelfilesha",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_write"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wd := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/files/"+file.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wd.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", wd.Code, wd.Body.String())
	}
}

func TestShare_DeleteFile_shouldRejectReadOnly(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("shdelreado_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)

	folderID := f.ID
	file := &model.File{
		UserID:       u.ID,
		Filename:     "keepme.jpg",
		OriginalName: "keepme.jpg",
		Path:         "/uploads/keepme.jpg",
		SizeBytes:    512,
		MimeType:     "image/jpeg",
		SHA256:       "shdelrosha",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &folderID,
	}
	fs.Create(file)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wd := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/files/"+file.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wd.Code != http.StatusForbidden {
		t.Errorf("expected 403 for read-only share, got %d", wd.Code)
	}
}

func TestShare_CreateFolder_shouldCreateFolder(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shcrfold_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Parent", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_write","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wc := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/folders", `{"name":"New Subfolder"}`, map[string]string{
		"X-Share-Session-Token": sessionToken,
		"Content-Type":          "application/json",
	})
	if wc.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", wc.Code, wc.Body.String())
	}
}

func TestShare_DeleteFolder_shouldDeleteFolder(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shdelfold_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	parent, _ := folders.Create(u.ID, "Parent", nil)
	parentID := parent.ID
	child, _ := folders.Create(u.ID, "Child", &parentID)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+parent.ID+"/shares", `{"permissions":"read_write","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var shareResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &shareResp)
	shareToken := shareResp["token"].(string)

	wu := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", "", map[string]string{
		"Content-Type": "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(wu.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	wd := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/folders/"+child.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if wd.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", wd.Code, wd.Body.String())
	}
}
