package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/scoututm/scoutd/internal/stats"
)

func statsRoutes(r chi.Router, s *Server) {
	r.Get("/stats/history", s.handleStatsHistory)
	r.Get("/stats/connections", s.handleStatsConnections)
}

func (s *Server) handleStatsConnections(w http.ResponseWriter, r *http.Request) {
	conns := stats.ReadConnections(50)
	if conns == nil {
		conns = []stats.Connection{}
	}
	writeJSON(w, http.StatusOK, conns)
}

func (s *Server) handleStatsHistory(w http.ResponseWriter, r *http.Request) {
	var seconds int
	switch r.URL.Query().Get("range") {
	case "30m":
		seconds = 1800
	case "1h":
		seconds = 3600
	case "12h":
		seconds = 43200
	case "24h":
		seconds = 86400
	default: // "5m" or unspecified
		seconds = 300
	}

	samples := s.StatsHistory.Query(seconds)
	if samples == nil {
		samples = []stats.ThroughputSample{}
	}
	writeJSON(w, http.StatusOK, samples)
}
