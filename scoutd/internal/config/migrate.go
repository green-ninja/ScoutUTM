package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func init() {
	migrations[1] = migrateV1toV2
}

// migrateV1toV2 merges block_malware, block_phishing, and block_ads into
// a single block_threats toggle. Also removes use_doh (never implemented).
func migrateV1toV2(data []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var dns map[string]json.RawMessage
	if d, ok := raw["dns"]; ok {
		if err := json.Unmarshal(d, &dns); err != nil {
			return nil, err
		}
	} else {
		dns = make(map[string]json.RawMessage)
	}

	// Merge: block_threats = block_malware || block_phishing || block_ads (default true).
	threats := true
	for _, key := range []string{"block_malware", "block_phishing", "block_ads"} {
		if val, ok := dns[key]; ok {
			var v bool
			if err := json.Unmarshal(val, &v); err == nil && v {
				threats = true
			}
			delete(dns, key)
		}
	}

	bt, _ := json.Marshal(threats)
	dns["block_threats"] = bt

	// Remove use_doh (never implemented, Unbound doesn't support DoH natively).
	delete(dns, "use_doh")

	dnsBytes, _ := json.Marshal(dns)
	raw["dns"] = dnsBytes

	// Bump version.
	raw["version"] = json.RawMessage("2")

	return json.MarshalIndent(raw, "", "  ")
}

// MigrateFunc transforms a config from version N to N+1.
type MigrateFunc func(data []byte) ([]byte, error)

// migrations maps source version → migration function.
// Empty for v1 since there's nothing to migrate from.
var migrations = map[int]MigrateFunc{}

// MigrateIfNeeded checks the config version and runs sequential migrations
// if the config is older than CurrentSchemaVersion. If the config is newer
// (downgrade/rollback scenario), it returns an error so the caller can
// restore the pre-migrate backup.
func (s *Store) MigrateIfNeeded() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return fmt.Errorf("no config loaded")
	}

	if s.current.Version == CurrentSchemaVersion {
		return nil // Nothing to do.
	}

	if s.current.Version > CurrentSchemaVersion {
		return fmt.Errorf("config version %d is newer than supported %d (rollback needed)", s.current.Version, CurrentSchemaVersion)
	}

	// Save a pre-migrate backup.
	if err := s.writeConfigLocked(s.preMigratePath, s.current); err != nil {
		return fmt.Errorf("saving pre-migrate backup: %w", err)
	}

	// Read raw JSON from disk so old fields (removed from struct) are preserved
	// for migration functions.
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("reading config for migration: %w", err)
	}

	for v := s.current.Version; v < CurrentSchemaVersion; v++ {
		fn, ok := migrations[v]
		if !ok {
			return fmt.Errorf("no migration from version %d to %d", v, v+1)
		}
		data, err = fn(data)
		if err != nil {
			return fmt.Errorf("migration v%d→v%d failed: %w", v, v+1, err)
		}
	}

	var migrated Config
	if err := json.Unmarshal(data, &migrated); err != nil {
		return fmt.Errorf("parsing migrated config: %w", err)
	}

	s.current = &migrated
	return s.writeConfigLocked(s.configPath, s.current)
}

// RestorePreMigrate restores the pre-migration backup.
func (s *Store) RestorePreMigrate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.preMigratePath)
	if err != nil {
		return fmt.Errorf("reading pre-migrate backup: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing pre-migrate backup: %w", err)
	}

	if err := atomicWriteFile(s.configPath, &cfg); err != nil {
		return fmt.Errorf("restoring config: %w", err)
	}

	s.current = &cfg
	return nil
}
