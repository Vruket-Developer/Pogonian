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
		return "", fmt.Errorf("failed to write .env file: %w", err)
	}

	return envPath, nil
}

// LoadEnv loads cascading .env files in standard priority:
// 1. .env.[APP_ENV].local
// 2. .env.[APP_ENV]
// 3. .env.local
// 4. .env
func LoadEnv(projectDir string) map[string]string {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	files := []string{
		filepath.Join(projectDir, ".env"),
		filepath.Join(projectDir, ".env.local"),
		filepath.Join(projectDir, fmt.Sprintf(".env.%s", appEnv)),
		filepath.Join(projectDir, fmt.Sprintf(".env.%s.local", appEnv)),
	}

	envMap := make(map[string]string)
	for _, file := range files {
		parseAndSetEnvFile(file, envMap)
	}
	return envMap
}

func parseAndSetEnvFile(path string, envMap map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			// Strip quotes if present
			val = strings.Trim(val, `"'`)

			envMap[key] = val

			// Only set if not already present in environment (env vars take precedence over files)
			if _, exists := os.LookupEnv(key); !exists {
				os.Setenv(key, val)
			}
		}
	}
}
