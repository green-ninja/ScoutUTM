package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func resetWizardState() {
	wizardMu.Lock()
	wizardComplete = make(map[int]bool)
	wizardMu.Unlock()
}

func TestWizard_Step1_Welcome(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	// Reset setup_done to false.
	cfg := srv.Config.Current()
	cfg.System.SetupDone = false
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("step 1: status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

func TestWizard_Step2_Password(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	cfg := srv.Config.Current()
	cfg.System.SetupDone = false
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	router := srv.Router()

	// Step 1 first.
	req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Step 2: short password should fail.
	body, _ := json.Marshal(map[string]string{"password": "short"})
	req = httptest.NewRequest("POST", "/api/v1/wizard/step/2", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("step 2 short password: status = %d, want 400", w.Code)
	}

	// Step 2: valid password.
	body, _ = json.Marshal(map[string]string{"password": "newpassword123"})
	req = httptest.NewRequest("POST", "/api/v1/wizard/step/2", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("step 2 valid password: status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

func TestWizard_BlockedAfterSetup(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	// Set setup_done = true.
	cfg := srv.Config.Current()
	cfg.System.SetupDone = true
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("wizard after setup: status = %d, want 403", w.Code)
	}
}

func TestWizard_StepOrdering(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	cfg := srv.Config.Current()
	cfg.System.SetupDone = false
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	// Try step 2 without completing step 1.
	body, _ := json.Marshal(map[string]string{"password": "validpassword1"})
	req := httptest.NewRequest("POST", "/api/v1/wizard/step/2", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("step 2 without step 1: status = %d, want 400", w.Code)
	}
}

func TestWizard_FullFlow(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	cfg := srv.Config.Current()
	cfg.System.SetupDone = false
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	router := srv.Router()
	doStep := func(step int, bodyMap interface{}) int {
		var bodyReader *bytes.Reader
		if bodyMap != nil {
			b, _ := json.Marshal(bodyMap)
			bodyReader = bytes.NewReader(b)
		} else {
			bodyReader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest("POST", "/api/v1/wizard/step/"+itoa(step), bodyReader)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	// Step 1: Welcome.
	if code := doStep(1, nil); code != 200 {
		t.Fatalf("step 1: %d", code)
	}

	// Step 2: Password.
	if code := doStep(2, map[string]string{"password": "securepassword1"}); code != 200 {
		t.Fatalf("step 2: %d", code)
	}

	// Step 3: Network detection.
	if code := doStep(3, nil); code != 200 {
		t.Fatalf("step 3: %d", code)
	}

	// Step 4: LAN config.
	if code := doStep(4, map[string]string{"lan_address": "192.168.1.1/24"}); code != 200 {
		t.Fatalf("step 4: %d", code)
	}

	// Step 5: Security defaults.
	if code := doStep(5, map[string]interface{}{}); code != 200 {
		t.Fatalf("step 5: %d", code)
	}

	// Step 6: VLAN (skip).
	if code := doStep(6, map[string]interface{}{"vlans": []interface{}{}, "zones": []interface{}{}}); code != 200 {
		t.Fatalf("step 6: %d", code)
	}

	// Step 7: Apply.
	if code := doStep(7, nil); code != 200 {
		t.Fatalf("step 7: %d", code)
	}

	// Verify setup_done is now true.
	finalCfg := srv.Config.Current()
	if !finalCfg.System.SetupDone {
		t.Error("expected setup_done = true after wizard completion")
	}

	// Verify wizard is blocked now.
	if code := doStep(1, nil); code != 403 {
		t.Errorf("post-setup wizard: %d, want 403", code)
	}
}

func itoa(n int) string {
	return string(rune('0' + n))
}

// Ensure concurrent wizard access doesn't panic.
func TestWizard_ConcurrentAccess(t *testing.T) {
	resetWizardState()
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	cfg := srv.Config.Current()
	cfg.System.SetupDone = false
	srv.Config.SetCurrent(cfg)
	srv.Config.Save()

	router := srv.Router()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/v1/wizard/step/1", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()
	}
	wg.Wait()
}
