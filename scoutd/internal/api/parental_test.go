package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scoututm/scoutd/internal/config"
)

func TestParental_CreateProfile(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]interface{}{
		"name":              "Kids",
		"block_categories":  []string{"adult", "gaming"},
		"safe_search":       true,
		"youtube_restricted": true,
	})

	req := httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create profile: status = %d, want 201. Body: %s", w.Code, w.Body.String())
	}

	var profile config.ParentalProfile
	json.Unmarshal(w.Body.Bytes(), &profile)

	if profile.Name != "Kids" {
		t.Errorf("profile name = %q, want Kids", profile.Name)
	}
	if profile.ID == "" {
		t.Error("profile ID is empty")
	}
	if !profile.SafeSearch {
		t.Error("expected safe_search = true")
	}
}

func TestParental_CreateProfile_DuplicateName(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")
	router := srv.Router()

	body, _ := json.Marshal(map[string]interface{}{
		"name":             "Kids",
		"block_categories": []string{},
	})

	// Create first.
	req := httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: %d", w.Code)
	}

	// Create duplicate.
	body, _ = json.Marshal(map[string]interface{}{
		"name":             "Kids",
		"block_categories": []string{},
	})
	req = httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate create: status = %d, want 409", w.Code)
	}
}

func TestParental_DeleteCascade(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")
	router := srv.Router()

	// Create profile.
	body, _ := json.Marshal(map[string]interface{}{
		"name":             "Kids",
		"block_categories": []string{"adult"},
	})
	req := httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var profile config.ParentalProfile
	json.Unmarshal(w.Body.Bytes(), &profile)

	// Assign a device to this profile.
	assignments, _ := json.Marshal([]map[string]string{
		{"mac": "aa:bb:cc:dd:ee:01", "profile_id": profile.ID},
	})
	req = httptest.NewRequest("PUT", "/api/v1/parental/devices", bytes.NewReader(assignments))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("assign devices: status = %d", w.Code)
	}

	// Delete profile — should cascade and unassign device.
	req = httptest.NewRequest("DELETE", "/api/v1/parental/profiles/"+profile.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete profile: status = %d", w.Code)
	}

	// Verify device is unassigned.
	cfg := srv.Config.Current()
	for _, d := range cfg.Parental.Devices {
		if d.ProfileID == profile.ID {
			t.Errorf("device still assigned to deleted profile %s", profile.ID)
		}
	}
}

func TestParental_AssignDevice(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")
	router := srv.Router()

	// Create profile.
	body, _ := json.Marshal(map[string]interface{}{
		"name":             "Kids",
		"block_categories": []string{},
	})
	req := httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var profile config.ParentalProfile
	json.Unmarshal(w.Body.Bytes(), &profile)

	// Assign device.
	assignments, _ := json.Marshal([]map[string]string{
		{"mac": "aa:bb:cc:dd:ee:01", "profile_id": profile.ID},
	})
	req = httptest.NewRequest("PUT", "/api/v1/parental/devices", bytes.NewReader(assignments))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("assign: status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify config.
	cfg := srv.Config.Current()
	if len(cfg.Parental.Devices) != 1 {
		t.Fatalf("expected 1 device assignment, got %d", len(cfg.Parental.Devices))
	}
	if cfg.Parental.Devices[0].ProfileID != profile.ID {
		t.Errorf("device profile = %s, want %s", cfg.Parental.Devices[0].ProfileID, profile.ID)
	}
}

func TestParental_InvalidCategory(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()
	setAdminPassword(t, srv, "testpassword")
	token := loginAndGetToken(t, srv, "testpassword")

	body, _ := json.Marshal(map[string]interface{}{
		"name":             "Bad",
		"block_categories": []string{"nonexistent"},
	})

	req := httptest.NewRequest("POST", "/api/v1/parental/profiles", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid category: status = %d, want 400", w.Code)
	}
}
