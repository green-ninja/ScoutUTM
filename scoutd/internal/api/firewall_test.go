package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestGetFirewall(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/firewall", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /firewall: status = %d, want 200", w.Code)
	}

	var fw config.FirewallConfig
	json.Unmarshal(w.Body.Bytes(), &fw)
	if len(fw.Forwardings) == 0 {
		t.Error("expected at least one forwarding rule")
	}
}

func TestPutFirewall(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	fw := config.FirewallConfig{
		Forwardings:  []config.ForwardingRule{{Src: "lan", Dest: "wan"}},
		Rules:        []config.FirewallRule{},
		PortForwards: []config.PortForward{},
	}
	body, _ := json.Marshal(fw)

	req := httptest.NewRequest("PUT", "/api/v1/firewall", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /firewall: status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestCRUDRules(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Create rule.
	rule := config.FirewallRule{Name: "test", Src: "lan", Dest: "wan", Proto: "tcp", Port: 80, Action: "drop"}
	body, _ := json.Marshal(rule)

	req := httptest.NewRequest("POST", "/api/v1/firewall/rules", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /firewall/rules: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Get rules — should have 1.
	req = httptest.NewRequest("GET", "/api/v1/firewall/rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var rules []config.FirewallRule
	json.Unmarshal(w.Body.Bytes(), &rules)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	// Update rule.
	rule.Port = 443
	body, _ = json.Marshal(rule)
	req = httptest.NewRequest("PUT", "/api/v1/firewall/rules/0", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /firewall/rules/0: status = %d", w.Code)
	}

	// Delete rule.
	req = httptest.NewRequest("DELETE", "/api/v1/firewall/rules/0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /firewall/rules/0: status = %d", w.Code)
	}

	// Verify empty.
	req = httptest.NewRequest("GET", "/api/v1/firewall/rules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &rules)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestDeleteRule_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("DELETE", "/api/v1/firewall/rules/99", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE nonexistent rule: status = %d, want 404", w.Code)
	}
}

func TestCRUDPortForwards(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	router := srv.Router()

	// Create.
	pf := config.PortForward{Name: "HTTP", Proto: "tcp", SrcPort: 80, DestIP: "192.168.1.100", DestPort: 8080}
	body, _ := json.Marshal(pf)

	req := httptest.NewRequest("POST", "/api/v1/firewall/port-forwards", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /firewall/port-forwards: status = %d", w.Code)
	}

	// Get.
	req = httptest.NewRequest("GET", "/api/v1/firewall/port-forwards", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var pfs []config.PortForward
	json.Unmarshal(w.Body.Bytes(), &pfs)
	if len(pfs) != 1 {
		t.Fatalf("expected 1 port forward, got %d", len(pfs))
	}

	// Delete.
	req = httptest.NewRequest("DELETE", "/api/v1/firewall/port-forwards/0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /firewall/port-forwards/0: status = %d", w.Code)
	}
}

func TestCreateRule_InvalidPort(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	rule := config.FirewallRule{Name: "bad", Src: "lan", Dest: "wan", Proto: "tcp", Port: 99999, Action: "drop"}
	body, _ := json.Marshal(rule)

	req := httptest.NewRequest("POST", "/api/v1/firewall/rules", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid port rule: status = %d, want 422", w.Code)
	}
}
