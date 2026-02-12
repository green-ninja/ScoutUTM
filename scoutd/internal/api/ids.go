package api

import "github.com/go-chi/chi/v5"

func idsRoutes(r chi.Router, s *Server) {
	r.Get("/ids", handleNotImplemented)
	r.Put("/ids", handleNotImplemented)
	r.Get("/ids/alerts", handleNotImplemented)
}
