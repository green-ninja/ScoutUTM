package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the password.
// bcrypt silently truncates at 72 bytes, so reject longer passwords
// to avoid a false sense of security.
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		return "", errors.New("password exceeds 72 bytes (bcrypt limit)")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword returns true if password matches the bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
