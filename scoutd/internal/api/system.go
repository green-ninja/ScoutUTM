package api

import "github.com/go-chi/chi/v5"

func systemRoutes(r chi.Router, s *Server) {
	r.Get("/system", handleNotImplemented)
	r.Put("/system", handleNotImplemented)
	r.Post("/system/reboot", handleNotImplemented)
	r.Get("/system/backup", handleNotImplemented)
	r.Post("/system/restore", handleNotImplemented)
	r.Post("/system/regenerate-cert", handleNotImplemented)
	r.Get("/system/about", handleNotImplemented)
}
