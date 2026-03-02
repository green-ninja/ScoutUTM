package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/config"
)

func parentalRoutes(r chi.Router, s *Server) {
	r.Get("/parental", s.handleGetParental)
	r.Get("/parental/profiles", s.handleListProfiles)
	r.Post("/parental/profiles", s.handleCreateProfile)
	r.Put("/parental/profiles/{id}", s.handleUpdateProfile)
	r.Delete("/parental/profiles/{id}", s.handleDeleteProfile)
	r.Put("/parental/devices", s.handleAssignDevices)
}

var allowedCategories = map[string]bool{
	"social_media": true,
	"gaming":       true,
	"adult":        true,
	"streaming":    true,
	"gambling":     true,
	"drugs":        true,
	"weapons":      true,
	"malware":      true,
}

var validDays = map[string]bool{
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true,
	"friday": true, "saturday": true, "sunday": true,
}

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func (s *Server) handleGetParental(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Parental)
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Parental.Profiles)
}

type createProfileRequest struct {
	Name              string          `json:"name"`
	BlockCategories   []string        `json:"block_categories"`
	SafeSearch        bool            `json:"safe_search"`
	YoutubeRestricted bool            `json:"youtube_restricted"`
	Schedule          config.Schedule `json:"schedule"`
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if err := validateProfile(req.BlockCategories, req.Schedule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Check unique name.
	for _, p := range cfg.Parental.Profiles {
		if strings.EqualFold(p.Name, req.Name) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "profile name already exists"})
			return
		}
	}

	profile := config.ParentalProfile{
		ID:                generateID(),
		Name:              req.Name,
		BlockCategories:   req.BlockCategories,
		SafeSearch:        req.SafeSearch,
		YoutubeRestricted: req.YoutubeRestricted,
		Schedule:          req.Schedule,
	}
	if profile.BlockCategories == nil {
		profile.BlockCategories = []string{}
	}
	if profile.Schedule.Days == nil {
		profile.Schedule.Days = []string{}
	}

	cfg.Parental.Profiles = append(cfg.Parental.Profiles, profile)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	if err := validateProfile(req.BlockCategories, req.Schedule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	idx := -1
	for i, p := range cfg.Parental.Profiles {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
		return
	}

	// Check unique name (excluding self).
	for _, p := range cfg.Parental.Profiles {
		if p.ID != id && strings.EqualFold(p.Name, req.Name) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "profile name already exists"})
			return
		}
	}

	cfg.Parental.Profiles[idx].Name = req.Name
	cfg.Parental.Profiles[idx].BlockCategories = req.BlockCategories
	cfg.Parental.Profiles[idx].SafeSearch = req.SafeSearch
	cfg.Parental.Profiles[idx].YoutubeRestricted = req.YoutubeRestricted
	cfg.Parental.Profiles[idx].Schedule = req.Schedule
	if cfg.Parental.Profiles[idx].BlockCategories == nil {
		cfg.Parental.Profiles[idx].BlockCategories = []string{}
	}
	if cfg.Parental.Profiles[idx].Schedule.Days == nil {
		cfg.Parental.Profiles[idx].Schedule.Days = []string{}
	}

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cfg.Parental.Profiles[idx])
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	idx := -1
	for i, p := range cfg.Parental.Profiles {
		if p.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "profile not found"})
		return
	}

	// Remove profile.
	cfg.Parental.Profiles = append(cfg.Parental.Profiles[:idx], cfg.Parental.Profiles[idx+1:]...)

	// Unassign devices that referenced this profile.
	filtered := make([]config.ParentalDevice, 0, len(cfg.Parental.Devices))
	for _, d := range cfg.Parental.Devices {
		if d.ProfileID != id {
			filtered = append(filtered, d)
		}
	}
	cfg.Parental.Devices = filtered

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type deviceAssignment struct {
	MAC       string `json:"mac"`
	ProfileID string `json:"profile_id"`
}

func (s *Server) handleAssignDevices(w http.ResponseWriter, r *http.Request) {
	var assignments []deviceAssignment
	if err := json.NewDecoder(r.Body).Decode(&assignments); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Build profile ID set for validation.
	profileIDs := make(map[string]bool)
	for _, p := range cfg.Parental.Profiles {
		profileIDs[p.ID] = true
	}

	// Validate assignments.
	for _, a := range assignments {
		if a.ProfileID != "" && !profileIDs[a.ProfileID] {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("unknown profile_id %q", a.ProfileID),
			})
			return
		}
	}

	// Build MAC→IP lookup from scanner.
	scannerDevices := s.Scanner.Devices()
	macToIP := make(map[string]string, len(scannerDevices))
	for _, d := range scannerDevices {
		macToIP[strings.ToLower(d.MAC)] = d.IP
	}

	// Replace device assignments.
	devices := make([]config.ParentalDevice, 0, len(assignments))
	for _, a := range assignments {
		if a.ProfileID != "" {
			mac := strings.ToLower(a.MAC)
			devices = append(devices, config.ParentalDevice{
				MAC:       mac,
				ProfileID: a.ProfileID,
				IP:        macToIP[mac],
			})
		}
	}
	cfg.Parental.Devices = devices

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cfg.Parental)
}

func validateProfile(categories []string, sched config.Schedule) error {
	for _, c := range categories {
		if !allowedCategories[c] {
			return fmt.Errorf("invalid category %q", c)
		}
	}
	if sched.Enabled {
		if sched.StartTime != "" && !timeRe.MatchString(sched.StartTime) {
			return fmt.Errorf("invalid start_time format, expected HH:MM")
		}
		if sched.EndTime != "" && !timeRe.MatchString(sched.EndTime) {
			return fmt.Errorf("invalid end_time format, expected HH:MM")
		}
		for _, d := range sched.Days {
			if !validDays[strings.ToLower(d)] {
				return fmt.Errorf("invalid day %q", d)
			}
		}
	}
	return nil
}
