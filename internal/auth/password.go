package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

// HashPassword creates a salted SHA-256 hash formatted as salt$hash.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	saltHex := hex.EncodeToString(salt)
	hash := sha256.Sum256([]byte(saltHex + password))
	hashHex := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%s$%s", saltHex, hashHex), nil
}

// CheckPasswordHash verifies a plain password against a stored salt$hash string.
func CheckPasswordHash(password, storedHash string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 2 {
		return false
	}

	saltHex := parts[0]
	expectedHashHex := parts[1]

	actualHash := sha256.Sum256([]byte(saltHex + password))
	actualHashHex := hex.EncodeToString(actualHash[:])

	return subtle.ConstantTimeCompare([]byte(actualHashHex), []byte(expectedHashHex)) == 1
}
