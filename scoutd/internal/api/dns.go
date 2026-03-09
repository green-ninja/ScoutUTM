package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/blocklist"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/dnslog"
)

func dnsRoutes(r chi.Router, s *Server) {
	r.Get("/dns", s.handleGetDNS)
	r.Put("/dns", s.handlePutDNS)
	r.Get("/dns/query-log", s.handleDNSQueryLog)
}

func (s *Server) handleGetDNS(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	if cfg == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config not loaded"})
		return
	}
	writeJSON(w, http.StatusOK, cfg.DNS)
}

func (s *Server) handleDNSQueryLog(w http.ResponseWriter, r *http.Request) {
	if s.DNSLogger == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	entries := s.DNSLogger.RecentEntries(100)
	if entries == nil {
		entries = make([]dnslog.Entry, 0)
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handlePutDNS(w http.ResponseWriter, r *http.Request) {
	var req config.DNSConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg := s.requireConfig(w)
	if cfg == nil {
		return
	}

	// If BlockThreats is being enabled, fetch the blocklist before applying
	// so Unbound has a populated zone file to load.
	if req.BlockThreats && !cfg.DNS.BlockThreats {
		n, err := blocklist.FetchThreatsBlocklist("")
		if err != nil {
			log.Printf("threats blocklist fetch: %v", err)
		} else {
			log.Printf("threats blocklist: loaded %d domains", n)
		}
	}

	cfg.DNS = req

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	auditLog(r, "dns_config_update")

	writeJSON(w, http.StatusOK, cfg.DNS)
}
