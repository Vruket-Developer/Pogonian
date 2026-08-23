package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the pogonian.yaml project configuration.
type Config struct {
	Name         string             `yaml:"name"`
	Version      string             `yaml:"version"`
	Framework    string             `yaml:"framework"`
	Server       ServerConfig       `yaml:"server"`
	Architecture ArchitectureConfig `yaml:"architecture"`
	Database     DatabaseConfig     `yaml:"database"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type ArchitectureConfig struct {
	Style  string `yaml:"style"`
	Strict bool   `yaml:"strict"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	Path   string `yaml:"path"`
	URL    string `yaml:"url,omitempty"`
}

// DefaultConfig returns a sensible default configuration for Pogonian apps.
func DefaultConfig(name string) *Config {
	return &Config{
		Name:      name,
		Version:   "0.1.0",
		Framework: "pogonian",
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 3000,
		},
		Architecture: ArchitectureConfig{
			Style:  "modular-monolith",
			Strict: false,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   "./data.db",
		},
	}
}

// Load loads the configuration from pogonian.yaml in the specified directory, and loads .env if present.
func Load(projectDir string) (*Config, error) {
	// Auto-load .env file if present
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
