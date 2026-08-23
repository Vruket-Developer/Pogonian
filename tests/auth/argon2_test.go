package auth_test

import (
	"strings"
	"testing"

	"pogonian/internal/auth"
)

func TestAuth_Argon2idPasswordHashing(t *testing.T) {
	password := "UltraSecureP@ssw0rd!2026"

	// 1. Hash password
	encodedHash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !strings.HasPrefix(encodedHash, "$argon2id$") {
		t.Errorf("Expected hash to begin with '$argon2id$', got: %s", encodedHash)
	}

	// 2. Check valid password
	if !auth.CheckPassword(password, encodedHash) {
		t.Errorf("CheckPassword failed on valid password")
	}

	// 3. Check invalid password
	if auth.CheckPassword("WrongPassword123", encodedHash) {
		t.Errorf("CheckPassword succeeded on incorrect password")
	}

	// 4. Verify unique salt per hash
	secondHash, _ := auth.HashPassword(password)
	if encodedHash == secondHash {
		t.Errorf("Expected different random salts across hashing calls")
	}
}
