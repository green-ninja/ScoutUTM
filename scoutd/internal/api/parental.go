package api

import "github.com/go-chi/chi/v5"

func parentalRoutes(r chi.Router, s *Server) {
	r.Get("/parental", handleNotImplemented)
	r.Get("/parental/profiles", handleNotImplemented)
	r.Post("/parental/profiles", handleNotImplemented)
	r.Put("/parental/profiles/{id}", handleNotImplemented)
	r.Delete("/parental/profiles/{id}", handleNotImplemented)
	r.Put("/parental/devices", handleNotImplemented)
}
