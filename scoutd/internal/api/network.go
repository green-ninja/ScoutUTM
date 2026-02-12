package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/config"
)

func networkRoutes(r chi.Router, s *Server) {
	r.Get("/network", s.handleGetNetwork)
	r.Put("/network", s.handlePutNetwork)
	r.Get("/network/zones", s.handleGetZones)
	r.Post("/network/zones", s.handleCreateZone)
	r.Delete("/network/zones/{name}", s.handleDeleteZone)
	r.Get("/network/vlans", s.handleGetVLANs)
	r.Post("/network/vlans", s.handleCreateVLAN)
	r.Delete("/network/vlans/{id}", s.handleDeleteVLAN)
}

func (s *Server) handleGetNetwork(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Network)
}

func (s *Server) handlePutNetwork(w http.ResponseWriter, r *http.Request) {
	var net config.NetworkConfig
	if err := json.NewDecoder(r.Body).Decode(&net); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.Network = net

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg.Network)
}

func (s *Server) handleGetZones(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Network.Zones)
}

func (s *Server) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	var zone config.ZoneConfig
	if err := json.NewDecoder(r.Body).Decode(&zone); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.Network.Zones = append(cfg.Network.Zones, zone)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, zone)
}

func (s *Server) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Protect built-in zones.
	if name == "wan" || name == "lan" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete built-in zone"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	found := false
	zones := make([]config.ZoneConfig, 0, len(cfg.Network.Zones))
	for _, z := range cfg.Network.Zones {
		if z.Name == name {
			found = true
			continue
		}
		zones = append(zones, z)
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "zone not found"})
		return
	}

	// Check for firewall forwardings that reference this zone.
	for _, fwd := range cfg.Firewall.Forwardings {
		if fwd.Src == name || fwd.Dest == name {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": fmt.Sprintf("zone %q is referenced by a firewall forwarding rule", name),
			})
			return
		}
	}

	// Check for firewall rules that reference this zone.
	for _, rule := range cfg.Firewall.Rules {
		if rule.Src == name || rule.Dest == name {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": fmt.Sprintf("zone %q is referenced by firewall rule %q", name, rule.Name),
			})
			return
		}
	}

	cfg.Network.Zones = zones

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetVLANs(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Network.VLANs)
}

func (s *Server) handleCreateVLAN(w http.ResponseWriter, r *http.Request) {
	var vlan config.VLANConfig
	if err := json.NewDecoder(r.Body).Decode(&vlan); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Check for duplicate VLAN ID.
	for _, v := range cfg.Network.VLANs {
		if v.ID == vlan.ID {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("VLAN ID %d already exists", vlan.ID)})
			return
		}
	}

	cfg.Network.VLANs = append(cfg.Network.VLANs, vlan)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, vlan)
}

func (s *Server) handleDeleteVLAN(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid VLAN ID"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Find the VLAN and its zone.
	found := false
	var vlanZone string
	vlans := make([]config.VLANConfig, 0, len(cfg.Network.VLANs))
	for _, v := range cfg.Network.VLANs {
		if v.ID == id {
			found = true
			vlanZone = v.Zone
			continue
		}
		vlans = append(vlans, v)
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "VLAN not found"})
		return
	}

	// Check for firewall forwardings that reference this VLAN's zone.
	if vlanZone != "" {
		for _, fwd := range cfg.Firewall.Forwardings {
			if fwd.Src == vlanZone || fwd.Dest == vlanZone {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error": fmt.Sprintf("VLAN zone %q is referenced by a firewall forwarding rule", vlanZone),
				})
				return
			}
		}
		for _, rule := range cfg.Firewall.Rules {
			if rule.Src == vlanZone || rule.Dest == vlanZone {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error": fmt.Sprintf("VLAN zone %q is referenced by firewall rule %q", vlanZone, rule.Name),
				})
				return
			}
		}
	}

	cfg.Network.VLANs = vlans

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
