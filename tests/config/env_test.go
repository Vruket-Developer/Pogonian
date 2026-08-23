package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/internal/config"
)

func TestConfig_EnvFileAndKeyGeneration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn-env-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Test GenerateSecretKey
	key, err := config.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey failed: %v", err)
	}
	if !strings.HasPrefix(key, "pogn_sec_") || len(key) < 40 {
		t.Errorf("Unexpected secret key format: %s", key)
	}

	// 2. Test EnsureEnvFile
	envPath, err := config.EnsureEnvFile(tempDir, "test-app")
	if err != nil {
		t.Fatalf("EnsureEnvFile failed: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read created .env: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "APP_NAME=test-app") {
		t.Errorf("Expected APP_NAME=test-app in .env, got:\n%s", content)
	}
	if !strings.Contains(content, "APP_KEY=pogn_sec_") {
		t.Errorf("Expected APP_KEY=pogn_sec_ in .env, got:\n%s", content)
	}
	if !strings.Contains(content, "SESSION_SECRET=pogn_sec_") {
		t.Errorf("Expected SESSION_SECRET=pogn_sec_ in .env, got:\n%s", content)
	}

	// 3. Test .env.example created
	examplePath := filepath.Join(tempDir, ".env.example")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Errorf("Expected .env.example to be created")
	}

	// 4. Test LoadEnv
	envMap := config.LoadEnv(tempDir)
	if envMap["APP_NAME"] != "test-app" {
		t.Errorf("Expected LoadEnv APP_NAME='test-app', got '%s'", envMap["APP_NAME"])
	}
	if !strings.HasPrefix(envMap["APP_KEY"], "pogn_sec_") {
		t.Errorf("Expected LoadEnv APP_KEY with prefix 'pogn_sec_', got '%s'", envMap["APP_KEY"])
	}
}
