package ai

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/architecture"
	"pogonian/pkg/ir"
)

// AIContext contains the minimal, token-efficient architectural context for an AI prompt.
type AIContext struct {
	Prompt           string
	RelevantFiles    []string
	RelevantEntities []string
	RelevantRoutes   []string
	ArchitectureFlow string
	ContextSummary   string
	FullRepoTokens   int
	ScopedTokens     int
	SavingsPercent   float64
}

// ContextEngine extracts token-efficient architecture context for AI agents.
type ContextEngine struct {
	ProjectDir   string
	AppIR        *ir.Application
	Graph        *architecture.DependencyGraph
	TokenCounter *TokenCounter
}

// NewContextEngine creates a new AI Context Engine with BPE Tokenizer.
func NewContextEngine(projectDir string, appIR *ir.Application, graph *architecture.DependencyGraph) *ContextEngine {
	return &ContextEngine{
		ProjectDir:   projectDir,
		AppIR:        appIR,
		Graph:        graph,
		TokenCounter: NewTokenCounter(),
	}
}

// ExtractContext extracts only the relevant architectural subgraphs and files for a prompt.
func (e *ContextEngine) ExtractContext(prompt string) (*AIContext, error) {
	promptLower := strings.ToLower(prompt)
	words := strings.Fields(promptLower)

	relevantFiles := make([]string, 0)
	relevantEntities := make([]string, 0)
	relevantRoutes := make([]string, 0)

	// 1. Identify relevant entities & modules from prompt using semantic symbol matching
	matchedModules := make(map[string]bool)
	for _, word := range words {
		cleaned := strings.Trim(word, `.,!?"'()[]`)
		if len(cleaned) <= 2 {
			continue
		}

		// Match entities in IR
		for _, ent := range e.AppIR.Entities {
			if strings.Contains(strings.ToLower(ent.Name), cleaned) || strings.Contains(cleaned, strings.ToLower(ent.Name)) {
				relevantEntities = append(relevantEntities, ent.Name)
				matchedModules[strings.ToLower(ent.Name)] = true
			}
		}

		// Match routes in IR
		for _, rt := range e.AppIR.Routes {
			if strings.Contains(strings.ToLower(rt.Path), cleaned) {
				relevantRoutes = append(relevantRoutes, rt.Path)
				if rt.PagePath != "" {
					relevantFiles = append(relevantFiles, rt.PagePath)
				}
			}
		}
	}

	// 2. Identify relevant backend layers (Handlers, Services, Repositories)
	for _, h := range e.AppIR.Handlers {
		for mod := range matchedModules {
			if strings.Contains(strings.ToLower(h.Name), mod) || strings.Contains(strings.ToLower(h.Package), mod) {
				if h.File != "" {
					relevantFiles = append(relevantFiles, filepath.Join(e.ProjectDir, h.File))
				}
			}
		}
	}

	for _, s := range e.AppIR.Services {
		for mod := range matchedModules {
			if strings.Contains(strings.ToLower(s.Name), mod) || strings.Contains(strings.ToLower(s.Package), mod) {
				if s.File != "" {
					relevantFiles = append(relevantFiles, filepath.Join(e.ProjectDir, s.File))
				}
			}
		}
	}

	for _, r := range e.AppIR.Repositories {
		for mod := range matchedModules {
			if strings.Contains(strings.ToLower(r.Name), mod) || strings.Contains(strings.ToLower(r.Package), mod) {
				if r.File != "" {
					relevantFiles = append(relevantFiles, filepath.Join(e.ProjectDir, r.File))
				}
			}
		}
	}

	// Always include .ai architecture spec as minimal semantic ground truth
	aiFile := filepath.Join(e.ProjectDir, ".pogonian", "architecture", "app.ai")
	if _, err := os.Stat(aiFile); err == nil {
		relevantFiles = append(relevantFiles, aiFile)
	}

	relevantFiles = uniqueStrings(relevantFiles)
	relevantEntities = uniqueStrings(relevantEntities)
	relevantRoutes = uniqueStrings(relevantRoutes)

	// 3. Compute Real Token Counts using BPE Tokenizer
	allFiles := e.discoverAllCodeFiles()
	fullTokens := e.TokenCounter.CountRepositoryTokens(allFiles)
	scopedTokens := e.TokenCounter.CountRepositoryTokens(relevantFiles)
	if scopedTokens == 0 {
		scopedTokens = 250 // minimal schema baseline
	}

	savings := 0.0
	if fullTokens > scopedTokens {
		savings = float64(fullTokens-scopedTokens) / float64(fullTokens) * 100.0
	}

	// 4. Build Context Summary
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Pogonian AI Context for '%s' ===\n", prompt))
	sb.WriteString(fmt.Sprintf("Application: %s (%s)\n", e.AppIR.Name, e.AppIR.Architecture))
	sb.WriteString("Flow: Page → Form Action → Handler → Service → Repository → Database\n\n")

	sb.WriteString("Relevant Entities: " + strings.Join(relevantEntities, ", ") + "\n")
	sb.WriteString("Relevant Routes: " + strings.Join(relevantRoutes, ", ") + "\n")
	sb.WriteString("Targeted Files:\n")
	for _, f := range relevantFiles {
		rel, _ := filepath.Rel(e.ProjectDir, f)
		sb.WriteString("  • " + rel + "\n")
	}

	return &AIContext{
		Prompt:           prompt,
		RelevantFiles:    relevantFiles,
		RelevantEntities: relevantEntities,
		RelevantRoutes:   relevantRoutes,
		ArchitectureFlow: "Page → Handler → Service → Repository → Database",
		ContextSummary:   sb.String(),
		FullRepoTokens:   fullTokens,
		ScopedTokens:     scopedTokens,
		SavingsPercent:   savings,
	}, nil
}

func (e *ContextEngine) discoverAllCodeFiles() []string {
	var files []string
	_ = filepath.WalkDir(e.ProjectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Skip binaries, hidden git dirs, dependencies, build caches
		rel, _ := filepath.Rel(e.ProjectDir, path)
		if strings.HasPrefix(rel, ".") || strings.Contains(rel, "node_modules") ||
			strings.Contains(rel, "dist") || strings.Contains(rel, "bin") {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".html" || ext == ".css" || ext == ".js" || ext == ".yaml" || ext == ".ai" || ext == ".sql" {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range slice {
		if !keys[entry] && entry != "" {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
