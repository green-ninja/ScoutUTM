package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
)

func wizardRoutes(r chi.Router, s *Server) {
	r.Post("/wizard/step/{step}", s.handleWizardStep)
}

// wizardProgress tracks which steps have been completed (in-memory, not persisted).
var (
	wizardMu       sync.Mutex
	wizardComplete = make(map[int]bool)
)

func (s *Server) handleWizardStep(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	if cfg.System.SetupDone {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "setup already complete"})
		return
	}

	stepStr := chi.URLParam(r, "step")
	step, err := strconv.Atoi(stepStr)
	if err != nil || step < 1 || step > 7 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid step number"})
		return
	}

	// Guard: steps 2-7 require previous step completed.
	wizardMu.Lock()
	if step > 1 && !wizardComplete[step-1] {
		wizardMu.Unlock()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "previous step not completed"})
		return
	}
	wizardMu.Unlock()

	switch step {
	case 1:
		s.wizardStep1(w, r)
	case 2:
		s.wizardStep2(w, r)
	case 3:
		s.wizardStep3(w, r)
	case 4:
		s.wizardStep4(w, r)
	case 5:
		s.wizardStep5(w, r)
	case 6:
		s.wizardStep6(w, r)
	case 7:
		s.wizardStep7(w, r)
	}
}

func markComplete(step int) {
	wizardMu.Lock()
	wizardComplete[step] = true
	wizardMu.Unlock()
}

// Step 1: Welcome — no-op.
func (s *Server) wizardStep1(w http.ResponseWriter, r *http.Request) {
	markComplete(1)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Step 2: Set admin password.
func (s *Server) wizardStep2(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if len(req.Password) < 10 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 10 characters"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.System.AdminHash = hash

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	markComplete(2)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Step 3: Network detection — return detected interfaces.
func (s *Server) wizardStep3(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()

	type ifaceInfo struct {
		Name string `json:"name"`
		Zone string `json:"zone"`
		Type string `json:"type"`
	}

	ifaces := make([]ifaceInfo, len(cfg.Network.Interfaces))
	for i, iface := range cfg.Network.Interfaces {
		ifaces[i] = ifaceInfo{
			Name: iface.Name,
			Zone: iface.Zone,
			Type: iface.Type,
		}
	}

	markComplete(3)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"interfaces": ifaces,
	})
}

// Step 4: LAN configuration.
func (s *Server) wizardStep4(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LANAddress string `json:"lan_address"`
		DHCPStart  string `json:"dhcp_start"`
		DHCPEnd    string `json:"dhcp_end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Update LAN interface address if provided.
	if req.LANAddress != "" {
		for i, iface := range cfg.Network.Interfaces {
			if iface.Zone == "lan" {
				cfg.Network.Interfaces[i].IPv4Address = req.LANAddress
				break
			}
		}
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	markComplete(4)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Step 5: Security defaults.
func (s *Server) wizardStep5(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDSEnabled    *bool `json:"ids_enabled,omitempty"`
		IDSMode       string `json:"ids_mode,omitempty"`
		BlockMalware  *bool `json:"block_malware,omitempty"`
		BlockPhishing *bool `json:"block_phishing,omitempty"`
		BlockAds      *bool `json:"block_ads,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if req.IDSEnabled != nil {
		cfg.IDS.Enabled = *req.IDSEnabled
	}
	if req.IDSMode == "ids" || req.IDSMode == "ips" {
		cfg.IDS.Mode = req.IDSMode
	}
	if req.BlockMalware != nil {
		cfg.DNS.BlockMalware = *req.BlockMalware
	}
	if req.BlockPhishing != nil {
		cfg.DNS.BlockPhishing = *req.BlockPhishing
	}
	if req.BlockAds != nil {
		cfg.DNS.BlockAds = *req.BlockAds
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	markComplete(5)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Step 6: VLAN setup (optional).
func (s *Server) wizardStep6(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VLANs []config.VLANConfig `json:"vlans"`
		Zones []config.ZoneConfig `json:"zones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Add new zones (don't overwrite existing).
	existingZones := make(map[string]bool)
	for _, z := range cfg.Network.Zones {
		existingZones[z.Name] = true
	}
	for _, z := range req.Zones {
		if !existingZones[z.Name] {
			cfg.Network.Zones = append(cfg.Network.Zones, z)
		}
	}

	// Add VLANs.
	if len(req.VLANs) > 0 {
		cfg.Network.VLANs = append(cfg.Network.VLANs, req.VLANs...)
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	markComplete(6)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Step 7: Apply — mark setup done.
func (s *Server) wizardStep7(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	cfg.System.SetupDone = true

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	markComplete(7)

	// Reset wizard progress.
	wizardMu.Lock()
	wizardComplete = make(map[int]bool)
	wizardMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
