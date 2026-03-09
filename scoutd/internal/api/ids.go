package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func idsRoutes(r chi.Router, s *Server) {
	r.Get("/ids", s.handleGetIDS)
	r.Put("/ids", s.handlePutIDS)
	r.Get("/ids/alerts", s.handleGetAlerts)
}

func (s *Server) handleGetIDS(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	if cfg == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config not loaded"})
		return
	}
	writeJSON(w, http.StatusOK, cfg.IDS)
}

// idsUpdate is the request body for PUT /ids (partial update).
type idsUpdate struct {
	Enabled    *bool     `json:"enabled,omitempty"`
	Mode       *string   `json:"mode,omitempty"`
	Rulesets   *[]string `json:"rulesets,omitempty"`
	HomeNet    *[]string `json:"home_net,omitempty"`
	Suppressed *[]int    `json:"suppressed,omitempty"`
}

func (s *Server) handlePutIDS(w http.ResponseWriter, r *http.Request) {
	var update idsUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg := s.requireConfig(w)
	if cfg == nil {
		return
	}

	if update.Enabled != nil {
		cfg.IDS.Enabled = *update.Enabled
	}
	if update.Mode != nil {
		if *update.Mode != "ids" && *update.Mode != "ips" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mode must be \"ids\" or \"ips\""})
			return
		}
		cfg.IDS.Mode = *update.Mode
	}
	if update.Rulesets != nil {
		cfg.IDS.Rulesets = *update.Rulesets
	}
	if update.HomeNet != nil {
		cfg.IDS.HomeNet = *update.HomeNet
	}
	if update.Suppressed != nil {
		cfg.IDS.Suppressed = *update.Suppressed
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	auditLog(r, "ids_config_update")

	writeJSON(w, http.StatusOK, cfg.IDS)
}

func (s *Server) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	if s.EVEReader == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	alerts := s.EVEReader.Alerts().All()
	if len(alerts) == 0 {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}
