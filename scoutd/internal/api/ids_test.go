package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestGetIDS_Defaults(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/ids", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ids: status = %d, want 200", w.Code)
	}

	var ids config.IDSConfig
	json.Unmarshal(w.Body.Bytes(), &ids)

	if !ids.Enabled {
		t.Error("IDS should be enabled by default")
	}
	if ids.Mode != "ids" {
		t.Errorf("IDS mode = %q, want ids", ids.Mode)
	}
}

func TestPutIDS_ChangeMode(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{"mode": "ips"})
	req := httptest.NewRequest("PUT", "/api/v1/ids", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /ids mode=ips: status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var ids config.IDSConfig
	json.Unmarshal(w.Body.Bytes(), &ids)
	if ids.Mode != "ips" {
		t.Errorf("IDS mode after update = %q, want ips", ids.Mode)
	}
}

func TestPutIDS_InvalidMode(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{"mode": "invalid"})
	req := httptest.NewRequest("PUT", "/api/v1/ids", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT /ids invalid mode: status = %d, want 422", w.Code)
	}
}

func TestPutIDS_Disable(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	enabled := false
	body, _ := json.Marshal(map[string]interface{}{"enabled": enabled})
	req := httptest.NewRequest("PUT", "/api/v1/ids", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /ids disable: status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var ids config.IDSConfig
	json.Unmarshal(w.Body.Bytes(), &ids)
	if ids.Enabled {
		t.Error("IDS should be disabled after update")
	}
}

func TestGetAlerts_Empty(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/ids/alerts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ids/alerts: status = %d, want 200", w.Code)
	}

	var alerts []interface{}
	json.Unmarshal(w.Body.Bytes(), &alerts)
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
}
