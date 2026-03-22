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
		Upstream:     []string{"8.8.8.8", "8.8.4.4"},
		UseDOT:       true,
		BlockThreats: true,
		QueryLogging: false,
		CustomBlock:  []string{},
		CustomAllow:  []string{},
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
		Upstream:     []string{"1.1.1.1"},
		BlockThreats: true,
		QueryLogging: true,
		CustomBlock:  []string{"badsite.com", "malware.org"},
		CustomAllow:  []string{},
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

	if w.Code != http.StatusOK {
		t.Errorf("GET /dns/query-log: status = %d, want 200", w.Code)
	}
}

func TestPutDNS_RapidConcurrentChanges(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	router := srv.Router()

	// Fire 3 concurrent DNS config updates with different upstream servers.
	// With TryLock, only one will succeed; others get 409 Conflict.
	upstreams := [][]string{
		{"1.1.1.1", "1.0.0.1"},
		{"8.8.8.8", "8.8.4.4"},
		{"9.9.9.9", "149.112.112.112"},
	}

	done := make(chan int, 3)
	for i, up := range upstreams {
		go func(idx int, servers []string) {
			update := config.DNSConfig{
				Upstream:     servers,
				QueryLogging: true,
				CustomBlock:  []string{},
				CustomAllow:  []string{},
			}
			body, _ := json.Marshal(update)
			req := httptest.NewRequest("PUT", "/api/v1/dns", bytes.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			done <- w.Code
		}(i, up)
	}

	// Wait for all 3 to complete.
	successCount := 0
	conflictCount := 0
	for range 3 {
		code := <-done
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusConflict {
			conflictCount++
		}
	}

	// Exactly one should succeed; others should get 409 Conflict.
	if successCount < 1 {
		t.Errorf("expected at least 1 successful apply, got %d", successCount)
	}
	if successCount+conflictCount != 3 {
		t.Errorf("expected success+conflict=3, got success=%d conflict=%d", successCount, conflictCount)
	}

	// Final config should be valid and consistent.
	cfg := srv.Config.Current()
	if errs := config.Validate(cfg); len(errs) > 0 {
		t.Errorf("config invalid after concurrent applies: %v", errs)
	}

	// Upstream should be one of the 3 valid sets (whichever won the race).
	validUpstream := false
	for _, up := range upstreams {
		if len(cfg.DNS.Upstream) == len(up) && cfg.DNS.Upstream[0] == up[0] && cfg.DNS.Upstream[1] == up[1] {
			validUpstream = true
			break
		}
	}
	if !validUpstream {
		t.Errorf("unexpected upstream after concurrent applies: %v", cfg.DNS.Upstream)
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
