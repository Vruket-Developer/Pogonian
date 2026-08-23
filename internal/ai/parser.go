package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AISpec represents the semantic architecture specification found in .ai files.
type AISpec struct {
	Application string            `yaml:"application"`
	Frontend    map[string]string `yaml:"frontend"`
	Backend     map[string]string `yaml:"backend"`
	Flow        string            `yaml:"flow"`
	Modules     []string          `yaml:"modules"`
	Entities    []string          `yaml:"entities,omitempty"`
	Security    map[string]string `yaml:"security,omitempty"`
}

// ParseAISpec reads and parses all .ai metadata files from .pogonian/architecture/.
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

	return &spec, nil
}
