package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func enableWireGuard(t *testing.T, srv *Server) {
	t.Helper()
	cfg, err := srv.Config.Current().DeepCopy()
	if err != nil {
		t.Fatalf("DeepCopy: %v", err)
	}
	cfg.VPN.WireGuard.Enabled = true
	// Generate a test keypair.
	priv, err := generatePrivateKey()
	if err != nil {
		t.Fatalf("generatePrivateKey: %v", err)
	}
	cfg.VPN.WireGuard.PrivateKey = priv
	cfg.VPN.WireGuard.KeyCreatedAt = "2024-01-01T00:00:00Z"
	if err := srv.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
}

func TestGetWireGuard(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/vpn/wireguard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /vpn/wireguard: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp wireGuardResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ListenPort != 51820 {
		t.Errorf("listen_port = %d, want 51820", resp.ListenPort)
	}
}

func TestPutWireGuard(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	update := map[string]interface{}{
		"enabled":     true,
		"listen_port": 51821,
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/api/v1/vpn/wireguard", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /vpn/wireguard: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp wireGuardResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Enabled {
		t.Error("expected enabled = true")
	}
	if resp.ListenPort != 51821 {
		t.Errorf("listen_port = %d, want 51821", resp.ListenPort)
	}
	if resp.PublicKey == "" {
		t.Error("expected public_key to be derived after enabling")
	}
}

func TestGetPeers_Empty(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("GET", "/api/v1/vpn/wireguard/peers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /vpn/wireguard/peers: status = %d", w.Code)
	}

	var peers []config.WireGuardPeer
	json.Unmarshal(w.Body.Bytes(), &peers)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}
}

func TestCreatePeer(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	body, _ := json.Marshal(createPeerRequest{Name: "phone"})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /vpn/wireguard/peers: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp createPeerResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Peer.Name != "phone" {
		t.Errorf("peer name = %q, want phone", resp.Peer.Name)
	}
	if resp.Peer.TunnelIP != "10.0.0.2" {
		t.Errorf("tunnel IP = %q, want 10.0.0.2", resp.Peer.TunnelIP)
	}
	if resp.ClientConfig == "" {
		t.Error("expected client_config in response")
	}
	if resp.QRCode == "" {
		t.Error("expected qr_code in response")
	}
}

func TestCreatePeer_SequentialIPs(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	// Create first peer.
	body, _ := json.Marshal(createPeerRequest{Name: "peer1"})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("peer1: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp1 createPeerResponse
	json.Unmarshal(w.Body.Bytes(), &resp1)

	// Create second peer.
	body, _ = json.Marshal(createPeerRequest{Name: "peer2"})
	req = httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("peer2: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp2 createPeerResponse
	json.Unmarshal(w.Body.Bytes(), &resp2)

	if resp1.Peer.TunnelIP != "10.0.0.2" {
		t.Errorf("peer1 tunnel IP = %q, want 10.0.0.2", resp1.Peer.TunnelIP)
	}
	if resp2.Peer.TunnelIP != "10.0.0.3" {
		t.Errorf("peer2 tunnel IP = %q, want 10.0.0.3", resp2.Peer.TunnelIP)
	}
}

func TestDeletePeer(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	// Create peer.
	body, _ := json.Marshal(createPeerRequest{Name: "phone"})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create peer: status = %d", w.Code)
	}

	// Delete peer.
	req = httptest.NewRequest("DELETE", "/api/v1/vpn/wireguard/peers/phone", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE peer: status = %d, body = %s", w.Code, w.Body.String())
	}

	// Verify peer is gone.
	req = httptest.NewRequest("GET", "/api/v1/vpn/wireguard/peers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var peers []config.WireGuardPeer
	json.Unmarshal(w.Body.Bytes(), &peers)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers after delete, got %d", len(peers))
	}
}

func TestDeletePeer_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	req := httptest.NewRequest("DELETE", "/api/v1/vpn/wireguard/peers/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE nonexistent peer: status = %d, want 404", w.Code)
	}
}

func TestRegenerateKeys(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	// Create a peer first.
	body, _ := json.Marshal(createPeerRequest{Name: "phone"})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create peer: status = %d", w.Code)
	}

	// Get current public key.
	req = httptest.NewRequest("GET", "/api/v1/vpn/wireguard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var before wireGuardResponse
	json.Unmarshal(w.Body.Bytes(), &before)

	// Regenerate keys.
	req = httptest.NewRequest("POST", "/api/v1/vpn/wireguard/regenerate-keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("regenerate-keys: status = %d, body = %s", w.Code, w.Body.String())
	}

	var after wireGuardResponse
	json.Unmarshal(w.Body.Bytes(), &after)

	if after.PublicKey == before.PublicKey {
		t.Error("expected new public key after regeneration")
	}
	if len(after.Peers) != 0 {
		t.Errorf("expected 0 peers after regeneration, got %d", len(after.Peers))
	}
}

func TestCreatePeer_WireGuardDisabled(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")

	body, _ := json.Marshal(createPeerRequest{Name: "phone"})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("create peer with WG disabled: status = %d, want 400", w.Code)
	}
}

func TestCreatePeer_FullTunnel(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	body, _ := json.Marshal(createPeerRequest{Name: "phone", SplitTunnel: false})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /vpn/wireguard/peers: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp createPeerResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Full tunnel should have AllowedIPs = 0.0.0.0/0.
	if !bytes.Contains([]byte(resp.ClientConfig), []byte("AllowedIPs = 0.0.0.0/0")) {
		t.Errorf("full tunnel client config missing AllowedIPs = 0.0.0.0/0\n%s", resp.ClientConfig)
	}

	// DNS should point to LAN IP (192.168.1.1), not tunnel IP (10.0.0.1).
	if !bytes.Contains([]byte(resp.ClientConfig), []byte("DNS = 192.168.1.1")) {
		t.Errorf("client config should use LAN IP for DNS\n%s", resp.ClientConfig)
	}
}

func TestCreatePeer_SplitTunnel(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "test")
	token := loginAndGetToken(t, srv, "test")
	enableWireGuard(t, srv)
	router := srv.Router()

	body, _ := json.Marshal(createPeerRequest{Name: "laptop", SplitTunnel: true})
	req := httptest.NewRequest("POST", "/api/v1/vpn/wireguard/peers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /vpn/wireguard/peers: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp createPeerResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Split tunnel should NOT have AllowedIPs = 0.0.0.0/0.
	if bytes.Contains([]byte(resp.ClientConfig), []byte("AllowedIPs = 0.0.0.0/0")) {
		t.Errorf("split tunnel should not have AllowedIPs = 0.0.0.0/0\n%s", resp.ClientConfig)
	}

	// Should include the LAN subnet.
	if !bytes.Contains([]byte(resp.ClientConfig), []byte("192.168.1.0/24")) {
		t.Errorf("split tunnel should include LAN subnet\n%s", resp.ClientConfig)
	}

	// Should include the tunnel subnet so peers can reach the server.
	if !bytes.Contains([]byte(resp.ClientConfig), []byte("10.0.0.0/24")) {
		t.Errorf("split tunnel should include tunnel subnet\n%s", resp.ClientConfig)
	}
}

func TestAllocateTunnelIP(t *testing.T) {
	// Empty peers — should get .2
	ip, err := allocateTunnelIP("10.0.0.0/24", nil)
	if err != nil {
		t.Fatalf("allocateTunnelIP: %v", err)
	}
	if ip != "10.0.0.2" {
		t.Errorf("first IP = %q, want 10.0.0.2", ip)
	}

	// With .2 taken — should get .3
	peers := []config.WireGuardPeer{{TunnelIP: "10.0.0.2"}}
	ip, err = allocateTunnelIP("10.0.0.0/24", peers)
	if err != nil {
		t.Fatalf("allocateTunnelIP: %v", err)
	}
	if ip != "10.0.0.3" {
		t.Errorf("second IP = %q, want 10.0.0.3", ip)
	}
}

func TestDerivePublicKey(t *testing.T) {
	priv, err := generatePrivateKey()
	if err != nil {
		t.Fatalf("generatePrivateKey: %v", err)
	}

	pub, err := derivePublicKey(priv)
	if err != nil {
		t.Fatalf("derivePublicKey: %v", err)
	}
	if pub == "" {
		t.Error("expected non-empty public key")
	}
	if pub == priv {
		t.Error("public key should differ from private key")
	}
}
