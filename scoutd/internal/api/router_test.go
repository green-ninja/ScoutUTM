package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/ids"
	"github.com/scoututm/scoutd/internal/netmon"
	"github.com/scoututm/scoutd/internal/render"
	"github.com/scoututm/scoutd/internal/service"
	"github.com/scoututm/scoutd/internal/ws"
)

func testServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dir := t.TempDir()
	store := config.NewStore(filepath.Join(dir, "config.json"))
	if err := store.Load(true); err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	sessions := auth.NewSessionManager()
	hub := ws.NewHub()
	tmplDir := copyTemplates(t)
	outputDir := filepath.Join(dir, "output")
	engine := render.NewEngine(tmplDir, outputDir, true)
	svcMgr := service.NewManager(true)
	eveReader := ids.NewReader("", "")
	scanner := testScanner(t, dir)
	scanner.Scan()
	srv := NewServer(store, sessions, hub, engine, svcMgr, eveReader, scanner, true, "0.1.0-test")
	return srv, func() {}
}

// testScanner creates a scanner with temp neigh and lease files for testing.
func testScanner(t *testing.T, dir string) *netmon.Scanner {
	t.Helper()
	neighFile := filepath.Join(dir, "neigh")
	neighData := "192.168.1.100 dev eth1 lladdr aa:bb:cc:dd:ee:01 REACHABLE\n" +
		"192.168.1.101 dev eth1 lladdr aa:bb:cc:dd:ee:02 REACHABLE\n" +
		"192.168.1.102 dev eth1 lladdr aa:bb:cc:dd:ee:03 REACHABLE\n" +
		"192.168.1.103 dev eth1 lladdr aa:bb:cc:dd:ee:04 REACHABLE\n" +
		"192.168.1.104 dev eth1 lladdr aa:bb:cc:dd:ee:05 REACHABLE\n" +
		"192.168.1.105 dev eth1 lladdr aa:bb:cc:dd:ee:06 REACHABLE\n"
	os.WriteFile(neighFile, []byte(neighData), 0644)

	leaseFile := filepath.Join(dir, "leases")
	leaseData := "1999999999 aa:bb:cc:dd:ee:01 192.168.1.100 laptop 01:aa:bb:cc:dd:ee:01\n" +
		"1999999999 aa:bb:cc:dd:ee:02 192.168.1.101 phone 01:aa:bb:cc:dd:ee:02\n" +
		"1999999999 aa:bb:cc:dd:ee:03 192.168.1.102 tablet 01:aa:bb:cc:dd:ee:03\n" +
		"1999999999 aa:bb:cc:dd:ee:04 192.168.1.103 smart-tv 01:aa:bb:cc:dd:ee:04\n" +
		"1999999999 aa:bb:cc:dd:ee:05 192.168.1.104 gaming-pc 01:aa:bb:cc:dd:ee:05\n" +
		"1999999999 aa:bb:cc:dd:ee:06 192.168.1.105 iot-hub 01:aa:bb:cc:dd:ee:06\n"
	os.WriteFile(leaseFile, []byte(leaseData), 0644)

	s := netmon.NewScanner(leaseFile)
	s.SetNeighFile(neighFile)
	return s
}

// copyTemplates copies template files to a temp dir for tests.
func copyTemplates(t *testing.T) string {
	t.Helper()
	// Find the templates directory relative to the test file.
	// In tests, we're in scoutd/internal/api/ — templates are at scoutd/templates/.
	tmplDir := filepath.Join("..", "..", "templates")
	return tmplDir
}

func setAdminPassword(t *testing.T, srv *Server, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	cfg := srv.Config.Current()
	cfg.System.AdminHash = hash
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()
}

func loginAndGetToken(t *testing.T, srv *Server, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": password})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login returned %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["token"]
}

func TestHealth_Returns200(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["healthy"] != true {
		t.Errorf("healthy = %v, want true", resp["healthy"])
	}
	if _, ok := resp["version"]; !ok {
		t.Error("response missing version field")
	}
}

func TestHealth_NoAuthRequired(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health without auth: status = %d, want 200", w.Code)
	}
}

func TestLogin_CorrectPassword(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{"password": "testpassword"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == "" {
		t.Error("login response missing token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]string{"password": "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("login with wrong password: status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_BlocksWithoutToken(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/firewall", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated request: status = %d, want 401", w.Code)
	}
}

func TestDevicesEndpoint_Returns200(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("devices endpoint: status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_AcceptsCookie(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	req := httptest.NewRequest("GET", "/api/v1/firewall", nil)
	req.AddCookie(&http.Cookie{Name: "scout_session", Value: token})
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Errorf("cookie auth: status = %d, want non-401", w.Code)
	}
}

func TestWizard_AccessibleWithoutAuth(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Wizard step 1 should be accessible without authentication when setup is not done.
	req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Errorf("wizard step 1 without auth: status = %d, should not be 401", w.Code)
	}
}

func TestWizard_BlockedAfterSetupNoAuth(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Mark setup as done.
	cfg := srv.Config.Current()
	cfg.System.SetupDone = true
	srv.Config.SetCurrent(cfg)

	// Without any auth token, wizard should still return 403 (not 401).
	req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("wizard after setup (no auth): status = %d, want 403", w.Code)
	}
}

func TestBackup_RedactsSecrets(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	// Set a WireGuard private key so we can verify it's redacted.
	cfg := srv.Config.Current()
	cfg.VPN.WireGuard.PrivateKey = "secret-private-key"
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	req := httptest.NewRequest("GET", "/api/v1/system/backup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("backup: status = %d, want 200", w.Code)
	}

	body := w.Body.String()
	if strings.Contains(body, "secret-private-key") {
		t.Error("backup response contains WireGuard private key")
	}
	// AdminHash should be redacted (empty string in JSON).
	var backup map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &backup)
	sys := backup["system"].(map[string]interface{})
	if hash, ok := sys["admin_hash"]; ok && hash != "" {
		t.Error("backup response contains non-empty admin_hash")
	}
}

func TestPasswordChange_RevokesOtherSessions(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")

	token1 := loginAndGetToken(t, srv, "testpassword")
	token2 := loginAndGetToken(t, srv, "testpassword")

	// Change password using token1.
	body, _ := json.Marshal(map[string]string{
		"current":      "testpassword",
		"new_password": "newpassword123",
	})
	req := httptest.NewRequest("POST", "/api/v1/system/password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("password change: status = %d, want 200", w.Code)
	}

	// token1 (used for password change) should still be valid.
	req = httptest.NewRequest("GET", "/api/v1/system", nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Error("token used for password change should still be valid")
	}

	// token2 (other session) should be revoked.
	req = httptest.NewRequest("GET", "/api/v1/system", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("other session after password change: status = %d, want 401", w.Code)
	}
}

func TestSecurityHeaders_Present(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff header")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options: DENY header")
	}
	if w.Header().Get("Strict-Transport-Security") == "" {
		t.Error("missing Strict-Transport-Security header")
	}
	if w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("missing Referrer-Policy header")
	}
	if w.Header().Get("Permissions-Policy") == "" {
		t.Error("missing Permissions-Policy header")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("missing Cache-Control: no-store header on API response")
	}
}

func TestLogin_UnifiedErrorMessages(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// With no admin password set, login should return generic "invalid credentials"
	// (not "no admin password configured" which leaks setup state).
	body, _ := json.Marshal(map[string]string{"password": "anything"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "invalid credentials" {
		t.Errorf("login error with no password = %q, want 'invalid credentials'", resp["error"])
	}

	// With wrong password, should return the same message.
	setAdminPassword(t, srv, "testpassword")
	body, _ = json.Marshal(map[string]string{"password": "wrong"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "invalid credentials" {
		t.Errorf("login error with wrong password = %q, want 'invalid credentials'", resp["error"])
	}
}

func TestQueryParamToken_BlockedForNonWebSocket(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	// Query param token should NOT work for normal API requests.
	req := httptest.NewRequest("GET", "/api/v1/firewall?token="+token, nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("query param token on non-WebSocket: status = %d, want 401", w.Code)
	}
}

func TestLogin_RateLimited(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")

	router := srv.Router()

	// 5 failed attempts.
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{"password": "wrong"})
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// 6th attempt should be rate-limited.
	body, _ := json.Marshal(map[string]string{"password": "testpassword"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("rate-limited login: status = %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("rate-limited response missing Retry-After header")
	}
}
