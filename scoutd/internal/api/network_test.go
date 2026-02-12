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

func TestDeleteZone_ReferencedByForwarding(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Create a "guest" zone.
	zone := config.ZoneConfig{Name: "guest", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"}
	body, _ := json.Marshal(zone)
	req := httptest.NewRequest("POST", "/api/v1/network/zones", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create zone: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Add a forwarding rule referencing "guest".
	cfg, _ := srv.Config.Current().DeepCopy()
	cfg.Firewall.Forwardings = append(cfg.Firewall.Forwardings, config.ForwardingRule{Src: "guest", Dest: "wan"})
	// Add an interface so the forwarding validates.
	cfg.Network.Interfaces = append(cfg.Network.Interfaces, config.InterfaceConfig{Name: "eth2", Zone: "guest", Type: "ethernet", IPv4Mode: "static", IPv4Address: "10.0.0.1/24"})
	srv.applyConfig(cfg)

	// Try to delete the zone — should fail with 409 Conflict.
	req = httptest.NewRequest("DELETE", "/api/v1/network/zones/guest", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("DELETE zone with dependents: status = %d, want 409, body = %s", w.Code, w.Body.String())
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

func TestCRUDVLANs(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Add a zone for the VLAN.
	zone := config.ZoneConfig{Name: "guest", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"}
	body, _ := json.Marshal(zone)
	req := httptest.NewRequest("POST", "/api/v1/network/zones", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create zone: status = %d", w.Code)
	}

	// Get VLANs — should be empty.
	req = httptest.NewRequest("GET", "/api/v1/network/vlans", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /network/vlans: status = %d", w.Code)
	}
	var vlans []config.VLANConfig
	json.Unmarshal(w.Body.Bytes(), &vlans)
	if len(vlans) != 0 {
		t.Errorf("expected 0 VLANs, got %d", len(vlans))
	}

	// Create VLAN.
	vlan := config.VLANConfig{ID: 100, Parent: "eth1", Zone: "guest", Subnet: "10.0.100.1/24"}
	body, _ = json.Marshal(vlan)
	req = httptest.NewRequest("POST", "/api/v1/network/vlans", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /network/vlans: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Get VLANs — should have one.
	req = httptest.NewRequest("GET", "/api/v1/network/vlans", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &vlans)
	if len(vlans) != 1 {
		t.Errorf("expected 1 VLAN, got %d", len(vlans))
	}

	// Delete VLAN.
	req = httptest.NewRequest("DELETE", "/api/v1/network/vlans/100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /network/vlans/100: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Get VLANs — should be empty again.
	req = httptest.NewRequest("GET", "/api/v1/network/vlans", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &vlans)
	if len(vlans) != 0 {
		t.Errorf("expected 0 VLANs after delete, got %d", len(vlans))
	}
}

func TestCreateVLAN_DuplicateID(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Create zone first.
	zone := config.ZoneConfig{Name: "guest", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"}
	body, _ := json.Marshal(zone)
	req := httptest.NewRequest("POST", "/api/v1/network/zones", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Create first VLAN.
	vlan := config.VLANConfig{ID: 100, Parent: "eth1", Zone: "guest", Subnet: "10.0.100.1/24"}
	body, _ = json.Marshal(vlan)
	req = httptest.NewRequest("POST", "/api/v1/network/vlans", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first VLAN: status = %d", w.Code)
	}

	// Try to create duplicate.
	body, _ = json.Marshal(vlan)
	req = httptest.NewRequest("POST", "/api/v1/network/vlans", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate VLAN: status = %d, want 409", w.Code)
	}
}

func TestCreateVLAN_InvalidID(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	vlan := config.VLANConfig{ID: 5000, Parent: "eth1", Zone: "lan", Subnet: "10.0.100.1/24"}
	body, _ := json.Marshal(vlan)
	req := httptest.NewRequest("POST", "/api/v1/network/vlans", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid VLAN ID: status = %d, want 422", w.Code)
	}
}

func TestDeleteVLAN_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("DELETE", "/api/v1/network/vlans/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE nonexistent VLAN: status = %d, want 404", w.Code)
	}
}

func TestDeleteVLAN_ReferencedByForwarding(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Create guest zone.
	zone := config.ZoneConfig{Name: "guest", InputPolicy: "drop", ForwardPolicy: "drop", OutputPolicy: "accept"}
	body, _ := json.Marshal(zone)
	req := httptest.NewRequest("POST", "/api/v1/network/zones", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Create VLAN in guest zone.
	vlan := config.VLANConfig{ID: 100, Parent: "eth1", Zone: "guest", Subnet: "10.0.100.1/24"}
	body, _ = json.Marshal(vlan)
	req = httptest.NewRequest("POST", "/api/v1/network/vlans", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create VLAN: status = %d", w.Code)
	}

	// Add a forwarding rule referencing "guest".
	cfg, _ := srv.Config.Current().DeepCopy()
	cfg.Firewall.Forwardings = append(cfg.Firewall.Forwardings, config.ForwardingRule{Src: "guest", Dest: "wan"})
	srv.applyConfig(cfg)

	// Try to delete the VLAN — should fail with 409.
	req = httptest.NewRequest("DELETE", "/api/v1/network/vlans/100", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("DELETE VLAN with forwarding ref: status = %d, want 409, body = %s", w.Code, w.Body.String())
	}
}
