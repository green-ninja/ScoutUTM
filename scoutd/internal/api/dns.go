package api

import "github.com/go-chi/chi/v5"

func dnsRoutes(r chi.Router, s *Server) {
	r.Get("/dns", handleNotImplemented)
	r.Put("/dns", handleNotImplemented)
	r.Get("/dns/query-log", handleNotImplemented)
}
