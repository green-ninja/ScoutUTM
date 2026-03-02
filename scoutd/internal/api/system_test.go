package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSystem(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/system", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /system: status = %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should have hostname.
	if _, ok := resp["hostname"]; !ok {
		t.Error("response missing hostname")
	}

	// Should NOT have admin_hash.
	if _, ok := resp["admin_hash"]; ok {
		t.Error("response should not include admin_hash")
	}
}

func TestGetSystem_NoAdminHashLeak(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/system", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	body := w.Body.String()
	if bytes.Contains([]byte(body), []byte("admin_hash")) {
		t.Error("admin_hash leaked in system response")
	}
}

func TestPatchSystem_PartialUpdate(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	// Update only hostname.
	body, _ := json.Marshal(map[string]interface{}{
		"hostname": "myrouter",
	})

	req := httptest.NewRequest("PATCH", "/api/v1/system", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH /system: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Verify hostname changed.
	cfg := srv.Config.Current()
	if cfg.System.Hostname != "myrouter" {
		t.Errorf("hostname = %q, want myrouter", cfg.System.Hostname)
	}

	// Verify timezone unchanged (partial update).
	if cfg.System.Timezone != "UTC" {
		t.Errorf("timezone changed to %q, should remain UTC", cfg.System.Timezone)
	}
}

func TestPatchSystem_NTPServers(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	body, _ := json.Marshal(map[string]interface{}{
		"ntp_servers": []string{"time.google.com", "time.cloudflare.com"},
	})

	req := httptest.NewRequest("PATCH", "/api/v1/system", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH /system: status = %d", w.Code)
	}

	cfg := srv.Config.Current()
	if len(cfg.System.NTPServers) != 2 {
		t.Errorf("NTPServers = %v, want 2 servers", cfg.System.NTPServers)
	}
}

func TestPatchSystem_InvalidHostname(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	body, _ := json.Marshal(map[string]interface{}{
		"hostname": "",
	})

	req := httptest.NewRequest("PATCH", "/api/v1/system", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PATCH /system with empty hostname: status = %d, want 422", w.Code)
	}
}

func TestSystem_About(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/system/about", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("about: status = %d, want 200", w.Code)
	}

	var resp aboutResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Version != "0.1.0-test" {
		t.Errorf("version = %q, want 0.1.0-test", resp.Version)
	}
	if resp.GoVersion == "" {
		t.Error("go_version is empty")
	}
}

func TestSystem_Backup(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/system/backup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("backup: status = %d, want 200", w.Code)
	}

	if ct := w.Header().Get("Content-Disposition"); ct == "" {
		t.Error("missing Content-Disposition header")
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &cfg); err != nil {
		t.Errorf("backup is not valid JSON: %v", err)
	}
}

func TestSystem_Restore(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	cfg := srv.Config.Current()
	cfg.System.Hostname = "restored-utm"
	data, _ := json.Marshal(cfg)

	req := httptest.NewRequest("POST", "/api/v1/system/restore", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("restore: status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}

	newCfg := srv.Config.Current()
	if newCfg.System.Hostname != "restored-utm" {
		t.Errorf("hostname = %q, want restored-utm", newCfg.System.Hostname)
	}
}

func TestSystem_Reboot_DevMode(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("POST", "/api/v1/system/reboot", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("reboot dev-mode: status = %d, want 200", w.Code)
	}
}

func TestSystem_ChangePassword(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{
		"current":      "testpassword",
		"new_password": "newpassword123",
	})
	req := httptest.NewRequest("POST", "/api/v1/system/password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("change password: status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

func TestSystem_ChangePassword_WrongCurrent(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{
		"current":      "wrongpassword",
		"new_password": "newpassword123",
	})
	req := httptest.NewRequest("POST", "/api/v1/system/password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong current password: status = %d, want 401", w.Code)
	}
}

func TestSystem_RegenerateCert_DevMode(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("POST", "/api/v1/system/regenerate-cert", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("regenerate cert: status = %d, want 200", w.Code)
	}
}
