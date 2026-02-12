package auth

import "testing"

func TestHashAndCheck(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if !CheckPassword("correct-horse-battery", hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashUniqueness(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (different salts)")
	}
}
