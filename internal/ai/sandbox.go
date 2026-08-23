package ai

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pogonian/internal/architecture"
)

// Snapshot represents a pre-change backup of the project workspace.
type Snapshot struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Task      string    `json:"task"`
	Dir       string    `json:"dir"`
	Files     []string  `json:"files"`
}

// SandboxEngine manages pre-commit snapshots, shadow verification, and rollbacks.
type SandboxEngine struct {
	ProjectDir string
	SnapshotDir string
}

// NewSandboxEngine creates a new SandboxEngine.
func NewSandboxEngine(projectDir string) *SandboxEngine {
	return &SandboxEngine{
		ProjectDir:  projectDir,
		SnapshotDir: filepath.Join(projectDir, ".pogonian", "snapshots"),
	}
}

// CreateSnapshot captures a snapshot of code files prior to applying an AI plan.
func (s *SandboxEngine) CreateSnapshot(taskDescription string) (*Snapshot, error) {
	if err := os.MkdirAll(s.SnapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshots dir: %w", err)
	}

	snapID := fmt.Sprintf("%d", time.Now().UnixNano())
	targetDir := filepath.Join(s.SnapshotDir, snapID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, err
	}

	var snapFiles []string

	// Backup server/, app/, migrations/, .pogonian/architecture/
	for _, folder := range []string{"server", "app", "migrations", ".pogonian"} {
		srcFolder := filepath.Join(s.ProjectDir, folder)
		if _, err := os.Stat(srcFolder); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(srcFolder, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			// Don't backup snapshots recursively
			if strings.Contains(path, "snapshots") || strings.Contains(path, "build") {
				return nil
			}

			rel, _ := filepath.Rel(s.ProjectDir, path)
			dstPath := filepath.Join(targetDir, rel)
			_ = os.MkdirAll(filepath.Dir(dstPath), 0755)

			if copyFile(path, dstPath) == nil {
				snapFiles = append(snapFiles, rel)
			}
			return nil
		})
	}

	// Write task metadata
	metaPath := filepath.Join(targetDir, "snapshot.meta")
	metaContent := fmt.Sprintf("TASK: %s\nTIMESTAMP: %s\nFILES: %d\n", taskDescription, time.Now().Format(time.RFC3339), len(snapFiles))
	_ = os.WriteFile(metaPath, []byte(metaContent), 0644)

	return &Snapshot{
		ID:        snapID,
		Timestamp: time.Now(),
		Task:      taskDescription,
		Dir:       targetDir,
		Files:     snapFiles,
	}, nil
}

// ValidateInSandbox verifies architecture boundaries and integrity after changes.
func (s *SandboxEngine) ValidateInSandbox() error {
	analyzer, err := architecture.NewAnalyzer(s.ProjectDir)
	if err != nil {
		return fmt.Errorf("sandbox analyzer failed: %w", err)
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("sandbox IR generation failed: %w", err)
	}

	validator := architecture.NewValidator(true)
	violations := validator.Validate(appIR, graph)

	for _, v := range violations {
		if v.Severity == "ERROR" {
			return fmt.Errorf("sandbox contract violation: [%s] %s: %s", v.Layer, v.Component, v.Message)
		}
	}

	return nil
}

// Rollback restores the most recent snapshot or a specific snapshot by ID.
func (s *SandboxEngine) Rollback(snapshotID string) (*Snapshot, error) {
	if _, err := os.Stat(s.SnapshotDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no snapshots found in .pogonian/snapshots")
	}

	targetSnapDir := ""
	if snapshotID != "" {
		targetSnapDir = filepath.Join(s.SnapshotDir, snapshotID)
		if _, err := os.Stat(targetSnapDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot '%s' does not exist", snapshotID)
		}
	} else {
		// Find latest
		entries, err := os.ReadDir(s.SnapshotDir)
		if err != nil || len(entries) == 0 {
			return nil, fmt.Errorf("no snapshots available to rollback")
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() > entries[j].Name() // newest first
		})

		targetSnapDir = filepath.Join(s.SnapshotDir, entries[0].Name())
		snapshotID = entries[0].Name()
	}

	// Restore files from snapshot
	var restored []string
	_ = filepath.WalkDir(targetSnapDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == "snapshot.meta" {
			return nil
		}

		rel, _ := filepath.Rel(targetSnapDir, path)
		dstPath := filepath.Join(s.ProjectDir, rel)
		_ = os.MkdirAll(filepath.Dir(dstPath), 0755)

		if copyFile(path, dstPath) == nil {
			restored = append(restored, rel)
		}
		return nil
	})

	return &Snapshot{
		ID:        snapshotID,
		Timestamp: time.Now(),
		Dir:       targetSnapDir,
		Files:     restored,
	}, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
