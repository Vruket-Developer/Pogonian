package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SpecVersion defines the standard grammar version for .ai files.
const CurrentSpecVersion = "1.0"

// AISpec represents the formal architecture specification grammar in .ai files.
type AISpec struct {
	Schema      string            `yaml:"schema"`
	Application string            `yaml:"application"`
	Version     string            `yaml:"version,omitempty"`
	Frontend    map[string]string `yaml:"frontend"`
	Backend     map[string]string `yaml:"backend"`
	Flow        string            `yaml:"flow"`
	Modules     []string          `yaml:"modules"`
	Entities    []EntitySpec      `yaml:"entities,omitempty"`
	Security    SecuritySpec      `yaml:"security,omitempty"`
	Permissions PermissionsSpec   `yaml:"permissions,omitempty"`
}

// EntitySpec defines an entity schema within .ai files.
type EntitySpec struct {
	Name   string            `yaml:"name"`
	Fields map[string]string `yaml:"fields"`
}

// SecuritySpec defines security rules in .ai specs.
type SecuritySpec struct {
	CSRF     bool   `yaml:"csrf"`
	Auth     string `yaml:"auth"` // "argon2id", "session"
	RateLimit string `yaml:"rate_limit,omitempty"`
}

// PermissionsSpec defines AI permissions.
type PermissionsSpec struct {
	AllowCreateFile     bool     `yaml:"allow_create_file"`
	AllowModifyFile     bool     `yaml:"allow_modify_file"`
	AllowDeleteFile     bool     `yaml:"allow_delete_file"`
	AllowDatabaseMigrate bool     `yaml:"allow_database_migrate"`
	RestrictedPaths     []string `yaml:"restricted_paths,omitempty"`
}

// ParseAISpec reads and validates all .ai metadata files from .pogonian/architecture/.
func ParseAISpec(projectDir string) (*AISpec, error) {
	aiPath := filepath.Join(projectDir, ".pogonian", "architecture", "app.ai")
	data, err := os.ReadFile(aiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read app.ai spec: %w", err)
	}

	var spec AISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse app.ai YAML: %w", err)
	}

	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("app.ai schema validation failed: %w", err)
	}

	return &spec, nil
}

// Validate checks the AISpec against the formal Pogonian architecture grammar.
func (s *AISpec) Validate() error {
	if s.Application == "" {
		return fmt.Errorf("missing required field: 'application'")
	}
	if s.Flow == "" {
		s.Flow = "Page → Form Action → Handler → Service → Repository → Database"
	}
	if s.Schema == "" {
		s.Schema = CurrentSpecVersion
	}
	return nil
}

// Save writes the updated AISpec back to .pogonian/architecture/app.ai.
func (s *AISpec) Save(projectDir string) error {
	dir := filepath.Join(projectDir, ".pogonian", "architecture")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	aiPath := filepath.Join(dir, "app.ai")
	return os.WriteFile(aiPath, data, 0644)
}

// EnsureDefaultSpec generates a valid default app.ai spec if missing.
func EnsureDefaultSpec(projectDir, appName string) (*AISpec, error) {
	spec, err := ParseAISpec(projectDir)
	if err == nil && spec != nil {
		return spec, nil
	}

	defaultSpec := &AISpec{
		Schema:      CurrentSpecVersion,
		Application: appName,
		Version:     "0.1.0",
		Frontend: map[string]string{
			"type":       "HTML-First",
			"runtime":    "Zero-dependency Soft Navigation",
			"styling":    "Vanilla CSS",
		},
		Backend: map[string]string{
			"language": "Go",
			"database": "SQLite",
			"pattern":  "Layered Modular Monolith",
		},
		Flow: "Page → Form Action → Handler → Service → Repository → Database",
		Modules: []string{"products"},
		Security: SecuritySpec{
			CSRF: true,
			Auth: "argon2id",
		},
		Permissions: PermissionsSpec{
			AllowCreateFile:      true,
			AllowModifyFile:      true,
			AllowDeleteFile:      false,
			AllowDatabaseMigrate: true,
			RestrictedPaths:      []string{".env", "go.mod", ".git"},
		},
	}

	if err := defaultSpec.Save(projectDir); err != nil {
		return nil, err
	}

	return defaultSpec, nil
}

// AddModule registers a new module into the .ai specification.
func (s *AISpec) AddModule(moduleName string) bool {
	modLower := strings.ToLower(moduleName)
	for _, m := range s.Modules {
		if strings.EqualFold(m, modLower) {
			return false // already exists
		}
	}
	s.Modules = append(s.Modules, modLower)
	return true
}
