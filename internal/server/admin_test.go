package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/google/uuid"
)

func TestAdmin_CreateUser_shouldReturnCreated(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "admincr_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "POST", "/api/v1/admin/users", `{"username":"newmember1","password":"password123","role":"member"}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["username"] != "newmember1" {
		t.Errorf("expected newmember1, got %v", resp["username"])
	}
	if resp["role"] != "member" {
		t.Errorf("expected member, got %v", resp["role"])
	}
}

func TestAdmin_CreateUser_shouldCreateAdmin(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "superadm_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "POST", "/api/v1/admin/users", `{"username":"newadmin2","password":"adminpass12","role":"admin"}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["role"] != "admin" {
		t.Errorf("expected admin, got %v", resp["role"])
	}
}

func TestAdmin_CreateUser_shouldRejectNonAdmin(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	memberName := "memcreate_" + uuid.NewString()[:8]
	member, _ := us.Create(memberName, "memberpass12", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, member.ID, "member")

	w := testRequest(t, srv, "POST", "/api/v1/admin/users", `{"username":"hackuser1","password":"password123","role":"admin"}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAdmin_CreateUser_shouldRejectDuplicateUsername(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "dupadmin_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	us.Create("duplicate", "password123", model.RoleMember, nil)

	w := testRequest(t, srv, "POST", "/api/v1/admin/users", `{"username":"duplicate","password":"password123","role":"member"}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAdmin_CreateUser_shouldRejectShortPassword(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "shortpwadmin_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "POST", "/api/v1/admin/users", `{"username":"shortuser","password":"123","role":"member"}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAdmin_GetRegistration_shouldReturnStatus(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "regadmin_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/registration", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["allow_registration"] != true {
		t.Errorf("expected true, got %v", resp["allow_registration"])
	}
}

func TestAdmin_ToggleRegistration_shouldSetAndGet(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "togadmin_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "PUT", "/api/v1/admin/registration", `{"enabled":false}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var putResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &putResp)
	if putResp["allow_registration"] != false {
		t.Errorf("expected false, got %v", putResp["allow_registration"])
	}

	w2 := testRequest(t, srv, "GET", "/api/v1/admin/registration", "", map[string]string{"Authorization": "Bearer " + token})
	var getResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &getResp)
	if getResp["allow_registration"] != false {
		t.Errorf("expected false after toggle, got %v", getResp["allow_registration"])
	}

	w3 := testRequest(t, srv, "PUT", "/api/v1/admin/registration", `{"enabled":true}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}

	w4 := testRequest(t, srv, "GET", "/api/v1/admin/registration", "", map[string]string{"Authorization": "Bearer " + token})
	var finalResp map[string]interface{}
	json.Unmarshal(w4.Body.Bytes(), &finalResp)
	if finalResp["allow_registration"] != true {
		t.Errorf("expected true after re-toggle, got %v", finalResp["allow_registration"])
	}
}

func TestAuth_Register_shouldBeBlockedWhenDisabled(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "blockreg_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	testRequest(t, srv, "PUT", "/api/v1/admin/registration", `{"enabled":false}`, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	})

	w := testRequest(t, srv, "POST", "/api/v1/auth/register", `{"username":"shouldfail","password":"password123"}`, jsonHeaders())
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when registration disabled, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Config_shouldReturnRegistrationStatus(t *testing.T) {
	srv, _, cleanup := newTestServer(t)
	defer cleanup()

	w := testRequest(t, srv, "GET", "/api/v1/auth/config", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["allow_registration"] != true {
		t.Errorf("expected true, got %v", resp["allow_registration"])
	}
}
