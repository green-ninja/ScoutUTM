package api

import "github.com/go-chi/chi/v5"

func wizardRoutes(r chi.Router, s *Server) {
	r.Post("/wizard/step/{step}", handleNotImplemented)
}
