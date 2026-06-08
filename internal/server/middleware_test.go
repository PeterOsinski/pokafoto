package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMiddleware_Auth_shouldRejectMissingToken(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/files", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_Auth_shouldRejectMalformedHeader(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/files", "", map[string]string{"Authorization": "NotBearer token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_Auth_shouldRejectInvalidToken(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/files", "", map[string]string{"Authorization": "Bearer invalid-token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_Auth_shouldAcceptValidToken(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "GET", "/api/v1/stats", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMiddleware_Admin_shouldRejectNonAdmin(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "member")

	w := testRequest(t, srv, "GET", "/api/v1/admin/users", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestMiddleware_Admin_shouldAllowAdmin(t *testing.T) {
	t.Parallel()
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	token := generateTestToken(srv.cfg.Auth.JWTSecret, "user-id", "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/users", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func generateTestToken(secret, userID, role string) string {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().UTC().Add(1 * time.Hour).Unix(),
		"iat":     time.Now().UTC().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}
