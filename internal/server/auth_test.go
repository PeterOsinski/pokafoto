package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestAuth_Register_shouldCreateUser(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"newuser123","password":"password123"}`, jsonHeaders())
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	user, _ := resp["user"].(map[string]interface{})
	if user["username"] != "newuser123" {
		t.Errorf("expected newuser123, got %v", user["username"])
	}
	if user["role"] != "member" {
		t.Errorf("expected member role, got %v", user["role"])
	}
}

func TestAuth_Register_shouldRejectDuplicateUsername(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"dupeuser11","password":"pass12345678"}`, jsonHeaders())
	w := testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"dupeuser11","password":"pass12345678"}`, jsonHeaders())
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAuth_Register_shouldRejectShortPassword(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"shortpw123","password":"123"}`, jsonHeaders())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuth_Register_shouldRejectShortUsername(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"ab","password":"password123"}`, jsonHeaders())
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuth_Login_shouldReturnTokens(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	uname := "loginuser1_" + uuid.NewString()[:8]
	us.Create(uname, "passphrase", model.RoleMember, nil)

	w := testRequest(t, srv, "POST", "/api/v1/auth/login", `{"username":"`+uname+`","password":"passphrase"}`, jsonHeaders())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Error("expected access_token")
	}
	if resp["refresh_token"] == nil || resp["refresh_token"] == "" {
		t.Error("expected refresh_token")
	}
}

func TestAuth_Login_shouldRejectWrongPassword(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	uname := "badpwuser_" + uuid.NewString()[:8]
	us.Create(uname, "correct123", model.RoleMember, nil)

	w := testRequest(t, srv, "POST", "/api/v1/auth/login", `{"username":"`+uname+`","password":"wrongpass"}`, jsonHeaders())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_Login_shouldRejectNonexistentUser(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "POST", "/api/v1/auth/login", `{"username":"nobody123","password":"whatever"}`, jsonHeaders())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_Me_shouldReturnUser(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	uname := "meuser1_" + uuid.NewString()[:8]
	u, _ := us.Create(uname, "password123", model.RoleMember, nil)

	token := generateTestToken(srv.cfg.Auth.JWTSecret, u.ID, "member")

	w := testRequest(t, srv, "GET", "/api/v1/auth/me", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Me_shouldRejectUnauthenticated(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/auth/me", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_Refresh_shouldReturnNewTokens(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	uname := "refresh_"+uuid.NewString()[:8]
	us.Create(uname, "passphrase", model.RoleMember, nil)

	loginResp := testRequest(t, srv, "POST", "/api/v1/auth/login", `{"username":"`+uname+`","password":"passphrase"}`, jsonHeaders())
	var loginData map[string]interface{}
	json.Unmarshal(loginResp.Body.Bytes(), &loginData)
	refreshToken := loginData["refresh_token"].(string)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	w := testRequest(t, srv, "POST", "/api/v1/auth/refresh", string(body), jsonHeaders())
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == nil {
		t.Error("expected access_token in refresh response")
	}
}

func TestAuth_Refresh_shouldRejectInvalidToken(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"refresh_token": "invalid-token"})
	w := testRequest(t, srv, "POST", "/api/v1/auth/refresh", string(body), jsonHeaders())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_Logout_shouldReturn204(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	uname := "logoutuser1_"+uuid.NewString()[:8]
	us.Create(uname, "passphrase", model.RoleMember, nil)

	loginResp := testRequest(t, srv, "POST", "/api/v1/auth/login", `{"username":"`+uname+`","password":"passphrase"}`, jsonHeaders())
	var loginData map[string]interface{}
	json.Unmarshal(loginResp.Body.Bytes(), &loginData)
	refreshToken := loginData["refresh_token"].(string)

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	w := testRequest(t, srv, "POST", "/api/v1/auth/logout", string(body), jsonHeaders())
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	body, _ = json.Marshal(map[string]string{"refresh_token": refreshToken})
	w2 := testRequest(t, srv, "POST", "/api/v1/auth/refresh", string(body), jsonHeaders())
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d", w2.Code)
	}
}
