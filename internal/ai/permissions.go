package ai

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AIPermissions enforces safety constraints on AI operations.
type AIPermissions struct {
	AllowCreateFile      bool
	AllowModifyFile      bool
	AllowDeleteFile      bool
	AllowDatabaseMigrate bool
	RestrictedPaths      []string
	ProjectDir           string
}

// NewDefaultPermissions creates a safe permission boundary for AI modifications.
func NewDefaultPermissions(projectDir string) *AIPermissions {
	return &AIPermissions{
		AllowCreateFile:      true,
		AllowModifyFile:      true,
		AllowDeleteFile:      false, // Deletions denied by default
		AllowDatabaseMigrate: true,
		RestrictedPaths:      []string{".env", "go.mod", "go.sum", ".git"},
		ProjectDir:           projectDir,
	}
}

// CheckFileWrite verifies whether the AI has permission to create or modify a file.
func (p *AIPermissions) CheckFileWrite(targetPath string, isNew bool) error {
	if isNew && !p.AllowCreateFile {
		return fmt.Errorf("AI permission denied: creation of new files is disabled")
	}
	if !isNew && !p.AllowModifyFile {
		return fmt.Errorf("AI permission denied: modification of existing files is disabled")
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	absProject, err := filepath.Abs(p.ProjectDir)
	if err != nil {
		return err
	}

	// Prevent writing outside project workspace
	rel, err := filepath.Rel(absProject, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("AI permission denied: target path '%s' is outside project root", targetPath)
	}

	// Check restricted paths
	for _, restricted := range p.RestrictedPaths {
		if strings.EqualFold(rel, restricted) || strings.HasPrefix(rel, restricted+string(filepath.Separator)) {
			return fmt.Errorf("AI permission denied: path '%s' is protected and restricted from AI modifications", rel)
		}
	}

	return nil
}

// CheckDatabaseMigrate verifies whether the AI has permission to execute schema migrations.
func (p *AIPermissions) CheckDatabaseMigrate() error {
	if !p.AllowDatabaseMigrate {
		return fmt.Errorf("AI permission denied: database schema migration is disabled")
	}
	return nil
}
