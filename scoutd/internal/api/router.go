package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/ids"
	"github.com/scoututm/scoutd/internal/render"
	"github.com/scoututm/scoutd/internal/service"
	"github.com/scoututm/scoutd/internal/ws"
)

// Server holds dependencies for all API handlers.
type Server struct {
	Config      *config.Store
	Sessions    *auth.SessionManager
	RateLimiter *auth.LoginRateLimiter
	Hub         *ws.Hub
	Render      *render.Engine
	Services    *service.Manager
	EVEReader   *ids.Reader
	DevMode     bool
	Version     string

	startTime time.Time
}

// NewServer creates an API server.
func NewServer(cfg *config.Store, sessions *auth.SessionManager, hub *ws.Hub, renderer *render.Engine, svcMgr *service.Manager, eveReader *ids.Reader, devMode bool, version string) *Server {
	return &Server{
		Config:      cfg,
		Sessions:    sessions,
		RateLimiter: auth.NewLoginRateLimiter(),
		Hub:         hub,
		Render:      renderer,
		Services:    svcMgr,
		EVEReader:   eveReader,
		DevMode:     devMode,
		Version:     version,
		startTime:   time.Now(),
	}
}

// Router builds the chi router with all routes and middleware.
func (s *Server) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(maxBodySize(1 << 20)) // 1MB request body limit

	r.Route("/api/v1", func(r chi.Router) {
		// Unauthenticated routes.
		r.Get("/health", s.handleHealth)
		r.Post("/auth/login", s.handleLogin)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Post("/auth/logout", s.handleLogout)

			firewallRoutes(r, s)
			dnsRoutes(r, s)
			vpnRoutes(r, s)
			networkRoutes(r, s)
			systemRoutes(r, s)
			devicesRoutes(r, s)
			idsRoutes(r, s)
			parentalRoutes(r, s)
			wizardRoutes(r, s)
		})
	})

	return r
}

// handleHealth returns system health status (unauthenticated).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"healthy": true,
		"uptime":  int(time.Since(s.startTime).Seconds()),
		"version": s.Version,
	})
}

// handleLogin authenticates with password and returns a session token.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr

	// Rate limiting check.
	if !s.RateLimiter.Allow(ip) {
		w.Header().Set("Retry-After", "300")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts"})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	cfg := s.Config.Current()
	if cfg == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config not loaded"})
		return
	}

	if cfg.System.AdminHash == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "no admin password configured"})
		return
	}

	if !auth.CheckPassword(req.Password, cfg.System.AdminHash) {
		s.RateLimiter.RecordFailure(ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid password"})
		return
	}

	s.RateLimiter.ResetIP(ip)

	token, err := s.Sessions.Create()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scout_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// handleLogout revokes the current session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		s.Sessions.Revoke(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scout_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// authMiddleware checks for a valid session token.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" || !s.Sessions.Validate(token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleNotImplemented returns 501 for stub endpoints.
func handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func extractToken(r *http.Request) string {
	// Check Authorization header first.
	if h := r.Header.Get("Authorization"); h != "" {
		if strings.HasPrefix(h, "Bearer ") {
			return strings.TrimPrefix(h, "Bearer ")
		}
	}
	// Fall back to cookie.
	if c, err := r.Cookie("scout_session"); err == nil {
		return c.Value
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// maxBodySize limits the size of incoming request bodies.
func maxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
