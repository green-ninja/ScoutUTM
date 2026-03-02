package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevices_List(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list devices: status = %d, want 200", w.Code)
	}

	var devices []deviceResponse
	json.Unmarshal(w.Body.Bytes(), &devices)

	// Dev-mode scanner returns synthetic devices.
	if len(devices) < 5 {
		t.Errorf("expected at least 5 devices, got %d", len(devices))
	}
}

func TestDevices_GetByMAC(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/devices/aa:bb:cc:dd:ee:01", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get device: status = %d, want 200", w.Code)
	}

	var device deviceResponse
	json.Unmarshal(w.Body.Bytes(), &device)

	if device.MAC != "aa:bb:cc:dd:ee:01" {
		t.Errorf("device MAC = %s, want aa:bb:cc:dd:ee:01", device.MAC)
	}
}

func TestDevices_GetByMAC_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/devices/ff:ff:ff:ff:ff:ff", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("get unknown device: status = %d, want 404", w.Code)
	}
}

func TestDevices_UpdateName(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{"name": "My Laptop"})
	req := httptest.NewRequest("PUT", "/api/v1/devices/aa:bb:cc:dd:ee:01", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update device: status = %d, want 200", w.Code)
	}

	var device deviceResponse
	json.Unmarshal(w.Body.Bytes(), &device)

	if device.Name != "My Laptop" {
		t.Errorf("device name = %q, want 'My Laptop'", device.Name)
	}
}
