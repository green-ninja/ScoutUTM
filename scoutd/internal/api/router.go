package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/dnslog"
	"github.com/scoututm/scoutd/internal/ids"
	"github.com/scoututm/scoutd/internal/netmon"
	"github.com/scoututm/scoutd/internal/render"
	"github.com/scoututm/scoutd/internal/service"
	"github.com/scoututm/scoutd/internal/stats"
	"github.com/scoututm/scoutd/internal/ws"
	scoutweb "github.com/scoututm/scoutd/web"
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
	Scanner     *netmon.Scanner
	DNSLogger    *dnslog.Logger
	StatsHistory *stats.History
	DevMode      bool
	Version      string

	startTime time.Time
}

// NewServer creates an API server.
func NewServer(cfg *config.Store, sessions *auth.SessionManager, hub *ws.Hub, renderer *render.Engine, svcMgr *service.Manager, eveReader *ids.Reader, scanner *netmon.Scanner, devMode bool, version string) *Server {
	return &Server{
		Config:      cfg,
		Sessions:    sessions,
		RateLimiter: auth.NewLoginRateLimiter(),
		Hub:         hub,
		Render:      renderer,
		Services:    svcMgr,
		EVEReader:   eveReader,
		Scanner:     scanner,
		DevMode:     devMode,
		Version:     version,
		startTime:   time.Now(),
	}
}

// Router builds the chi router with all routes and middleware.
func (s *Server) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(redactTokenParam)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(checkOrigin)
	r.Use(maxBodySize(1 << 20)) // 1MB request body limit

	r.Route("/api/v1", func(r chi.Router) {
		// Unauthenticated routes.
		r.Get("/health", s.handleHealth)
		r.Post("/auth/login", s.handleLogin)

		// Wizard is unauthenticated (needed for first-boot setup when no
		// admin password exists yet). The handler guards with SetupDone check.
		wizardRoutes(r, s)

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware)

			r.Post("/auth/logout", s.handleLogout)
			r.Get("/ws", s.handleWebSocket)

			firewallRoutes(r, s)
			dnsRoutes(r, s)
			vpnRoutes(r, s)
			networkRoutes(r, s)
			systemRoutes(r, s)
			devicesRoutes(r, s)
			idsRoutes(r, s)
			parentalRoutes(r, s)
			statsRoutes(r, s)
		})
	})

	// SPA file server: serve embedded frontend, fallback to index.html.
	if dist, err := fs.Sub(scoutweb.FS, "dist"); err == nil {
		if _, err := dist.Open("index.html"); err == nil {
			r.NotFound(spaHandler(dist))
		}
	}

	return r
}

// spaHandler serves static files from the embedded FS, falling back to
// index.html for paths that don't match a file (SPA client-side routing).
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := fsys.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// File not found — serve index.html for SPA routing.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}
}

// handleHealth returns system health status (unauthenticated).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := s.Config.Current()
	setupDone := false
	if cfg != nil {
		setupDone = cfg.System.SetupDone
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"healthy":    true,
		"uptime":     int(time.Since(s.startTime).Seconds()),
		"version":    s.Version,
		"setup_done": setupDone,
	})
}

// handleLogin authenticates with password and returns a session token.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	// Rate limiting check.
	if !s.RateLimiter.Allow(ip) {
		auditLog(r, "login_rate_limited")
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

	// Use a single error message for all auth failures to prevent
	// enumeration of whether the admin password is configured.
	if cfg.System.AdminHash == "" || !auth.CheckPassword(req.Password, cfg.System.AdminHash) {
		s.RateLimiter.RecordFailure(ip)
		auditLog(r, "login_failed")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	s.RateLimiter.ResetIP(ip)
	auditLog(r, "login_success")

	token, err := s.Sessions.Create()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scout_session",
		Value:    token,
		Path:     "/api/v1",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// handleLogout revokes the current session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	auditLog(r, "logout")
	token := extractToken(r)
	if token != "" {
		s.Sessions.Revoke(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "scout_session",
		Value:    "",
		Path:     "/api/v1",
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
	// Fall back to query param only for WebSocket upgrade requests.
	if r.Header.Get("Upgrade") == "websocket" {
		if t := r.URL.Query().Get("token"); t != "" {
			return t
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// requireConfig returns the current config (already a deep copy from Store.Current).
// If the config is nil, it writes a 500 error and returns nil.
// Callers must check for nil before proceeding.
func (s *Server) requireConfig(w http.ResponseWriter) *config.Config {
	cfg := s.Config.Current()
	if cfg == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "config not loaded"})
		return nil
	}
	return cfg
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

// redactTokenParam removes the token query parameter from the URL before
// it reaches the logger middleware, preventing session tokens from appearing
// in access logs.
func redactTokenParam(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" && r.URL.Query().Get("token") != "" {
			q := r.URL.Query()
			q.Set("token", "REDACTED")
			r.URL.RawQuery = q.Encode()
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeaders adds standard security response headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' wss:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// checkOrigin rejects cross-origin state-changing requests as CSRF
// defense-in-depth (supplements SameSite=Strict cookies).
func checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check state-changing methods.
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			// No Origin header (e.g. curl, non-browser clients) — allow.
			next.ServeHTTP(w, r)
			return
		}
		// Origin must match the Host header (same-origin).
		// Origin is scheme://host[:port], Host is host[:port].
		expected1 := "https://" + r.Host
		expected2 := "http://" + r.Host
		if origin != expected1 && origin != expected2 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin request blocked"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
