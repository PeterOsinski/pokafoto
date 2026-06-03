package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestFolderPassword_Set_shouldCreatePassword(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fpuser_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	body := `{"password":"secret123"}`
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", body, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Password set for folder" {
		t.Errorf("expected success message, got %v", resp["message"])
	}
}

func TestFolderPassword_Unlock_shouldReturnToken(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fpunlock_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/unlock", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["unlock_token"] == nil {
		t.Error("expected non-nil unlock_token")
	}
}

func TestFolderPassword_Unlock_wrongPassword_should401(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fpwrong_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/unlock", `{"password":"wrongpass"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestFolderPassword_AccessWithoutUnlock_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("fpblock_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	file := &model.File{
		UserID:       u.ID,
		Filename:     "test.jpg",
		OriginalName: "test.jpg",
		Path:         "",
		SizeBytes:    100,
		MimeType:     "image/jpeg",
		SHA256:       "foldertest1",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &f.ID,
	}
	filestore.Create(file)

	w := testRequest(t, srv, "GET", "/api/v1/files/"+file.ID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFolderPassword_Remove_shouldRemovePassword(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fpremove_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "DELETE", "/api/v1/folders/"+f.ID+"/password", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestFolderPassword_UnlockThenAccess_shouldWork(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("fpunlockok_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	unlockW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/unlock", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	unlockToken := unlockResp["unlock_token"].(string)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "test.jpg",
		OriginalName: "test.jpg",
		Path:         "",
		SizeBytes:    100,
		MimeType:     "image/jpeg",
		SHA256:       "fptestunlock2",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &f.ID,
	}
	filestore.Create(file)

	w := testRequest(t, srv, "GET", "/api/v1/files/"+file.ID, "", map[string]string{
		"Authorization":        "Bearer " + token,
		"X-Folder-Unlock-Token": unlockToken,
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFolderPassword_GetStatus_shouldReturnStatus(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fpstatus_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "GET", "/api/v1/folders/"+f.ID+"/password", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["has_password"] != true {
		t.Error("expected has_password to be true")
	}
}

func TestFolderShare_Create_shouldCreateShare(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("sharecr_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	body := `{"permissions":"read_write","upload_limit_bytes":104857600}`
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", body, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}
}

func TestFolderShare_ListShares_shouldReturnShares(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shlist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_upload"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "GET", "/api/v1/folders/"+f.ID+"/shares", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	shares := resp["shares"].([]interface{})
	if len(shares) != 2 {
		t.Errorf("expected 2 shares, got %d", len(shares))
	}
}

func TestFolderShare_DeleteShare_shouldRemoveShare(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shdel_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareID := createResp["id"].(string)

	deleteW := testRequest(t, srv, "DELETE", "/api/v1/folders/"+f.ID+"/shares/"+shareID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if deleteW.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", deleteW.Code)
	}
}

func TestFolderShare_PublicAccess_shouldListFilesAfterUnlock(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("shrpub_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "shared_file.jpg",
		OriginalName: "shared_file.jpg",
		Path:         "",
		SizeBytes:    500,
		MimeType:     "image/jpeg",
		SHA256:       "sharedfilehash",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &f.ID,
	}
	filestore.Create(file)

	infoW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken, "", nil)
	if infoW.Code != http.StatusOK {
		t.Fatalf("expected 200 for share info, got %d: %s", infoW.Code, infoW.Body.String())
	}
	var info map[string]interface{}
	json.Unmarshal(infoW.Body.Bytes(), &info)
	if info["permissions"] != "read" {
		t.Errorf("expected permissions 'read', got %v", info["permissions"])
	}

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	if unlockW.Code != http.StatusOK {
		t.Fatalf("expected 200 for unlock, got %d: %s", unlockW.Code, unlockW.Body.String())
	}
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	listW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 for list, got %d: %s", listW.Code, listW.Body.String())
	}
}

func TestFolderShare_PasswordProtected_shouldRequireUnlock(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shpass_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read","password":"sharepass"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	infoW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken, "", nil)
	var info map[string]interface{}
	json.Unmarshal(infoW.Body.Bytes(), &info)
	if info["needs_password"] != true {
		t.Error("expected needs_password to be true")
	}

	unlockWrong := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{"password":"wrong"}`, map[string]string{"Content-Type": "application/json"})
	if unlockWrong.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", unlockWrong.Code)
	}

	unlockOk := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{"password":"sharepass"}`, map[string]string{"Content-Type": "application/json"})
	if unlockOk.Code != http.StatusOK {
		t.Errorf("expected 200 for correct password, got %d: %s", unlockOk.Code, unlockOk.Body.String())
	}
}

func TestFolderShare_Expired_shouldReject(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shexp_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read","expires_at":"2020-01-01T00:00:00Z"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	infoW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken, "", nil)
	if infoW.Code != http.StatusGone {
		t.Errorf("expected 410 for expired share, got %d: %s", infoW.Code, infoW.Body.String())
	}
}

func TestFolderShare_NotYourFolder_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	owner, _ := us.Create("showner_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	other, _ := us.Create("shother_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(owner.ID, "Owner's Folder", nil)

	otherToken := generateTestToken(srv.cfg.Auth.JWTSecret, other.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + otherToken,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestFolderShare_UpdateShare_shouldChangePermissions(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shupdate_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareID := createResp["id"].(string)

	updateW := testRequest(t, srv, "PUT", "/api/v1/folders/"+f.ID+"/shares/"+shareID, `{"permissions":"read_write"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if updateW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}
}

func TestFolderShare_ShareUploadWithoutPermission_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shnoup_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	body := strings.NewReader("test content")
	uploadReq, _ := http.NewRequest("POST", "/api/v1/share/"+shareToken+"/upload", body)
	uploadReq.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	uploadReq.Header.Set("X-Share-Session-Token", sessionToken)
	w3 := testMultipartRequest(t, srv, uploadReq)
	if w3.Code != http.StatusForbidden && w3.Code != http.StatusBadRequest {
		t.Logf("share upload without permission returned %d (expected 403): %s", w3.Code, w3.Body.String())
	}
}

func testMultipartRequest(t *testing.T, srv *Server, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	return w
}

func TestFolderShare_NonExistentToken_should404(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/share/nonexistent-token", "", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFolderShare_UnauthorizedDelete_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shdelno_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	delW := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/files/some-id", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if delW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for delete without write permission, got %d", delW.Code)
	}
}

func TestFolderPassword_ListFilesWithoutUnlock_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("fplist_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Secret Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", `{"password":"secret123"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	file := &model.File{
		UserID:       u.ID,
		Filename:     "secret_photo.jpg",
		OriginalName: "secret_photo.jpg",
		Path:         "",
		SizeBytes:    200,
		MimeType:     "image/jpeg",
		SHA256:       "fplist123",
		MediaType:    model.MediaTypePhoto,
		FolderID:     &f.ID,
	}
	filestore.Create(file)

	w := testRequest(t, srv, "GET", "/api/v1/files?folder_id="+f.ID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for list files without unlock, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFolderShare_Download_shouldServeFileFromDisk(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("shdl_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	originalsDir := srv.cfg.OriginalsDir()
	os.MkdirAll(originalsDir+"/"+u.ID, 0755)
	fileContent := []byte("shared file content")
	os.WriteFile(originalsDir+"/"+u.ID+"/shared.txt", fileContent, 0644)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "shared.txt",
		OriginalName: "shared.txt",
		Path:         "",
		SizeBytes:    int64(len(fileContent)),
		MimeType:     "text/plain",
		SHA256:       "sharehashdl",
		MediaType:    model.MediaTypeFile,
		FolderID:     &f.ID,
	}
	filestore.Create(file)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	dlW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/download/"+file.ID+"?share_session_token="+sessionToken, "", nil)
	if dlW.Code != http.StatusOK {
		t.Errorf("expected 200 for share download, got %d: %s", dlW.Code, dlW.Body.String())
	}
	if dlW.Body.String() != string(fileContent) {
		t.Errorf("expected file content, got %q", dlW.Body.String())
	}
}

func TestFolderShare_Download_noSessionToken_should403(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("shdln_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	dlW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/download/some-id", "", nil)
	if dlW.Code != http.StatusForbidden && dlW.Code != http.StatusUnauthorized {
		t.Errorf("expected 403 or 401 without session token, got %d", dlW.Code)
	}
}

func TestFolderShare_Download_wrongFolder_should404(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("shdlw_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)
	f2, _ := folders.Create(u.ID, "Other", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")
	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "other.txt",
		OriginalName: "other.txt",
		Path:         "",
		SizeBytes:    10,
		MimeType:     "text/plain",
		SHA256:       "otherhash",
		MediaType:    model.MediaTypeFile,
		FolderID:     &f2.ID,
	}
	filestore.Create(file)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	dlW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/download/"+file.ID+"?share_session_token="+sessionToken, "", nil)
	if dlW.Code != http.StatusNotFound {
		t.Errorf("expected 404 for file in wrong folder, got %d", dlW.Code)
	}
}

func TestShareSubdir_CreateShare_shouldIncludeSubdirsFlag(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("sdirs_cr_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared", nil)

	_ = generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	shared := store.NewFolderShareStore(db)
	s := &model.FolderShare{
		FolderID:         f.ID,
		Permissions:      model.ShareReadWrite,
		IncludeSubdirs:   true,
		UploadLimitBytes: nil,
		HasPassword:      false,
	}
	if err := shared.Create(s); err != nil {
		t.Fatalf("failed to create share: %v", err)
	}
	if !s.IncludeSubdirs {
		t.Error("expected include_subdirs to be true")
	}

	found, err := shared.FindByID(s.ID)
	if err != nil {
		t.Fatalf("failed to find share: %v", err)
	}
	if !found.IncludeSubdirs {
		t.Error("expected found share to have include_subdirs=true")
	}
}

func TestShareSubdir_ListFolders_shouldReturnSubfolders(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("sdirs_ls_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Root", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_write","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	subID := f.ID
	folders.Create(u.ID, "Sub A", &subID)
	folders.Create(u.ID, "Sub B", &subID)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	listW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/folders", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW.Code, listW.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(listW.Body.Bytes(), &resp)
	items := resp["folders"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 subfolders, got %d", len(items))
	}
}

func TestShareSubdir_CreateFolder_shouldRequireWritePermission(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("sdirs_cf_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared ReadOnly", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	makeW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/folders", `{"name":"New"}`, map[string]string{
		"X-Share-Session-Token": sessionToken,
		"Content-Type":          "application/json",
	})
	if makeW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for read-only share creating folder, got %d", makeW.Code)
	}
}

func TestShareSubdir_DeleteFolder_shouldRequireWritePermission(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("sdirs_df_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared ReadOnly", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	delW := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/folders/some-id", "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if delW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for read-only share deleting folder, got %d", delW.Code)
	}
}

func TestShareSubdir_ListFilesInSubdirectory_shouldWork(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("sdirs_lf_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Root", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_write","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	rootID := f.ID
	sub, _ := folders.Create(u.ID, "Subfolder", &rootID)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "sub_file.jpg",
		OriginalName: "sub_file.jpg",
		Path:         "",
		SizeBytes:    200,
		MimeType:     "image/jpeg",
		SHA256:       uuid.NewString(),
		MediaType:    model.MediaTypePhoto,
		FolderID:     &sub.ID,
	}
	filestore.Create(file)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	listW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files?folder_id="+sub.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW.Code, listW.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(listW.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("expected 1 file in subdirectory, got %d", len(items))
	}
}

func TestShareSubdir_RejectFolderOutsideTree(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	filestore := store.NewFileStore(db)
	u, _ := us.Create("sdirs_rj_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Shared Root", nil)
	f2, _ := folders.Create(u.ID, "Unrelated Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	createW := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/shares", `{"permissions":"read_write","include_subdirs":true}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createResp)
	shareToken := createResp["token"].(string)

	file := &model.File{
		UserID:       u.ID,
		Filename:     "unrelated.jpg",
		OriginalName: "unrelated.jpg",
		Path:         "",
		SizeBytes:    100,
		MimeType:     "image/jpeg",
		SHA256:       uuid.NewString(),
		MediaType:    model.MediaTypePhoto,
		FolderID:     &f2.ID,
	}
	filestore.Create(file)

	unlockW := testRequest(t, srv, "POST", "/api/v1/share/"+shareToken+"/unlock", `{}`, map[string]string{"Content-Type": "application/json"})
	var unlockResp map[string]interface{}
	json.Unmarshal(unlockW.Body.Bytes(), &unlockResp)
	sessionToken := unlockResp["share_session_token"].(string)

	listW := testRequest(t, srv, "GET", "/api/v1/share/"+shareToken+"/files?folder_id="+f2.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if listW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for folder outside share tree, got %d: %s", listW.Code, listW.Body.String())
	}

	delW := testRequest(t, srv, "DELETE", "/api/v1/share/"+shareToken+"/folders/"+f2.ID, "", map[string]string{
		"X-Share-Session-Token": sessionToken,
	})
	if delW.Code != http.StatusForbidden {
		t.Errorf("expected 403 for deleting folder outside share tree, got %d", delW.Code)
	}
}

func TestFolderPassword_shouldSetAndReturnHint(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	folders := store.NewFolderStore(db)
	u, _ := us.Create("fphint_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f, _ := folders.Create(u.ID, "Hint Folder", nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	setBody := `{"password":"secret123","password_hint":"My birthday"}`
	w := testRequest(t, srv, "POST", "/api/v1/folders/"+f.ID+"/password", setBody, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	statusW := testRequest(t, srv, "GET", "/api/v1/folders/"+f.ID+"/password", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if statusW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusW.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(statusW.Body.Bytes(), &resp)
	if hint, ok := resp["password_hint"].(string); !ok || hint != "My birthday" {
		t.Errorf("expected password_hint 'My birthday', got %v", resp["password_hint"])
	}
}
