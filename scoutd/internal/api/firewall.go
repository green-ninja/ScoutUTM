package api

import "github.com/go-chi/chi/v5"

func firewallRoutes(r chi.Router, s *Server) {
	r.Get("/firewall", handleNotImplemented)
	r.Put("/firewall", handleNotImplemented)
	r.Get("/firewall/rules", handleNotImplemented)
	r.Post("/firewall/rules", handleNotImplemented)
	r.Put("/firewall/rules/{id}", handleNotImplemented)
	r.Delete("/firewall/rules/{id}", handleNotImplemented)
	r.Get("/firewall/port-forwards", handleNotImplemented)
	r.Post("/firewall/port-forwards", handleNotImplemented)
	r.Delete("/firewall/port-forwards/{id}", handleNotImplemented)
}
