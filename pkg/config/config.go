package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"pogonian/internal/version"
)

// Config represents the pogonian.yaml project configuration.
type Config struct {
	Name         string             `yaml:"name"`
	Version      string             `yaml:"version"`
	Framework    string             `yaml:"framework"`
	Environment  string             `yaml:"environment,omitempty"` // "development", "test", "production"
	Server       ServerConfig       `yaml:"server"`
	Architecture ArchitectureConfig `yaml:"architecture"`
	Database     DatabaseConfig     `yaml:"database"`
	Native       NativeConfig       `yaml:"native,omitempty"`
}

type NativeConfig struct {
	PackageID string `yaml:"package_id,omitempty"`
	Org       string `yaml:"org,omitempty"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ArchitectureConfig struct {
	Style       string `yaml:"style"`
	Enforcement string `yaml:"enforcement,omitempty"` // "warn" | "strict" | "off"
	Strict      bool   `yaml:"strict,omitempty"`      // backwards compatibility
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	Path   string `yaml:"path"`
	URL    string `yaml:"url,omitempty"`
}

// DefaultConfig returns a sensible default configuration for Pogonian apps.
func DefaultConfig(name string) *Config {
	return &Config{
		Name:        name,
		Version:     version.Current,
		Framework:   "pogonian",
		Native: NativeConfig{
			Org: "com.pogonian",
		},
		Environment: "development",
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 3000,
		},
		Architecture: ArchitectureConfig{
			Style:       "modular-monolith",
			Enforcement: "warn",
			Strict:      false,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   "./data.db",
		},
	}
}

// Load loads the configuration from pogonian.yaml in the specified directory, and loads cascading .env files.
func Load(projectDir string) (*Config, error) {
	LoadEnv(projectDir)

	configPath := filepath.Join(projectDir, "pogonian.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pogonian.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse pogonian.yaml: %w", err)
	}

	// Environment variable overrides
	if appEnv := os.Getenv("APP_ENV"); appEnv != "" {
		cfg.Environment = appEnv
	} else if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	if portStr := os.Getenv("PORT"); portStr != "" {
		var p int
		if _, err := fmt.Sscanf(portStr, "%d", &p); err == nil && p > 0 {
			cfg.Server.Port = p
		}
	}
	if host := os.Getenv("HOST"); host != "" {
		cfg.Server.Host = host
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3000
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}

	// Standardize architecture enforcement semantics
	if cfg.Architecture.Enforcement == "" {
		if cfg.Architecture.Strict {
			cfg.Architecture.Enforcement = "strict"
		} else {
			cfg.Architecture.Enforcement = "warn"
		}
	}

	return &cfg, nil
}

// Save writes the configuration to pogonian.yaml in the specified directory.
func Save(projectDir string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	configPath := filepath.Join(projectDir, "pogonian.yaml")
	return os.WriteFile(configPath, data, 0644)
}

// GetPackageID derives a valid reverse-DNS package identifier for native application targets.
func (c *Config) GetPackageID() string {
	if c.Native.PackageID != "" {
		return c.Native.PackageID
	}
	org := c.Native.Org
	if org == "" {
		org = "com.pogonian"
	}
	cleanName := strings.ToLower(c.Name)
	var sb strings.Builder
	for _, ch := range cleanName {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			sb.WriteRune(ch)
		} else if ch == '-' || ch == '_' {
			sb.WriteRune('_')
		}
	}
	pkgName := sb.String()
	if pkgName == "" {
		pkgName = "app"
	}
	return fmt.Sprintf("%s.%s", org, pkgName)
}

