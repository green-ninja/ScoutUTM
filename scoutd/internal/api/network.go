package api

import "github.com/go-chi/chi/v5"

func networkRoutes(r chi.Router, s *Server) {
	r.Get("/network", handleNotImplemented)
	r.Put("/network", handleNotImplemented)
	r.Get("/network/zones", handleNotImplemented)
	r.Post("/network/zones", handleNotImplemented)
	r.Delete("/network/zones/{name}", handleNotImplemented)
	r.Get("/network/vlans", handleNotImplemented)
	r.Post("/network/vlans", handleNotImplemented)
	r.Delete("/network/vlans/{id}", handleNotImplemented)
}
