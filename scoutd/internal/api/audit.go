package api

import (
	"log"
	"net"
	"net/http"
)

// auditLog writes a structured audit log entry for security-relevant actions.
func auditLog(r *http.Request, action string) {
	log.Printf("[audit] action=%s ip=%s method=%s path=%s", action, clientIP(r), r.Method, r.URL.Path)
}

// clientIP extracts the client IP address, stripping the port.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
