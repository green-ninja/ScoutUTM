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
	"syscall"
	"time"

	"github.com/scoututm/scoutd/internal/api"
	"github.com/scoututm/scoutd/internal/auth"
	"github.com/scoututm/scoutd/internal/config"
	"github.com/scoututm/scoutd/internal/ws"
)

// version is set via ldflags at build time.
var version = "0.1.0-dev"

func main() {
	var (
		configPath string
		devMode    bool
		addr       string
		httpAddr   string
		tlsDir     string
		showVer    bool
	)

	flag.StringVar(&configPath, "config", "/etc/scout/config.json", "path to config file")
	flag.BoolVar(&devMode, "dev-mode", false, "enable development mode (stubs hardware ops)")
	flag.StringVar(&addr, "addr", ":443", "HTTPS listen address")
	flag.StringVar(&httpAddr, "http-addr", ":80", "HTTP redirect listen address")
	flag.StringVar(&tlsDir, "tls-dir", "/etc/scout/tls", "directory for TLS certificate and key")
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
	go hub.Run()

	// API server.
	srv := api.NewServer(store, sessions, hub, devMode, version)
	router := srv.Router()

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
		Addr:      addr,
		Handler:   router,
		TLSConfig: tlsConfig,
	}

	// HTTP → HTTPS redirect.
	httpServer := &http.Server{
		Addr: httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}),
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpServer.Shutdown(ctx)
	httpsServer.Shutdown(ctx)
	log.Println("scoutd stopped")
}

// ensureTLSCert generates a self-signed ECDSA P-256 certificate if none exists.
func ensureTLSCert(dir string) error {
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")

	if _, err := os.Stat(certFile); err == nil {
		return nil // Already exists.
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
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
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour), // ~10 years
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

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
