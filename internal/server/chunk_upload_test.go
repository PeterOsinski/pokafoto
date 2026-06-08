package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func setChunkHeaders(r *http.Request, filename string, totalSize, totalChunks, chunkIndex, chunkSize int, skipDedup bool) {
	r.Header.Set("X-Filename", filename)
	r.Header.Set("X-Total-Size", itoa(totalSize))
	r.Header.Set("X-Total-Chunks", itoa(totalChunks))
	r.Header.Set("X-Chunk-Index", itoa(chunkIndex))
	r.Header.Set("X-Chunk-Size", itoa(chunkSize))
	if skipDedup {
		r.Header.Set("X-Skip-Name-Size-Dedup", "true")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func authHeader(srv *Server, userID string) string {
	return "Bearer " + generateTestToken(srv.cfg.Auth.JWTSecret, userID, "member")
}

func TestServer_ChunkUpload_firstChunk_shouldStartSession(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkfirst_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	totalSize := 1024
	totalChunks := 1
	chunkData := make([]byte, 1024)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}

	reqBody := bytes.NewReader(chunkData)
	req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", reqBody)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", authHeader(srv, u.ID))
	setChunkHeaders(req, "test.bin", totalSize, totalChunks, 0, len(chunkData), true)

	w := serveRequest(srv, req)

	if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
		t.Fatalf("expected 200 or 202, got %d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		UploadID     string `json:"upload_id"`
		ResumeToken  string `json:"resume_token"`
		StoredChunks []int  `json:"stored_chunks"`
		MissingChunks []int `json:"missing_chunks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if res.UploadID == "" {
		t.Error("expected upload_id in response")
	}
	if res.ResumeToken == "" {
		t.Error("expected resume_token in response")
	}
	if len(res.StoredChunks) != 1 || res.StoredChunks[0] != 0 {
		t.Errorf("expected stored_chunks [0], got %v", res.StoredChunks)
	}
}

func TestServer_ChunkUpload_multipleChunks_shouldTrackAll(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkmulti_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	chunkSize := 512
	totalChunks := 3
	totalSize := chunkSize * totalChunks

	chunkData := make([]byte, chunkSize)
	for i := range chunkData {
		chunkData[i] = byte(i % 256)
	}

	var uploadID string
	var resumeToken string

	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
		reqBody := bytes.NewReader(chunkData)
		req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", reqBody)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", authHeader(srv, u.ID))
		setChunkHeaders(req, "multi.bin", totalSize, totalChunks, chunkIndex, chunkSize, true)
		if resumeToken != "" {
			req.Header.Set("X-Resume-Token", resumeToken)
		}

		w := serveRequest(srv, req)
		if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
			t.Fatalf("chunk %d: expected 200 or 202, got %d body=%s", chunkIndex, w.Code, w.Body.String())
		}

		var res struct {
			UploadID      string `json:"upload_id"`
			ResumeToken   string `json:"resume_token"`
			StoredChunks  []int  `json:"stored_chunks"`
			MissingChunks []int  `json:"missing_chunks"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("chunk %d: parse response: %v", chunkIndex, err)
		}
		uploadID = res.UploadID
		resumeToken = res.ResumeToken
	}

	if len(uploadID) == 0 {
		t.Fatal("expected upload_id")
	}

	cs := store.NewChunkStore(db, service.NewMockFS())
	count, err := cs.GetStoredChunkCount(uploadID)
	if err != nil {
		t.Fatalf("get stored chunk count: %v", err)
	}
	if count != totalChunks {
		t.Errorf("expected %d stored chunks, got %d", totalChunks, count)
	}
}

func TestServer_ChunkComplete_allChunks_shouldReturnNoMissing(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkcomplete_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	chunkSize := 512
	totalChunks := 2
	totalSize := chunkSize * totalChunks

	chunkData := make([]byte, chunkSize)
	var resumeToken string
	var uploadID string

	for chunkIndex := 0; chunkIndex < totalChunks; chunkIndex++ {
		reqBody := bytes.NewReader(chunkData)
		req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", reqBody)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", authHeader(srv, u.ID))
		setChunkHeaders(req, "complete.bin", totalSize, totalChunks, chunkIndex, chunkSize, true)
		if resumeToken != "" {
			req.Header.Set("X-Resume-Token", resumeToken)
		}

		w := serveRequest(srv, req)
		if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
			t.Fatalf("chunk %d: expected 200 or 202, got %d", chunkIndex, w.Code)
		}
		var res struct {
			UploadID    string `json:"upload_id"`
			ResumeToken string `json:"resume_token"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		resumeToken = res.ResumeToken
		uploadID = res.UploadID
	}

	completeBody, _ := json.Marshal(map[string]string{"upload_id": uploadID})
	req := makeTestRequest(t, "POST", "/api/v1/upload/chunk/"+resumeToken+"/complete", bytes.NewReader(completeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(srv, u.ID))

	w := serveRequest(srv, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		StoredChunks  int   `json:"stored_chunks"`
		MissingChunks []int `json:"missing_chunks"`
		TotalChunks   int   `json:"total_chunks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("parse complete response: %v", err)
	}
	if res.StoredChunks != totalChunks {
		t.Errorf("expected stored_chunks=%d, got %d", totalChunks, res.StoredChunks)
	}
	if len(res.MissingChunks) != 0 {
		t.Errorf("expected 0 missing chunks, got %v", res.MissingChunks)
	}
}

func TestServer_ChunkComplete_partialChunks_shouldReturnMissing(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkpartial_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	chunkSize := 512
	totalChunks := 3
	totalSize := chunkSize * totalChunks

	chunkData := make([]byte, chunkSize)
	var resumeToken string
	var uploadID string

	for chunkIndex := 0; chunkIndex < 2; chunkIndex++ {
		reqBody := bytes.NewReader(chunkData)
		req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", reqBody)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", authHeader(srv, u.ID))
		setChunkHeaders(req, "partial.bin", totalSize, totalChunks, chunkIndex, chunkSize, true)
		if resumeToken != "" {
			req.Header.Set("X-Resume-Token", resumeToken)
		}

		w := serveRequest(srv, req)
		var res struct {
			UploadID    string `json:"upload_id"`
			ResumeToken string `json:"resume_token"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		resumeToken = res.ResumeToken
		uploadID = res.UploadID
	}

	completeBody, _ := json.Marshal(map[string]string{"upload_id": uploadID})
	req := makeTestRequest(t, "POST", "/api/v1/upload/chunk/"+resumeToken+"/complete", bytes.NewReader(completeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(srv, u.ID))

	w := serveRequest(srv, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	var res struct {
		StoredChunks  int   `json:"stored_chunks"`
		MissingChunks []int `json:"missing_chunks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if res.StoredChunks != 2 {
		t.Errorf("expected stored_chunks=2, got %d", res.StoredChunks)
	}
	if len(res.MissingChunks) != 1 || res.MissingChunks[0] != 2 {
		t.Errorf("expected missing_chunks=[2], got %v", res.MissingChunks)
	}
}

func TestServer_ChunkResume_shouldReturnStoredChunks(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkresume_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	chunkSize := 512
	totalChunks := 3
	totalSize := chunkSize * totalChunks

	chunkData := make([]byte, chunkSize)
	var resumeToken string

	for chunkIndex := 0; chunkIndex < 2; chunkIndex++ {
		reqBody := bytes.NewReader(chunkData)
		req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", reqBody)
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Authorization", authHeader(srv, u.ID))
		setChunkHeaders(req, "resume.bin", totalSize, totalChunks, chunkIndex, chunkSize, true)
		if resumeToken != "" {
			req.Header.Set("X-Resume-Token", resumeToken)
		}

		w := serveRequest(srv, req)
		var res struct {
			ResumeToken string `json:"resume_token"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		resumeToken = res.ResumeToken
	}

	req := makeTestRequest(t, "HEAD", "/api/v1/upload/chunk/"+resumeToken, http.NoBody)
	req.Header.Set("Authorization", authHeader(srv, u.ID))

	w := serveRequest(srv, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	storedCount := w.Header().Get("X-Stored-Count")
	uploadStatus := w.Header().Get("X-Upload-Status")
	storedChunks := w.Header().Get("X-Stored-Chunks")
	uploadID := w.Header().Get("X-Upload-ID")
	totalSizeHeader := w.Header().Get("X-Total-Size")

	if storedCount != "2" {
		t.Errorf("expected X-Stored-Count=2, got %s", storedCount)
	}
	if uploadStatus != string(model.JobStatusQueued) {
		t.Errorf("expected X-Upload-Status=%s, got %s", model.JobStatusQueued, uploadStatus)
	}
	if storedChunks != "[0,1]" {
		t.Errorf("expected X-Stored-Chunks=[0,1], got %s", storedChunks)
	}
	if uploadID == "" {
		t.Error("expected X-Upload-ID")
	}
	if totalSizeHeader == "" {
		t.Error("expected X-Total-Size")
	}
}

func TestServer_ChunkUpload_missingHeaders_shouldBadRequest(t *testing.T) {
	t.Parallel()
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	u, _ := us.Create("chunkbad_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)

	req := makeTestRequest(t, "POST", "/api/v1/upload/chunk", bytes.NewReader([]byte("data")))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", authHeader(srv, u.ID))

	w := serveRequest(srv, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing headers, got %d", w.Code)
	}
}

func serveRequest(srv *Server, req *http.Request) *responseRecorder {
	w := &responseRecorder{HeaderMap: make(http.Header)}
	srv.router.ServeHTTP(w, req)
	return w
}

type responseRecorder struct {
	Code      int
	HeaderMap http.Header
	Body      bytes.Buffer
}

func (r *responseRecorder) Header() http.Header {
	return r.HeaderMap
}

func (r *responseRecorder) WriteHeader(code int) {
	r.Code = code
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.Body.Write(b)
}

func makeTestRequest(t *testing.T, method, path string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	return req
}
