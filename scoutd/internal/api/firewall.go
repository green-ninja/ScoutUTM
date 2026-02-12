package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/config"
)

func firewallRoutes(r chi.Router, s *Server) {
	r.Get("/firewall", s.handleGetFirewall)
	r.Put("/firewall", s.handlePutFirewall)
	r.Get("/firewall/rules", s.handleGetRules)
	r.Post("/firewall/rules", s.handleCreateRule)
	r.Put("/firewall/rules/{id}", s.handleUpdateRule)
	r.Delete("/firewall/rules/{id}", s.handleDeleteRule)
	r.Get("/firewall/port-forwards", s.handleGetPortForwards)
	r.Post("/firewall/port-forwards", s.handleCreatePortForward)
	r.Delete("/firewall/port-forwards/{id}", s.handleDeletePortForward)
}

func (s *Server) handleGetFirewall(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Firewall)
}

func (s *Server) handlePutFirewall(w http.ResponseWriter, r *http.Request) {
	var fw config.FirewallConfig
	if err := json.NewDecoder(r.Body).Decode(&fw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.Firewall = fw

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg.Firewall)
}

func (s *Server) handleGetRules(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Firewall.Rules)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule config.FirewallRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.Firewall.Rules = append(cfg.Firewall.Rules, rule)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rule id"})
		return
	}

	var rule config.FirewallRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if id < 0 || id >= len(cfg.Firewall.Rules) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}
	cfg.Firewall.Rules[id] = rule

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid rule id"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if id < 0 || id >= len(cfg.Firewall.Rules) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}
	cfg.Firewall.Rules = append(cfg.Firewall.Rules[:id], cfg.Firewall.Rules[id+1:]...)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetPortForwards(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	writeJSON(w, http.StatusOK, cfg.Firewall.PortForwards)
}

func (s *Server) handleCreatePortForward(w http.ResponseWriter, r *http.Request) {
	var pf config.PortForward
	if err := json.NewDecoder(r.Body).Decode(&pf); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	cfg.Firewall.PortForwards = append(cfg.Firewall.PortForwards, pf)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pf)
}

func (s *Server) handleDeletePortForward(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid port forward id"})
		return
	}

	cfg, err := s.Config.Current().DeepCopy()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if id < 0 || id >= len(cfg.Firewall.PortForwards) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "port forward not found"})
		return
	}
	cfg.Firewall.PortForwards = append(cfg.Firewall.PortForwards[:id], cfg.Firewall.PortForwards[id+1:]...)

	if err := s.applyConfig(cfg); err != nil {
		handleApplyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
