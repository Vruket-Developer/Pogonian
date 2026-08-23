package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/security"
)

// RunSecurity executes `pogn security audit [path]`.
func RunSecurity(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if arg != "audit" && !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	report, err := security.AuditProject(absDir)
	if err != nil {
		return fmt.Errorf("security audit failed: %w", err)
	}

	fmt.Print(report.Summary())

	if report.Score < 70 {
		return fmt.Errorf("security audit failed score threshold (Score: %d/100)", report.Score)
	}

	return nil
}
