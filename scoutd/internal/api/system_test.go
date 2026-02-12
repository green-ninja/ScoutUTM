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
