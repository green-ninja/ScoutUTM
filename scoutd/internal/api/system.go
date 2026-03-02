package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
)

func systemRoutes(r chi.Router, s *Server) {
	r.Get("/system", s.handleGetSystem)
	r.Patch("/system", s.handlePatchSystem)
	r.Post("/system/reboot", s.handleReboot)
	r.Get("/system/backup", s.handleBackup)
	r.Post("/system/restore", s.handleRestore)
	r.Post("/system/regenerate-cert", s.handleRegenerateCert)
	r.Get("/system/about", s.handleAbout)
	r.Post("/system/password", s.handleChangePassword)
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

func (s *Server) handleReboot(w http.ResponseWriter, r *http.Request) {
	if s.DevMode {
		log.Println("[dev-mode] reboot requested")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rebooting"})
	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command("reboot").Run()
	}()
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to marshal config"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=scout-config.json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	// Accept either multipart form or raw JSON body.
	var data []byte
	var err error

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		file, _, fErr := r.FormFile("config")
		if fErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing config file"})
			return
		}
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, 1<<20))
	} else {
		data, err = io.ReadAll(io.LimitReader(r.Body, 1<<20))
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid config JSON"})
		return
	}

	if err := s.applyConfig(&cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRegenerateCert(w http.ResponseWriter, r *http.Request) {
	if s.DevMode {
		log.Println("[dev-mode] certificate regeneration requested")
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	// In production, TLS cert regeneration requires restart.
	// For now, just acknowledge.
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "restart required"})
}

type aboutResponse struct {
	Version   string `json:"version"`
	Uptime    int    `json:"uptime"`
	GoVersion string `json:"go_version"`
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, aboutResponse{
		Version:   s.Version,
		Uptime:    int(time.Since(s.startTime).Seconds()),
		GoVersion: runtime.Version(),
	})
}

type changePasswordRequest struct {
	Current     string `json:"current"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg := s.Config.Current()

	// Verify current password.
	if !auth.CheckPassword(req.Current, cfg.System.AdminHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
		return
	}

	if len(req.NewPassword) < 10 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "new password must be at least 10 characters"})
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	newCfg, err := cfg.DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	newCfg.System.AdminHash = hash

	if err := s.applyConfig(newCfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
