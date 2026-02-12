package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestGetNetwork(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/network", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /network: status = %d", w.Code)
	}

	var net config.NetworkConfig
	json.Unmarshal(w.Body.Bytes(), &net)
	if len(net.Interfaces) == 0 {
		t.Error("expected at least one interface")
	}
	if len(net.Zones) == 0 {
		t.Error("expected at least one zone")
	}
}

func TestPutNetwork(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	net := config.DefaultConfig().Network
	body, _ := json.Marshal(net)

	req := httptest.NewRequest("PUT", "/api/v1/network", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /network: status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestCRUDZones(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Get zones.
	req := httptest.NewRequest("GET", "/api/v1/network/zones", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var zones []config.ZoneConfig
	json.Unmarshal(w.Body.Bytes(), &zones)
	initialCount := len(zones)

	// Create zone.
	zone := config.ZoneConfig{Name: "guest", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"}
	body, _ := json.Marshal(zone)

	req = httptest.NewRequest("POST", "/api/v1/network/zones", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /network/zones: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Get zones — should have one more.
	req = httptest.NewRequest("GET", "/api/v1/network/zones", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &zones)
	if len(zones) != initialCount+1 {
		t.Errorf("expected %d zones, got %d", initialCount+1, len(zones))
	}

	// Delete zone.
	req = httptest.NewRequest("DELETE", "/api/v1/network/zones/guest", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /network/zones/guest: status = %d", w.Code)
	}
}

func TestDeleteZone_ProtectBuiltin(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	for _, name := range []string{"wan", "lan"} {
		req := httptest.NewRequest("DELETE", "/api/v1/network/zones/"+name, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("DELETE builtin zone %s: status = %d, want 400", name, w.Code)
		}
	}
}

func TestDeleteZone_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("DELETE", "/api/v1/network/zones/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE nonexistent zone: status = %d, want 404", w.Code)
	}
}
