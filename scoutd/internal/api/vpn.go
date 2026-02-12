package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/scoututm/scoutd/internal/config"
	"golang.org/x/crypto/curve25519"
)

func vpnRoutes(r chi.Router, s *Server) {
	r.Get("/vpn/wireguard", s.handleGetWireGuard)
	r.Put("/vpn/wireguard", s.handlePutWireGuard)
	r.Get("/vpn/wireguard/peers", s.handleGetPeers)
	r.Post("/vpn/wireguard/peers", s.handleCreatePeer)
	r.Delete("/vpn/wireguard/peers/{id}", s.handleDeletePeer)
	r.Post("/vpn/wireguard/regenerate-keys", s.handleRegenerateKeys)
}

// wireGuardResponse is the public view of WireGuard config (private key excluded).
type wireGuardResponse struct {
	Enabled          bool                  `json:"enabled"`
	ListenPort       int                   `json:"listen_port"`
	Address          string                `json:"address"`
	TunnelSubnet     string                `json:"tunnel_subnet"`
	EndpointOverride string                `json:"endpoint_override,omitempty"`
	PublicKey        string                `json:"public_key,omitempty"`
	KeyCreatedAt     string                `json:"key_created_at,omitempty"`
	Peers            []config.WireGuardPeer `json:"peers"`
}

func (s *Server) handleGetWireGuard(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	wg := cfg.VPN.WireGuard

	resp := wireGuardResponse{
		Enabled:          wg.Enabled,
		ListenPort:       wg.ListenPort,
		Address:          wg.Address,
		TunnelSubnet:     wg.TunnelSubnet,
		EndpointOverride: wg.EndpointOverride,
		KeyCreatedAt:     wg.KeyCreatedAt,
		Peers:            wg.Peers,
	}

	// Derive public key from private key if present.
	if wg.PrivateKey != "" {
		pub, err := derivePublicKey(wg.PrivateKey)
		if err == nil {
			resp.PublicKey = pub
		}
	}

	if resp.Peers == nil {
		resp.Peers = []config.WireGuardPeer{}
	}

	writeJSON(w, http.StatusOK, resp)
}

// wireGuardUpdate is the request body for PUT /vpn/wireguard.
type wireGuardUpdate struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	ListenPort       *int    `json:"listen_port,omitempty"`
	Address          *string `json:"address,omitempty"`
	TunnelSubnet     *string `json:"tunnel_subnet,omitempty"`
	EndpointOverride *string `json:"endpoint_override,omitempty"`
}

func (s *Server) handlePutWireGuard(w http.ResponseWriter, r *http.Request) {
	var update wireGuardUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	wg := &cfg.VPN.WireGuard

	if update.Enabled != nil {
		wg.Enabled = *update.Enabled
	}
	if update.ListenPort != nil {
		wg.ListenPort = *update.ListenPort
	}
	if update.Address != nil {
		wg.Address = *update.Address
	}
	if update.TunnelSubnet != nil {
		wg.TunnelSubnet = *update.TunnelSubnet
	}
	if update.EndpointOverride != nil {
		wg.EndpointOverride = *update.EndpointOverride
	}

	// Auto-generate server keys if enabling and no key exists.
	if wg.Enabled && wg.PrivateKey == "" {
		priv, err := generatePrivateKey()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key generation failed"})
			return
		}
		wg.PrivateKey = priv
		wg.KeyCreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	// Return the updated config (no private key).
	s.handleGetWireGuard(w, r)
}

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	peers := cfg.VPN.WireGuard.Peers
	if peers == nil {
		peers = []config.WireGuardPeer{}
	}
	writeJSON(w, http.StatusOK, peers)
}

// createPeerRequest is the request body for POST /vpn/wireguard/peers.
type createPeerRequest struct {
	Name string `json:"name"`
}

// createPeerResponse is returned when a peer is created.
type createPeerResponse struct {
	Peer         config.WireGuardPeer `json:"peer"`
	ClientConfig string               `json:"client_config"`
	QRCode       string               `json:"qr_code"`
}

func (s *Server) handleCreatePeer(w http.ResponseWriter, r *http.Request) {
	var req createPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	wg := &cfg.VPN.WireGuard

	if !wg.Enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "WireGuard is not enabled"})
		return
	}

	if wg.PrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "server has no private key"})
		return
	}

	// Generate client keypair.
	clientPriv, err := generatePrivateKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key generation failed"})
		return
	}
	clientPub, err := derivePublicKey(clientPriv)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key derivation failed"})
		return
	}

	// Allocate tunnel IP.
	tunnelIP, err := allocateTunnelIP(wg.TunnelSubnet, wg.Peers)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	// Derive server public key.
	serverPub, err := derivePublicKey(wg.PrivateKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server key derivation failed"})
		return
	}

	peer := config.WireGuardPeer{
		Name:       req.Name,
		PublicKey:  clientPub,
		AllowedIPs: []string{tunnelIP + "/32"},
		TunnelIP:   tunnelIP,
	}

	wg.Peers = append(wg.Peers, peer)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	// Build client config.
	endpoint := wg.EndpointOverride
	if endpoint == "" {
		endpoint = "YOUR_SERVER_IP"
	}
	clientConfig := fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s/32\nDNS = %s\n\n[Peer]\nPublicKey = %s\nEndpoint = %s:%d\nAllowedIPs = 0.0.0.0/0\nPersistentKeepalive = 25\n",
		clientPriv,
		tunnelIP,
		extractIP(wg.Address),
		serverPub,
		endpoint,
		wg.ListenPort,
	)

	// Generate QR code.
	png, err := qrcode.Encode(clientConfig, qrcode.Medium, 256)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "QR code generation failed"})
		return
	}

	resp := createPeerResponse{
		Peer:         peer,
		ClientConfig: clientConfig,
		QRCode:       base64.StdEncoding.EncodeToString(png),
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleDeletePeer(w http.ResponseWriter, r *http.Request) {
	peerID := chi.URLParam(r, "id")

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	found := false
	peers := make([]config.WireGuardPeer, 0, len(cfg.VPN.WireGuard.Peers))
	for _, p := range cfg.VPN.WireGuard.Peers {
		if p.Name == peerID {
			found = true
			continue
		}
		peers = append(peers, p)
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "peer not found"})
		return
	}

	cfg.VPN.WireGuard.Peers = peers

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRegenerateKeys(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	priv, err := generatePrivateKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "key generation failed"})
		return
	}

	cfg.VPN.WireGuard.PrivateKey = priv
	cfg.VPN.WireGuard.KeyCreatedAt = time.Now().UTC().Format(time.RFC3339)
	cfg.VPN.WireGuard.Peers = []config.WireGuardPeer{} // Clear all peers.

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	// Return the updated config.
	s.handleGetWireGuard(w, r)
}

// generatePrivateKey creates a new WireGuard private key (32 random bytes, clamped per Curve25519).
func generatePrivateKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	// Clamp per Curve25519 spec.
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// derivePublicKey derives a WireGuard public key from a base64-encoded private key.
func derivePublicKey(privB64 string) (string, error) {
	privBytes, err := base64.StdEncoding.DecodeString(privB64)
	if err != nil {
		return "", err
	}
	if len(privBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: %d", len(privBytes))
	}
	var priv, pub [32]byte
	copy(priv[:], privBytes)
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(pub[:]), nil
}

// allocateTunnelIP finds the next available IP in the tunnel subnet.
// Skips .0 (network) and .1 (server). Returns the first unused IP.
func allocateTunnelIP(subnetCIDR string, peers []config.WireGuardPeer) (string, error) {
	_, ipnet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return "", fmt.Errorf("invalid tunnel subnet: %s", subnetCIDR)
	}

	used := make(map[string]bool)
	for _, p := range peers {
		used[p.TunnelIP] = true
	}

	// Convert network IP to uint32 for iteration.
	networkIP := ipnet.IP.To4()
	if networkIP == nil {
		return "", fmt.Errorf("only IPv4 supported")
	}
	start := binary.BigEndian.Uint32(networkIP)

	// Calculate broadcast address.
	ones, bits := ipnet.Mask.Size()
	hostBits := uint(bits - ones)
	broadcast := start | ((1 << hostBits) - 1)

	// Start from .2 (skip .0 network and .1 server).
	for ip := start + 2; ip < broadcast; ip++ {
		candidate := make(net.IP, 4)
		binary.BigEndian.PutUint32(candidate, ip)
		candidateStr := candidate.String()
		if !used[candidateStr] {
			return candidateStr, nil
		}
	}

	return "", fmt.Errorf("no available tunnel IPs in %s", subnetCIDR)
}

// extractIP returns the IP portion of a CIDR string.
func extractIP(cidr string) string {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}
	return ip.String()
}
