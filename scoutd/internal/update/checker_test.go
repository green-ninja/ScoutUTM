package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_DevMode(t *testing.T) {
	c := NewChecker("https://update.scoututm.com", "stable", true)
	info, err := c.Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info != nil {
		t.Error("dev-mode check should return nil")
	}
}

func TestChecker_RejectsHTTP(t *testing.T) {
	c := NewChecker("http://update.scoututm.com", "stable", false)
	_, err := c.Check("0.1.0")
	if err == nil {
		t.Fatal("expected error for HTTP URL")
	}
}

func TestChecker_RejectsHTTPBundleURL(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(UpdateInfo{
			Version:   "1.0.0",
			BundleURL: "http://evil.com/bundle.raucb",
		})
	}))
	defer srv.Close()

	c := NewChecker(srv.URL, "stable", false)
	c.client = srv.Client()

	_, err := c.Check("0.1.0")
	if err == nil {
		t.Fatal("expected error for HTTP bundle URL")
	}
}

func TestChecker_NoUpdate(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewChecker(srv.URL, "stable", false)
	c.client = srv.Client()

	info, err := c.Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info != nil {
		t.Error("expected nil when no update available")
	}
}

func TestChecker_UpdateAvailable(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query params.
		if r.URL.Query().Get("version") != "0.1.0" {
			http.Error(w, "bad version", 400)
			return
		}
		if r.URL.Query().Get("channel") != "stable" {
			http.Error(w, "bad channel", 400)
			return
		}
		json.NewEncoder(w).Encode(UpdateInfo{
			Version:     "1.0.0",
			Channel:     "stable",
			Description: "Security fixes",
			BundleURL:   "https://update.scoututm.com/bundles/1.0.0.raucb",
			BundleSHA:   "abc123",
			Size:        50 * 1024 * 1024,
		})
	}))
	defer srv.Close()

	c := NewChecker(srv.URL, "stable", false)
	c.client = srv.Client()

	info, err := c.Check("0.1.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info == nil {
		t.Fatal("expected update info")
	}
	if info.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", info.Version)
	}
	if info.BundleSHA != "abc123" {
		t.Errorf("sha = %q, want abc123", info.BundleSHA)
	}
}

func TestChecker_ServerError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()

	c := NewChecker(srv.URL, "stable", false)
	c.client = srv.Client()

	_, err := c.Check("0.1.0")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
