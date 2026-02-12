package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestGetDNS(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/dns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /dns: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp config.DNSConfig
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Upstream) == 0 {
		t.Error("expected upstream servers in response")
	}
	if resp.Upstream[0] != "1.1.1.1" {
		t.Errorf("upstream[0] = %q, want 1.1.1.1", resp.Upstream[0])
	}
}

func TestPutDNS(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:      []string{"8.8.8.8", "8.8.4.4"},
		UseDOT:        true,
		BlockMalware:  true,
		BlockPhishing: false,
		BlockAds:      true,
		QueryLogging:  false,
		CustomBlock:   []string{},
		CustomAllow:   []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /dns: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Verify changes persisted.
	cfg := srv.Config.Current()
	if cfg.DNS.Upstream[0] != "8.8.8.8" {
		t.Errorf("upstream[0] = %q, want 8.8.8.8", cfg.DNS.Upstream[0])
	}
	if !cfg.DNS.UseDOT {
		t.Error("expected use_dot to be true")
	}
	if !cfg.DNS.BlockAds {
		t.Error("expected block_ads to be true")
	}
	if cfg.DNS.QueryLogging {
		t.Error("expected query_logging to be false")
	}
}

func TestPutDNS_InvalidUpstream(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:    []string{"not-an-ip"},
		CustomBlock: []string{},
		CustomAllow: []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT /dns invalid upstream: status = %d, want 422", w.Code)
	}
}

func TestPutDNS_EmptyUpstream(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:    []string{},
		CustomBlock: []string{},
		CustomAllow: []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT /dns empty upstream: status = %d, want 422", w.Code)
	}
}

func TestPutDNS_CustomBlockDomains(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:      []string{"1.1.1.1"},
		BlockMalware:  true,
		BlockPhishing: true,
		QueryLogging:  true,
		CustomBlock:   []string{"badsite.com", "malware.org"},
		CustomAllow:   []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /dns custom block: status = %d, body = %s", w.Code, w.Body.String())
	}

	cfg := srv.Config.Current()
	if len(cfg.DNS.CustomBlock) != 2 {
		t.Errorf("custom_block length = %d, want 2", len(cfg.DNS.CustomBlock))
	}
	if cfg.DNS.CustomBlock[0] != "badsite.com" {
		t.Errorf("custom_block[0] = %q, want badsite.com", cfg.DNS.CustomBlock[0])
	}
}

func TestGetDNSQueryLog_NotImplemented(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/dns/query-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("GET /dns/query-log: status = %d, want 501", w.Code)
	}
}

func TestPutDNS_DOTAndDOHMutuallyExclusive(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:    []string{"1.1.1.1"},
		UseDOT:      true,
		UseDOH:      true,
		CustomBlock: []string{},
		CustomAllow: []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT /dns DoT+DoH: status = %d, want 422", w.Code)
	}
}

func TestPutDNS_InvalidCustomBlockDomain(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := config.DNSConfig{
		Upstream:    []string{"1.1.1.1"},
		CustomBlock: []string{"not a domain!"},
		CustomAllow: []string{},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PUT /dns invalid domain: status = %d, want 422", w.Code)
	}
}
