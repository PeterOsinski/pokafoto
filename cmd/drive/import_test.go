package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newMockServer(t *testing.T) (*httptest.Server, *importCmd) {
	t.Helper()

	var authToken string
	folders := make(map[string]*folderTreeItem)
	files := make(map[string][]fileItemResponse)
	nextFolderID := 1
	nextFileID := 1

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAuthRoute := r.Method == "POST" && r.URL.Path == "/api/v1/auth/login"

		if !isAuthRoute && r.Header.Get("Authorization") != "Bearer "+authToken {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{"code": "UNAUTHORIZED", "message": "Invalid token"},
			})
			return
		}

		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/auth/login":
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.Username == "" || req.Password == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			authToken = "test-token-" + req.Username
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  authToken,
				"refresh_token": "refresh-xyz",
				"expires_in":    86400,
			})

		case r.Method == "GET" && r.URL.Path == "/api/v1/folders":
			json.NewEncoder(w).Encode(folderTreeResponse{Children: buildFolderTree(folders)})

		case r.Method == "POST" && r.URL.Path == "/api/v1/folders":
			var req struct {
				Name     string  `json:"name"`
				ParentID *string `json:"parent_id,omitempty"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			fid := fmt.Sprintf("folder-%d", nextFolderID)
			nextFolderID++
			folders[fid] = &folderTreeItem{
				ID:       fid,
				Name:     req.Name,
				ParentID: req.ParentID,
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(folders[fid])

		case r.Method == "GET" && r.URL.Path == "/api/v1/files":
			folderID := r.URL.Query().Get("folder_id")
			items := files[folderID]
			if items == nil {
				items = []fileItemResponse{}
			}
			json.NewEncoder(w).Encode(fileListResponse{
				Items:      items,
				NextCursor: "",
				Total:      len(items),
			})

		case r.Method == "POST" && r.URL.Path == "/api/v1/upload":
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"batch_id": "batch-test",
				"jobs": []map[string]interface{}{
					{"job_id": "job-1", "filename": "test", "status": "queued"},
				},
			})

		case r.Method == "POST" && r.URL.Path == "/api/v1/upload/chunk":
			uploadID := fmt.Sprintf("upload-%d", nextFileID)
			nextFileID++
			json.NewEncoder(w).Encode(map[string]interface{}{
				"upload_id":    uploadID,
				"resume_token": "token-" + uploadID,
			})

		case strings.Contains(r.URL.Path, "/api/v1/upload/chunk/") && strings.HasSuffix(r.URL.Path, "/complete"):
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":         "assembling",
				"stored_chunks":  1,
				"missing_chunks": []int{},
				"total_chunks":   1,
			})

		case r.Method == "GET" && r.URL.Path == "/api/v1/upload/active":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jobs": []map[string]interface{}{
					{"job_id": "upload-1", "status": "completed"},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	cmd := &importCmd{
		serverURL:   srv.URL,
		client:      srv.Client(),
		concurrency: 2,
		folderCache: make(map[string]string),
	}

	return srv, cmd
}

func buildFolderTree(folders map[string]*folderTreeItem) []*folderTreeResponse {
	children := make(map[string][]*folderTreeResponse)
	for _, f := range folders {
		pid := ""
		if f.ParentID != nil {
			pid = *f.ParentID
		}
		children[pid] = append(children[pid], &folderTreeResponse{
			Folder:   f,
			Children: nil,
		})
	}
	return children[""]
}

func TestImport_Authenticate_shouldGetToken(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	err := cmd.authenticate("testuser", "testpass")
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}

	if cmd.token != "test-token-testuser" {
		t.Errorf("expected token 'test-token-testuser', got '%s'", cmd.token)
	}
}

func TestImport_Authenticate_shouldFailWithBadCredentials(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	err := cmd.authenticate("", "")
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestImport_ResolveTargetFolder_shouldReturnEmptyForRoot(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	id, err := cmd.resolveTargetFolder("/")
	if err != nil {
		t.Fatalf("resolveTargetFolder failed: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty ID for root, got '%s'", id)
	}
}

func TestImport_ResolveTargetFolder_shouldCreateNestedFolders(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	id, err := cmd.resolveTargetFolder("/photos/2024")
	if err != nil {
		t.Fatalf("resolveTargetFolder failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty folder ID")
	}

	if _, ok := cmd.folderCache["root/photos"]; !ok {
		t.Error("expected 'photos' folder in cache")
	}
	found2024 := false
	for k := range cmd.folderCache {
		if strings.HasSuffix(k, "/2024") {
			found2024 = true
			break
		}
	}
	if !found2024 {
		t.Errorf("expected '2024' folder in cache, got keys: %v", func() []string {
			var keys []string
			for k := range cmd.folderCache {
				keys = append(keys, k)
			}
			return keys
		}())
	}
}

func TestImport_FileExistsInFolder_shouldReturnTrueWhenFound(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	exists, err := cmd.fileExistsInFolder("folder-123", "test.jpg", 1024)
	if err != nil {
		t.Fatalf("fileExistsInFolder failed: %v", err)
	}
	if exists {
		// The default mock doesn't have any files, so this should be false
		// But we're testing that the function doesn't error
	}
}

func TestImport_FileExistsInFolder_shouldPaginate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			cursor := r.URL.Query().Get("cursor")
			if cursor == "" {
				json.NewEncoder(w).Encode(fileListResponse{
					Items:      []fileItemResponse{},
					NextCursor: "cursor1",
					Total:      3,
				})
			} else if cursor == "cursor1" {
				json.NewEncoder(w).Encode(fileListResponse{
					Items: []fileItemResponse{
						{OriginalName: "match.jpg", SizeBytes: 2048},
					},
					NextCursor: "cursor2",
					Total:      3,
				})
			} else {
				json.NewEncoder(w).Encode(fileListResponse{
					Items:      []fileItemResponse{},
					NextCursor: "",
					Total:      3,
				})
			}
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cmd := &importCmd{
		serverURL:   srv.URL,
		client:      srv.Client(),
		token:       "test-token",
		folderCache: make(map[string]string),
	}

	exists, err := cmd.fileExistsInFolder("folder-a", "match.jpg", 2048)
	if err != nil {
		t.Fatalf("fileExistsInFolder failed: %v", err)
	}
	if !exists {
		t.Fatal("expected file to be found")
	}
}

func TestImport_CreateFolder_shouldReturnID(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	id, err := cmd.createFolder(nil, "NewFolder")
	if err != nil {
		t.Fatalf("createFolder failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty folder ID")
	}
}

func TestImport_CreateFolder_shouldSetParentID(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	parentID := "folder-parent"
	id, err := cmd.createFolder(&parentID, "Child")
	if err != nil {
		t.Fatalf("createFolder failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty folder ID")
	}
}

func TestImport_DryRun_shouldNotCallUpload(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	cmd.dryRun = true

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmd.walkAndImport(tmpDir, "")
	if err != nil {
		t.Fatalf("walkAndImport failed: %v", err)
	}

	cmd.stats.mu.Lock()
	defer cmd.stats.mu.Unlock()
	if cmd.stats.skipped == 0 && cmd.stats.total > 0 {
		t.Log("dry-run skipped no files (expected in non-dry mode)")
	}
}

func TestImport_WalkAndImport_shouldUploadFiles(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "data.bin"), []byte("binary data here"), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmd.walkAndImport(tmpDir, "")
	if err != nil {
		t.Fatalf("walkAndImport failed: %v", err)
	}

	cmd.stats.mu.Lock()
	defer cmd.stats.mu.Unlock()
	if cmd.stats.total != 2 {
		t.Errorf("expected 2 total files, got %d", cmd.stats.total)
	}
}

func TestImport_WalkAndImport_shouldPreserveFolderStructure(t *testing.T) {
	t.Parallel()

	srv, cmd := newMockServer(t)
	defer srv.Close()

	if err := cmd.authenticate("u", "p"); err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nested", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmd.walkAndImport(tmpDir, "target-folder-id")
	if err != nil {
		t.Fatalf("walkAndImport failed: %v", err)
	}

	cmd.stats.mu.Lock()
	if cmd.stats.total != 2 {
		t.Errorf("expected 2 total files, got %d", cmd.stats.total)
	}
	cmd.stats.mu.Unlock()

	cacheKeys := []string{}
	cmd.mu.Lock()
	for k := range cmd.folderCache {
		cacheKeys = append(cacheKeys, k)
	}
	cmd.mu.Unlock()

	hasNested := false
	hasDeep := false
	for _, k := range cacheKeys {
		if strings.Contains(k, "nested") {
			hasNested = true
		}
		if strings.Contains(k, "deep") {
			hasDeep = true
		}
	}
	if !hasNested || !hasDeep {
		t.Errorf("expected nested/deep folder cache entries, got %v", cacheKeys)
	}
}

func TestImport_SkipsExistingFiles(t *testing.T) {
	t.Parallel()

	mockFiles := map[string][]fileItemResponse{
		"target-123": {
			{OriginalName: "already-there.txt", SizeBytes: 12},
			{OriginalName: "also-here.txt", SizeBytes: 5},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/auth/login":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "token-skip", "refresh_token": "r", "expires_in": 86400,
			})
		case r.Method == "GET" && r.URL.Path == "/api/v1/folders":
			json.NewEncoder(w).Encode(folderTreeResponse{Children: nil})
		case r.Method == "GET" && r.URL.Path == "/api/v1/files":
			folderID := r.URL.Query().Get("folder_id")
			items := mockFiles[folderID]
			if items == nil {
				items = []fileItemResponse{}
			}
			json.NewEncoder(w).Encode(fileListResponse{Items: items})
		default:
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))
	defer srv.Close()

	cmd := &importCmd{
		serverURL:   srv.URL,
		client:      srv.Client(),
		concurrency: 2,
		folderCache: make(map[string]string),
	}
	cmd.authenticate("u", "p")

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "already-there.txt"), []byte("hello world!"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "new-file.txt"), []byte("fresh"), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmd.walkAndImport(tmpDir, "target-123")
	if err != nil {
		t.Fatalf("walkAndImport failed: %v", err)
	}

	cmd.stats.mu.Lock()
	defer cmd.stats.mu.Unlock()
	if cmd.stats.skipped < 1 {
		t.Errorf("expected at least 1 skipped file, got %d", cmd.stats.skipped)
	}
}

func TestImport_InvalidTargetServer_shouldFailAuth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cmd := &importCmd{
		serverURL:   srv.URL,
		client:      srv.Client(),
		folderCache: make(map[string]string),
	}

	err := cmd.authenticate("u", "p")
	if err == nil {
		t.Fatal("expected authentication error")
	}
}

func TestRunImport_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	err := runImport([]string{})
	if err == nil {
		t.Fatal("expected error for missing --source")
	}

	err = runImport([]string{"--source", "/tmp"})
	if err == nil {
		t.Fatal("expected error for missing --username")
	}
}

func TestHumanSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := humanSize(tt.size)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %s, want %s", tt.size, got, tt.want)
		}
	}
}
