package api

import "github.com/go-chi/chi/v5"

func devicesRoutes(r chi.Router, s *Server) {
	r.Get("/devices", handleNotImplemented)
	r.Get("/devices/{mac}", handleNotImplemented)
	r.Put("/devices/{mac}", handleNotImplemented)
}
