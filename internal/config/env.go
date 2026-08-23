package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateSecretKey generates a cryptographically secure 256-bit (32-byte) hex string.
func GenerateSecretKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "pogn_sec_" + hex.EncodeToString(bytes), nil
}

// EnsureEnvFile checks for `.env` in the target directory; if missing, creates it with a freshly generated secret key.
func EnsureEnvFile(projectDir, appName string) (string, error) {
	envPath := filepath.Join(projectDir, ".env")
	examplePath := filepath.Join(projectDir, ".env.example")

	// Create .env.example if missing
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		exampleContent := fmt.Sprintf(`# Pogonian Environment Configuration Template
APP_NAME=%s
APP_ENV=development
APP_KEY=
PORT=3000
HOST=127.0.0.1
DATABASE_URL=sqlite://data.db
SESSION_SECRET=
`, appName)
		_ = os.WriteFile(examplePath, []byte(exampleContent), 0644)
	}

	// If .env already exists, just return
	if _, err := os.Stat(envPath); err == nil {
		return envPath, nil
	}

	appKey, err := GenerateSecretKey()
	if err != nil {
		return "", err
	}

	sessionSecret, _ := GenerateSecretKey()

	envContent := fmt.Sprintf(`# Pogonian Environment Configuration
# Automatically generated on project creation. Do NOT commit this file to Git!

APP_NAME=%s
APP_ENV=development
APP_KEY=%s
SESSION_SECRET=%s

PORT=3000
HOST=127.0.0.1
DATABASE_URL=sqlite://data.db
`, appName, appKey, sessionSecret)

	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write .env: %w", err)
	}

	return envPath, nil
}

// LoadEnv reads a .env file and sets environment variables in os.Setenv if not already set.
func LoadEnv(projectDir string) map[string]string {
	envPath := filepath.Join(projectDir, ".env")
	envMap := make(map[string]string)

	file, err := os.Open(envPath)
	if err != nil {
		return envMap
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Strip quotes if present
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
				(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				val = val[1 : len(val)-1]
			}

			envMap[key] = val
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}

	return envMap
}
