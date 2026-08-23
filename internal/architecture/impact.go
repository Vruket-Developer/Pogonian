package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImpactReport details the blast radius of changing a specific file or feature.
type ImpactReport struct {
	Target             string   `json:"target"`
	TargetType         string   `json:"target_type"` // "entity", "service", "handler", "route", "file"
	Feature            string   `json:"feature"`
	AffectedRoutes     []string `json:"affected_routes"`
	AffectedHandlers   []string `json:"affected_handlers"`
	AffectedServices   []string `json:"affected_services"`
	AffectedRepos      []string `json:"affected_repos"`
	AffectedDatabase   []string `json:"affected_database"`
	AffectedAIContext  []string `json:"affected_ai_context"`
	ImpactLevel        string   `json:"impact_level"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	PotentialBreaks    []string `json:"potential_breaks"`
}

// AnalyzeImpact analyzes the blast radius of modifying a given file or entity name.
func AnalyzeImpact(projectDir, target string) (*ImpactReport, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	cleanTarget := strings.TrimPrefix(target, "./")
	cleanTarget = strings.Trim(cleanTarget, "/")

	// Extract feature/module name from path or string
	feature := strings.ToLower(cleanTarget)
	if strings.Contains(cleanTarget, "/") {
		parts := strings.Split(cleanTarget, "/")
		for _, p := range parts {
			if p != "server" && p != "app" && p != "components" && !strings.Contains(p, ".") {
				feature = p
				break
			}
		}
	}

	report := &ImpactReport{
		Target:             cleanTarget,
		Feature:            feature,
		AffectedRoutes:     make([]string, 0),
		AffectedHandlers:   make([]string, 0),
		AffectedServices:   make([]string, 0),
		AffectedRepos:      make([]string, 0),
		AffectedDatabase:   make([]string, 0),
		AffectedAIContext:  make([]string, 0),
		PotentialBreaks:    make([]string, 0),
	}

	// 1. Determine Target Type
	if strings.HasSuffix(cleanTarget, "service.go") {
		report.TargetType = "service"
		report.PotentialBreaks = append(report.PotentialBreaks, "Modifications to service method signatures will break calling HTTP Handlers.")
	} else if strings.HasSuffix(cleanTarget, "entity.go") {
		report.TargetType = "entity"
		report.PotentialBreaks = append(report.PotentialBreaks, "Modifying entity struct fields affects database column mappings, HTTP form payloads, and HTML template renders.")
	} else if strings.HasSuffix(cleanTarget, "repository.go") {
		report.TargetType = "repository"
		report.PotentialBreaks = append(report.PotentialBreaks, "SQL query changes may cause runtime database schema scan errors.")
	} else if strings.HasSuffix(cleanTarget, "handler.go") {
		report.TargetType = "handler"
		report.PotentialBreaks = append(report.PotentialBreaks, "Route handler signature changes affect frontend form submissions and data loading.")
	} else {
		report.TargetType = "feature"
	}

	// 2. Discover Affected Routes in app/<feature>
	appFeatureDir := filepath.Join(absDir, "app", feature)
	if _, err := os.Stat(appFeatureDir); err == nil {
		report.AffectedRoutes = append(report.AffectedRoutes, "/"+feature)
		report.AffectedRoutes = append(report.AffectedRoutes, "/"+feature+"/:id")
	} else {
		report.AffectedRoutes = append(report.AffectedRoutes, "/"+feature)
	}

	// 3. Discover Affected Server Layers
	serverDir := filepath.Join(absDir, "server", feature)
	if _, err := os.Stat(serverDir); err == nil {
		report.AffectedHandlers = append(report.AffectedHandlers, fmt.Sprintf("server/%s/handler.go", feature))
		report.AffectedServices = append(report.AffectedServices, fmt.Sprintf("server/%s/service.go", feature))
		report.AffectedRepos = append(report.AffectedRepos, fmt.Sprintf("server/%s/repository.go", feature))
		report.AffectedDatabase = append(report.AffectedDatabase, fmt.Sprintf("table '%s'", feature))
	}

	// 4. Discover Affected AI Context
	aiPath := filepath.Join(absDir, ".pogonian", "architecture", "app.ai")
	if _, err := os.Stat(aiPath); err == nil {
		report.AffectedAIContext = append(report.AffectedAIContext, ".pogonian/architecture/app.ai")
	}

	// 5. Calculate Impact Score
	totalAffected := len(report.AffectedRoutes) + len(report.AffectedHandlers) +
		len(report.AffectedServices) + len(report.AffectedRepos)

	if totalAffected >= 5 || report.TargetType == "entity" {
		report.ImpactLevel = "HIGH"
	} else if totalAffected >= 3 {
		report.ImpactLevel = "MEDIUM"
	} else {
		report.ImpactLevel = "LOW"
	}

	return report, nil
}
