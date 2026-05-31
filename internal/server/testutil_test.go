package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/store"
)

func newTestServer(t *testing.T) (*Server, *store.DB, func()) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Auth.JWTSecret = "test-secret-key-12345"
	cfg.Auth.AllowRegistration = true

	db := store.OpenTestDB(t)

	settingStore := store.NewSettingStore(db)
	settingStore.Set("allow_registration", "true")

	s := New(cfg, db)

	return s, db, func() {
		s.Shutdown()
	}
}

func testRequest(t *testing.T, srv *Server, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	return w
}

func jsonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}
