package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/config"
)

func networkRoutes(r chi.Router, s *Server) {
	r.Get("/network", s.handleGetNetwork)
	r.Put("/network", s.handlePutNetwork)
	r.Get("/network/zones", s.handleGetZones)
	r.Post("/network/zones", s.handleCreateZone)
	r.Delete("/network/zones/{name}", s.handleDeleteZone)
	r.Get("/network/vlans", handleNotImplemented)
	r.Post("/network/vlans", handleNotImplemented)
	r.Delete("/network/vlans/{id}", handleNotImplemented)
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
