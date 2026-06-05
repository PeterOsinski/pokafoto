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

func TestAdmin_UpdateQuota_shouldSetQuota(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "qadmin_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	member, _ := us.Create("quotauser_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	tenGB := int64(10737418240)
	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+member.ID+"/quota", `{"space_quota":10737418240}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	q := resp["space_quota"]
	if qn, ok := q.(float64); !ok || int64(qn) != tenGB {
		t.Errorf("expected space_quota=10737418240, got %v", q)
	}

	updated, _ := us.FindByID(member.ID)
	if updated.SpaceQuota == nil || *updated.SpaceQuota != tenGB {
		t.Errorf("expected persisted quota 10737418240, got %v", updated.SpaceQuota)
	}
}

func TestAdmin_UpdateQuota_shouldAllowUnlimited(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	adminName := "qadmin2_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	member, _ := us.Create("qunlim_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+member.ID+"/quota", `{"space_quota":10737418240}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 setting quota, got %d", w.Code)
	}

	w = testRequest(t, srv, "PUT", "/api/v1/admin/users/"+member.ID+"/quota", `{"space_quota":null}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 setting null, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["space_quota"] != nil {
		t.Errorf("expected null, got %v", resp["space_quota"])
	}

	updated, _ := us.FindByID(member.ID)
	if updated.SpaceQuota != nil {
		t.Errorf("expected nil quota, got %v", *updated.SpaceQuota)
	}
}

func TestAdmin_UpdateQuota_shouldRejectBelowUsage(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	fs := store.NewFileStore(db)
	adminName := "qadmin3_" + uuid.NewString()[:8]
	admin, _ := us.Create(adminName, "adminpass123", model.RoleAdmin, nil)
	member, _ := us.Create("qbelow_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	f := &model.File{
		UserID:       member.ID,
		Filename:     "2024/07/bigfile.jpg",
		OriginalName: "bigfile.jpg",
		Path:         "2024/07",
		SizeBytes:    5 * 1024 * 1024,
		MimeType:     "image/jpeg",
		SHA256:       "qtest_" + uuid.NewString(),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f)

	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+member.ID+"/quota", `{"space_quota":1048576}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdmin_UpdateQuota_shouldRejectNonAdmin(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	member, _ := us.Create("qmember_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, member.ID, "member")

	w := testRequest(t, srv, "PUT", "/api/v1/admin/users/"+member.ID+"/quota", `{"space_quota":10737418240}`, map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAdmin_ListUsers_shouldIncludeQuotaFields(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("qlist_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	quota := int64(10737418240)
	us.UpdateSpaceQuota(admin.ID, &quota)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/users", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	users := resp["users"].([]interface{})
	if len(users) == 0 {
		t.Fatal("expected at least 1 user")
	}

	first := users[0].(map[string]interface{})
	if _, ok := first["space_quota"]; !ok {
		t.Error("expected space_quota field in list response")
	}
	if _, ok := first["thumbnail_size_bytes"]; !ok {
		t.Error("expected thumbnail_size_bytes field in list response")
	}
	if _, ok := first["file_count"]; !ok {
		t.Error("expected file_count field in list response")
	}
	if _, ok := first["total_size_bytes"]; !ok {
		t.Error("expected total_size_bytes field in list response")
	}
}

func TestAdmin_Stats_shouldIncludePerUserThumbnails(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("qstats_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/stats", "", map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatal("expected users array in stats response")
	}
	if len(users) == 0 {
		t.Fatal("expected at least 1 user")
	}

	u := users[0].(map[string]interface{})
	if _, ok := u["thumbnail_size_bytes"]; !ok {
		t.Error("expected thumbnail_size_bytes in per-user stats")
	}
	if _, ok := u["space_quota"]; !ok {
		t.Error("expected space_quota in per-user stats")
	}
}

func TestAdmin_S3DeletionQueue_shouldReturnPendingCount(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("s3qadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/s3-deletion-queue", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["pending"]; !ok {
		t.Error("expected pending field in response")
	}
}

func TestAdmin_Workers_shouldReturnStats(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("wrkadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/workers", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["active_workers"]; !ok {
		t.Error("expected active_workers in response")
	}
	if _, ok := resp["total_workers"]; !ok {
		t.Error("expected total_workers in response")
	}
}

func TestAdmin_BackupStatus_shouldReturnResult(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("bkupadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/backup/status", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["enabled"]; !ok {
		t.Error("expected enabled field in backup status")
	}
}

func TestAdmin_Events_shouldReturnEvents(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("evtadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/events", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["events"]; !ok {
		t.Error("expected events field in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Error("expected total field in response")
	}
}

func TestAdmin_EventCounts_shouldReturnCounts(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("evtcadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/events/counts", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["server_start"]; ok {
	}
}

func TestAdmin_Jobs_shouldReturnJobs(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("jobadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/jobs", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["jobs"]; !ok {
		t.Error("expected jobs field in response")
	}
}

func TestAdmin_Reconcile_shouldReturnCreated(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("recadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "POST", "/api/v1/admin/jobs/reconcile", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["created"]; !ok {
		t.Error("expected created field in response")
	}
}

func TestAdmin_ThumbnailStats_shouldReturnBreakdown(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("tadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/thumbnails/stats", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdmin_FileBreakdown_shouldReturnBreakdown(t *testing.T) {
	srv, db, cleanup := newTestServer(t)
	defer cleanup()

	us := store.NewUserStore(db)
	admin, _ := us.Create("fbadmin_"+uuid.NewString()[:8], "adminpass123", model.RoleAdmin, nil)
	token := generateTestToken(srv.cfg.Auth.JWTSecret, admin.ID, "admin")

	w := testRequest(t, srv, "GET", "/api/v1/admin/files/breakdown", "", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
