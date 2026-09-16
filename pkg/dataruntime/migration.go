package dataruntime

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a versioned schema migration file pair or block.
type Migration struct {
	Version  int
	Name     string
	UpSQL    string
	DownSQL  string
	UpFile   string
	DownFile string
	Checksum string
}

// MigrationRecord represents an applied migration from the tracking table.
type MigrationRecord struct {
	ID        int64
	Version   int
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// MigrationStatus describes the current state of a migration.
type MigrationStatus struct {
	Version   int
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

// MigrationEngine orchestrates schema migrations and version tracking.
type MigrationEngine struct {
	db     *sql.DB
	driver DBDriver
}

// NewMigrationEngine creates a migration engine for the given database connection.
func NewMigrationEngine(db *sql.DB, driver DBDriver) *MigrationEngine {
	if driver == nil {
		driver = &SQLiteDriver{}
	}
	return &MigrationEngine{
		db:     db,
		driver: driver,
	}
}

// EnsureTable creates the `_pogn_migrations` metadata table if it does not exist.
func (m *MigrationEngine) EnsureTable() error {
	_, err := m.db.Exec(m.driver.MigrationTableSQL())
	return err
}

// DiscoverMigrations scans the `migrations/` directory and parses `.up.sql` and `.down.sql` files.
func (m *MigrationEngine) DiscoverMigrations(dir string) ([]*Migration, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []*Migration{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}

		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		cleanName := parts[1]
		isUp := strings.HasSuffix(cleanName, ".up.sql")
		isDown := strings.HasSuffix(cleanName, ".down.sql")

		if !isUp && !isDown {
			continue
		}

		baseName := strings.TrimSuffix(strings.TrimSuffix(cleanName, ".up.sql"), ".down.sql")
		fullPath := filepath.Join(dir, name)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		mig, exists := migrationMap[ver]
		if !exists {
			mig = &Migration{
				Version: ver,
				Name:    baseName,
			}
			migrationMap[ver] = mig
		}

		if isUp {
			mig.UpSQL = string(content)
			mig.UpFile = name
			mig.Checksum = computeChecksum(string(content))
		} else if isDown {
			mig.DownSQL = string(content)
			mig.DownFile = name
		}
	}

	var list []*Migration
	for _, mig := range migrationMap {
		list = append(list, mig)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})

	return list, nil
}

// GetAppliedMigrations returns all migration records currently logged in the tracking table.
func (m *MigrationEngine) GetAppliedMigrations() ([]*MigrationRecord, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Query("SELECT id, version, name, applied_at, checksum FROM _pogn_migrations ORDER BY version ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*MigrationRecord
	for rows.Next() {
		var rec MigrationRecord
		if err := rows.Scan(&rec.ID, &rec.Version, &rec.Name, &rec.AppliedAt, &rec.Checksum); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}

	return records, rows.Err()
}

// MigrateUp executes all pending migrations in ascending order inside transactions.
func (m *MigrationEngine) MigrateUp(dir string) ([]*Migration, error) {
	return m.Apply(dir)
}

// Apply executes all pending migrations in ascending order inside transactions.
func (m *MigrationEngine) Apply(dir string) ([]*Migration, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[int]*MigrationRecord)
	for _, a := range applied {
		appliedMap[a.Version] = a
	}

	var newlyApplied []*Migration

	for _, mig := range discovered {
		if prev, ok := appliedMap[mig.Version]; ok {
			if prev.Checksum != "" && mig.Checksum != "" && prev.Checksum != mig.Checksum {
				return newlyApplied, fmt.Errorf("checksum mismatch for migration %03d_%s: database has %s, file has %s",
					mig.Version, mig.Name, prev.Checksum, mig.Checksum)
			}
			continue
		}

		tx, err := m.db.Begin()
		if err != nil {
			return newlyApplied, fmt.Errorf("failed to begin migration transaction: %w", err)
		}

		if strings.TrimSpace(mig.UpSQL) != "" {
			if _, err := tx.Exec(mig.UpSQL); err != nil {
				tx.Rollback()
				return newlyApplied, fmt.Errorf("migration %03d_%s failed: %w", mig.Version, mig.Name, err)
			}
		}

		query := "INSERT INTO _pogn_migrations (version, name, checksum) VALUES (?, ?, ?)"
		if _, err := tx.Exec(query, mig.Version, mig.Name, mig.Checksum); err != nil {
			tx.Rollback()
			return newlyApplied, fmt.Errorf("failed to log migration record %d: %w", mig.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return newlyApplied, fmt.Errorf("failed to commit migration transaction %d: %w", mig.Version, err)
		}

		newlyApplied = append(newlyApplied, mig)
	}

	return newlyApplied, nil
}

// Rollback reverts the last batch or specified number of steps of applied migrations.
func (m *MigrationEngine) Rollback(dir string, steps int) ([]*Migration, error) {
	if err := m.EnsureTable(); err != nil {
		return nil, err
	}

	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	discoveredMap := make(map[int]*Migration)
	for _, d := range discovered {
		discoveredMap[d.Version] = d
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	if len(applied) == 0 {
		return nil, nil
	}

	if steps <= 0 || steps > len(applied) {
		steps = len(applied)
	}

	toRollback := applied[len(applied)-steps:]
	var rolledBack []*Migration

	for i := len(toRollback) - 1; i >= 0; i-- {
		rec := toRollback[i]
		mig, ok := discoveredMap[rec.Version]
		if !ok || strings.TrimSpace(mig.DownSQL) == "" {
			return rolledBack, fmt.Errorf("cannot rollback migration %03d_%s: missing .down.sql", rec.Version, rec.Name)
		}

		tx, err := m.db.Begin()
		if err != nil {
			return rolledBack, fmt.Errorf("failed to begin rollback transaction: %w", err)
		}

		if _, err := tx.Exec(mig.DownSQL); err != nil {
			tx.Rollback()
			return rolledBack, fmt.Errorf("rollback %03d_%s failed: %w", rec.Version, rec.Name, err)
		}

		if _, err := tx.Exec("DELETE FROM _pogn_migrations WHERE version = ?", rec.Version); err != nil {
			tx.Rollback()
			return rolledBack, fmt.Errorf("failed to delete migration record %d: %w", rec.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return rolledBack, fmt.Errorf("failed to commit rollback transaction %d: %w", rec.Version, err)
		}

		rolledBack = append(rolledBack, mig)
	}

	return rolledBack, nil
}

// Status returns the current status of all discovered migrations.
func (m *MigrationEngine) Status(dir string) ([]MigrationStatus, error) {
	discovered, err := m.DiscoverMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[int]*MigrationRecord)
	for _, a := range applied {
		appliedMap[a.Version] = a
	}

	var statuses []MigrationStatus
	for _, d := range discovered {
		st := MigrationStatus{
			Version: d.Version,
			Name:    d.Name,
		}
		if a, ok := appliedMap[d.Version]; ok {
			st.Applied = true
			st.AppliedAt = &a.AppliedAt
		}
		statuses = append(statuses, st)
	}

	return statuses, nil
}

// NewMigration creates a new numbered `.up.sql` and `.down.sql` file pair.
func NewMigration(dir, name string) (string, string, error) {
	return GenerateMigration(dir, name)
}

// GenerateMigration creates a new numbered `.up.sql` and `.down.sql` file pair.
func GenerateMigration(dir, name string) (string, string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create migrations directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}

	maxVersion := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) >= 2 {
			if v, err := strconv.Atoi(parts[0]); err == nil && v > maxVersion {
				maxVersion = v
			}
		}
	}

	nextVersion := maxVersion + 1
	cleanName := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	if cleanName == "" {
		cleanName = "migration"
	}

	upFile := filepath.Join(dir, fmt.Sprintf("%03d_%s.up.sql", nextVersion, cleanName))
	downFile := filepath.Join(dir, fmt.Sprintf("%03d_%s.down.sql", nextVersion, cleanName))

	upTemplate := fmt.Sprintf("-- Migration: %s (Up)\n-- Created: %s\n\n", cleanName, time.Now().UTC().Format(time.RFC3339))
	downTemplate := fmt.Sprintf("-- Migration: %s (Down)\n-- Created: %s\n\n", cleanName, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(upFile, []byte(upTemplate), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(downFile, []byte(downTemplate), 0644); err != nil {
		return "", "", err
	}

	return upFile, downFile, nil
}

func computeChecksum(sql string) string {
	h := sha256.New()
	h.Write([]byte(strings.TrimSpace(sql)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
