package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestDocuments_Create_shouldCreateAndReturnDocument(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-create-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"My Note"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["originalName"] != "My Note.md" {
		t.Errorf("expected 'My Note.md', got %v", resp["originalName"])
	}
	if resp["content"] != "" {
		t.Errorf("expected empty content, got %v", resp["content"])
	}
}

func TestDocuments_Create_shouldCreateWithFolder(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFolderStore(db)
	u, _ := us.Create("doc-folder-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	folder, _ := fs.Create(u.ID, "Notes", nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	body := `{"name":"My Note","folder_id":"` + folder.ID + `"}`
	w := testRequest(t, srv, "POST", "/api/v1/documents", body, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDocuments_Create_shouldRejectEmptyName(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-empty-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":""}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDocuments_Create_shouldRequireAuth(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Test"}`, map[string]string{
		"Content-Type": "application/json",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDocuments_Get_shouldReturnContent(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-get-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Readme"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	w = testRequest(t, srv, "GET", "/api/v1/documents/"+docID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp["originalName"] != "Readme.md" {
		t.Errorf("expected 'Readme.md', got %v", getResp["originalName"])
	}
}

func TestDocuments_Update_shouldSaveContent(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-update-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Test"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	w = testRequest(t, srv, "PUT", "/api/v1/documents/"+docID, `{"content":"# Hello World"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	w = testRequest(t, srv, "GET", "/api/v1/documents/"+docID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	var getResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp["content"] != "# Hello World" {
		t.Errorf("expected '# Hello World', got %v", getResp["content"])
	}
}

func TestDocuments_Update_shouldRejectNonDocument(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	u, _ := us.Create("doc-nondoc-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	f := &model.File{
		UserID: u.ID, Filename: "2024/test.jpg", OriginalName: "test.jpg", Path: "/2024",
		SizeBytes: 100, MimeType: "image/jpeg", SHA256: uuid.NewString(), MediaType: model.MediaTypePhoto,
	}
	fs.Create(f)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "PUT", "/api/v1/documents/"+f.ID, `{"content":"test"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDocuments_Update_shouldRequireOwnership(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u1, _ := us.Create("doc-owner-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	u2, _ := us.Create("doc-other-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token1 := generateTestToken(srv.cfg.Auth.JWTSecret, u1.ID, "member")
	token2 := generateTestToken(srv.cfg.Auth.JWTSecret, u2.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Private"}`, map[string]string{
		"Authorization": "Bearer " + token1,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	w = testRequest(t, srv, "PUT", "/api/v1/documents/"+docID, `{"content":"hacked"}`, map[string]string{
		"Authorization": "Bearer " + token2,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDocuments_Delete_shouldSoftDeleteDocument(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-del-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"DeleteMe"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	w = testRequest(t, srv, "DELETE", "/api/v1/documents/"+docID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	fs := store.NewFileStore(db)
	f, _ := fs.FindByID(docID)
	if f == nil || !f.IsDeleted {
		t.Error("expected file to be soft-deleted")
	}
}

func TestDocuments_Download_shouldServeDocumentContent(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-dl-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Export"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	testRequest(t, srv, "PUT", "/api/v1/documents/"+docID, `{"content":"# Download Test"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w = testRequest(t, srv, "GET", "/api/v1/download/"+docID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "# Download Test" {
		t.Errorf("expected '# Download Test', got %q", w.Body.String())
	}
	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header")
	}
	if w.Header().Get("Content-Type") != "text/markdown" {
		t.Errorf("expected text/markdown, got %s", w.Header().Get("Content-Type"))
	}
}

func TestDocuments_ListFiles_shouldIncludeIsAppManaged(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-list-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Visible"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})

	w := testRequest(t, srv, "GET", "/api/v1/files", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) == 0 {
		t.Fatal("expected at least 1 file")
	}
	item := items[0].(map[string]interface{})
	if appManaged, ok := item["isAppManaged"]; !ok || appManaged != true {
		t.Errorf("expected isAppManaged=true, got %v", item["isAppManaged"])
	}
	if item["mimeType"] != "text/markdown" {
		t.Errorf("expected text/markdown, got %v", item["mimeType"])
	}
}

func TestDocuments_GetFile_shouldIncludeIsAppManaged(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("doc-detail-"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/documents", `{"name":"Detail"}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	docID := createResp["id"].(string)

	w = testRequest(t, srv, "GET", "/api/v1/files/"+docID, "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if appManaged, ok := resp["isAppManaged"]; !ok || appManaged != true {
		t.Errorf("expected isAppManaged=true, got %v", resp["isAppManaged"])
	}
}
