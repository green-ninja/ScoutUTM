package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"github.com/scoututm/scoutd/internal/api"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/blocklist"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/dnslog"
	"github.com/scoututm/scoutd/internal/ids"
	"github.com/scoututm/scoutd/internal/netmon"
	"github.com/scoututm/scoutd/internal/parental"
	"github.com/scoututm/scoutd/internal/render"
	"github.com/scoututm/scoutd/internal/scheduler"
	"github.com/scoututm/scoutd/internal/service"
	"github.com/scoututm/scoutd/internal/stats"
	"github.com/scoututm/scoutd/internal/ws"
)

// version is set via ldflags at build time.
var version = "0.1.0-dev"

func main() {
	var (
		configPath  string
		devMode     bool
		addr        string
		httpAddr    string
		tlsDir      string
		templateDir string
		outputDir   string
		showVer     bool
	)

	flag.StringVar(&configPath, "config", "/etc/scout/config.json", "path to config file")
	flag.BoolVar(&devMode, "dev-mode", false, "enable development mode (stubs hardware ops)")
	flag.StringVar(&addr, "addr", ":443", "HTTPS listen address")
	flag.StringVar(&httpAddr, "http-addr", ":80", "HTTP redirect listen address")
	flag.StringVar(&tlsDir, "tls-dir", "/etc/scout/tls", "directory for TLS certificate and key")
	flag.StringVar(&templateDir, "template-dir", "/usr/share/scoutd/templates", "directory containing config templates")
	flag.StringVar(&outputDir, "output-dir", "/etc", "directory for rendered config output")
	flag.BoolVar(&showVer, "version", false, "print version and exit")
	flag.Parse()

	if showVer {
		fmt.Println(version)
		os.Exit(0)
	}

	// Dev-mode overrides.
	if devMode {
		if !isFlagSet("addr") {
			addr = ":8443"
		}
		if !isFlagSet("http-addr") {
			httpAddr = ":8080"
		}
		if !isFlagSet("config") {
			configPath = "./config.json"
		}
		if !isFlagSet("tls-dir") {
			tlsDir = "./tls"
		}
		if !isFlagSet("template-dir") {
			templateDir = "./templates"
		}
		if !isFlagSet("output-dir") {
			outputDir = "./output"
		}
		log.Println("[dev-mode] enabled")
	}

	log.Printf("scoutd %s starting", version)

	// Config store.
	store := config.NewStore(configPath)

	if err := store.RecoverIfNeeded(); err != nil {
		log.Printf("warning: config recovery: %v", err)
	}

	if err := store.Load(devMode); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := store.MigrateIfNeeded(); err != nil {
		log.Printf("warning: config migration: %v", err)
	}

	if errs := config.Validate(store.Current()); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("config warning: %s", e)
		}
	}

	// Auth.
	sessions := auth.NewSessionManager()

	// WebSocket hub.
	hub := ws.NewHub()

	// Stats collector + broadcast goroutine.
	wanIface := wanInterfaceName(store.Current())
	collector := stats.NewCollector(wanIface)
	statsCtx, statsCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(hub.StatsInterval())
		defer ticker.Stop()
		for {
			select {
			case <-statsCtx.Done():
				return
			case <-ticker.C:
				if hub.ClientCount() == 0 {
					continue
				}
				snap, err := collector.Collect()
				if err != nil {
					log.Printf("stats collect: %v", err)
					continue
				}
				msg, err := json.Marshal(map[string]interface{}{
					"type": "stats",
					"data": snap,
				})
				if err != nil {
					log.Printf("stats marshal: %v", err)
					continue
				}
				hub.Broadcast(msg)
				// Adjust tick rate if client count changed.
				ticker.Reset(hub.StatsInterval())
			}
		}
	}()

	// Fetch threats blocklist on startup if enabled.
	if store.Current().DNS.BlockThreats {
		n, err := blocklist.FetchThreatsBlocklist("")
		if err != nil {
			log.Printf("warning: threats blocklist fetch: %v", err)
		} else {
			log.Printf("threats blocklist: loaded %d domains", n)
		}
	}

	// Template rendering engine.
	engine := render.NewEngine(templateDir, outputDir, devMode)
	if err := engine.RenderAll(store.Current()); err != nil {
		log.Printf("warning: initial template render: %v", err)
	}

	// Service manager. In dev mode (Vagrant VM), only nftables reloads are
	// stubbed (SSH safety). All other daemons reload normally.
	var svcMgr *service.Manager
	if devMode {
		svcMgr = service.NewVMManager()
	} else {
		svcMgr = service.NewManager(false)
	}

	// EVE alert reader.
	eveReader := ids.NewReader(
		"/var/run/suricata/eve.sock",
		"/var/lib/scout/suricata/eve.json",
	)
	// Bridge EVE alerts → WebSocket hub.
	eveReader.OnAlert = func(alert ids.Alert) {
		hub.BroadcastAlert(alert)
	}
	eveCtx, eveCancel := context.WithCancel(context.Background())
	go eveReader.Run(eveCtx)

	// Scheduler.
	eveLogPath := "/var/lib/scout/suricata/eve.json"
	sched := scheduler.New()
	sched.Register(scheduler.Task{
		Name:  "suricata-update",
		RunAt: "03:00",
		Fn: func(ctx context.Context) error {
			if err := svcMgr.SuricataUpdate(store.Current().IDS.Rulesets); err != nil {
				return err
			}
			return svcMgr.ReloadDaemons([]string{"suricata"})
		},
	})
	sched.Register(scheduler.Task{
		Name:     "eve-log-rotate",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			return ids.RotateEVELog(eveLogPath, 50*1024*1024, 7)
		},
	})
	schedCtx, schedCancel := context.WithCancel(context.Background())
	go sched.Start(schedCtx)

	// Network device scanner.
	scanner := netmon.NewScanner("/var/lib/misc/dnsmasq.leases")
	if err := scanner.Scan(); err != nil {
		log.Printf("warning: initial device scan: %v", err)
	}
	scanCtx, scanCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		var prevDevices []netmon.Device
		for {
			select {
			case <-scanCtx.Done():
				return
			case <-ticker.C:
				if err := scanner.Scan(); err != nil {
					log.Printf("device scan: %v", err)
					continue
				}
				devices := scanner.Devices()
				if !devicesEqual(prevDevices, devices) {
					msg, err := json.Marshal(map[string]interface{}{
						"type": "devices",
						"data": devices,
					})
					if err != nil {
						log.Printf("devices marshal: %v", err)
						continue
					}
					hub.Broadcast(msg)
					prevDevices = devices
				}
			}
		}
	}()

	// DNS query logger.
	dnsLogDir := dnslog.DefaultLogDir
	dnsLogger := dnslog.NewLogger(dnsLogDir, scanner)
	dnsLogDone := make(chan struct{})
	go dnsLogger.TailFile(filepath.Join(dnsLogDir, dnslog.DefaultLogFile), dnsLogDone)

	// API server.
	srv := api.NewServer(store, sessions, hub, engine, svcMgr, eveReader, scanner, devMode, version)
	srv.DNSLogger = dnsLogger
	router := srv.Router()

	// Register cleanup tasks (after srv is created so we can access RateLimiter).
	sched.Register(scheduler.Task{
		Name:     "session-cleanup",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			sessions.Cleanup()
			srv.RateLimiter.Cleanup()
			return nil
		},
	})

	// DNS log rotation (200MB max, 7 days retention).
	sched.Register(scheduler.Task{
		Name:     "dns-log-rotate",
		Interval: 1 * time.Hour,
		Fn: func(ctx context.Context) error {
			return dnslog.RotateLog(dnsLogDir, 200*1024*1024, 7)
		},
	})

	// Daily threats blocklist refresh (also serves as retry if startup fetch failed).
	sched.Register(scheduler.Task{
		Name:  "threats-blocklist-refresh",
		RunAt: "04:00",
		Fn: func(ctx context.Context) error {
			cfg := store.Current()
			if !cfg.DNS.BlockThreats {
				return nil
			}
			n, err := blocklist.FetchThreatsBlocklist("")
			if err != nil {
				return fmt.Errorf("threats blocklist fetch: %w", err)
			}
			log.Printf("threats blocklist refresh: loaded %d domains", n)
			if err := engine.RenderUnbound(cfg); err != nil {
				return fmt.Errorf("rendering unbound after blocklist refresh: %w", err)
			}
			return svcMgr.ReloadDaemons([]string{"unbound"})
		},
	})

	// Parental schedule enforcement — check every 60s and re-render Unbound if blocked set changes.
	var prevBlockedIPs []string
	sched.Register(scheduler.Task{
		Name:     "parental-schedule-check",
		Interval: 60 * time.Second,
		Fn: func(ctx context.Context) error {
			cfg := store.Current()
			blocked := parental.BlockedDeviceIPs(cfg, time.Now())

			// Skip if unchanged.
			if slicesEqual(prevBlockedIPs, blocked) {
				return nil
			}
			prevBlockedIPs = blocked

			if err := engine.RenderUnbound(cfg); err != nil {
				return fmt.Errorf("rendering unbound with schedule: %w", err)
			}
			return svcMgr.ReloadDaemons([]string{"unbound"})
		},
	})

	// TLS.
	if err := ensureTLSCert(tlsDir); err != nil {
		log.Fatalf("TLS setup: %v", err)
	}

	certFile := filepath.Join(tlsDir, "server.crt")
	keyFile := filepath.Join(tlsDir, "server.key")

	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("loading TLS cert: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	httpsServer := &http.Server{
		Addr:              addr,
		Handler:           router,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// HTTP → HTTPS redirect.
	httpsPort := addr // e.g. ":443" or ":8443"
	httpServer := &http.Server{
		Addr: httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			// Strip port from Host header, replace with HTTPS port.
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if httpsPort != ":443" {
				host = net.JoinHostPort(host, strings.TrimPrefix(httpsPort, ":"))
			}
			target := "https://" + host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		log.Printf("HTTP redirect on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	go func() {
		log.Printf("HTTPS on %s", addr)
		if err := httpsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received %v, shutting down", sig)

	statsCancel()
	eveCancel()
	schedCancel()
	scanCancel()
	close(dnsLogDone)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
	if err := httpsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTPS shutdown: %v", err)
	}
	log.Println("scoutd stopped")
}

// ensureTLSCert generates a self-signed ECDSA P-256 certificate if none exists.
func ensureTLSCert(dir string) error {
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	if _, err := os.Stat(certFile); err == nil {
		return nil // Already exists.
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating TLS dir: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "scout.local"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(2 * 365 * 24 * time.Hour), // ~2 years
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"scout.local", "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.1.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return fmt.Errorf("writing cert: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshaling key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("writing key: %w", err)
	}

	log.Printf("generated self-signed TLS cert in %s", dir)
	return nil
}

func wanInterfaceName(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	for _, iface := range cfg.Network.Interfaces {
		if iface.Zone == "wan" {
			return iface.Name
		}
	}
	return ""
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// slicesEqual compares two string slices for equality (order-sensitive).
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// devicesEqual compares two device slices for change detection.
func devicesEqual(a, b []netmon.Device) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]netmon.Device, len(a))
	for _, d := range a {
		m[d.MAC] = d
	}
	for _, d := range b {
		prev, ok := m[d.MAC]
		if !ok {
			return false
		}
		if prev.IP != d.IP || prev.Online != d.Online || prev.Hostname != d.Hostname || prev.Name != d.Name || prev.ProfileID != d.ProfileID || prev.Zone != d.Zone || prev.Manufacturer != d.Manufacturer {
			return false
		}
	}
	return true
}
