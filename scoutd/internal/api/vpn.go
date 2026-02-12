package api

import "github.com/go-chi/chi/v5"

func vpnRoutes(r chi.Router, s *Server) {
	r.Get("/vpn/wireguard", handleNotImplemented)
	r.Put("/vpn/wireguard", handleNotImplemented)
	r.Get("/vpn/wireguard/peers", handleNotImplemented)
	r.Post("/vpn/wireguard/peers", handleNotImplemented)
	r.Delete("/vpn/wireguard/peers/{id}", handleNotImplemented)
	r.Post("/vpn/wireguard/regenerate-keys", handleNotImplemented)
}
