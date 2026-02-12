package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func systemRoutes(r chi.Router, s *Server) {
	r.Get("/system", s.handleGetSystem)
	r.Patch("/system", s.handlePatchSystem)
	r.Post("/system/reboot", handleNotImplemented)
	r.Get("/system/backup", handleNotImplemented)
	r.Post("/system/restore", handleNotImplemented)
	r.Post("/system/regenerate-cert", handleNotImplemented)
	r.Get("/system/about", handleNotImplemented)
}

// systemResponse is the public system config (excludes admin_hash).
type systemResponse struct {
	Hostname   string   `json:"hostname"`
	Timezone   string   `json:"timezone"`
	SetupDone  bool     `json:"setup_done"`
	Language   string   `json:"language"`
	NTPServers []string `json:"ntp_servers"`
}

func (s *Server) handleGetSystem(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, systemResponse{
		Hostname:   cfg.System.Hostname,
		Timezone:   cfg.System.Timezone,
		SetupDone:  cfg.System.SetupDone,
		Language:   cfg.System.Language,
		NTPServers: cfg.System.NTPServers,
	})
}

// systemUpdateRequest supports partial updates to system config.
type systemUpdateRequest struct {
	Hostname   *string  `json:"hostname,omitempty"`
	Timezone   *string  `json:"timezone,omitempty"`
	Language   *string  `json:"language,omitempty"`
	NTPServers []string `json:"ntp_servers,omitempty"`
}

func (s *Server) handlePatchSystem(w http.ResponseWriter, r *http.Request) {
	var req systemUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if req.Hostname != nil {
		cfg.System.Hostname = *req.Hostname
	}
	if req.Timezone != nil {
		cfg.System.Timezone = *req.Timezone
	}
	if req.Language != nil {
		cfg.System.Language = *req.Language
	}
	if req.NTPServers != nil {
		cfg.System.NTPServers = req.NTPServers
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, systemResponse{
		Hostname:   cfg.System.Hostname,
		Timezone:   cfg.System.Timezone,
		SetupDone:  cfg.System.SetupDone,
		Language:   cfg.System.Language,
		NTPServers: cfg.System.NTPServers,
	})
}
