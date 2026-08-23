package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/router"
	"gopkg.in/yaml.v3"
)

// AppBlueprint represents a structured export of the application architecture.
type AppBlueprint struct {
	Name         string        `json:"name" yaml:"name"`
	Version      string        `json:"version" yaml:"version"`
	Style        string        `json:"style" yaml:"style"`
	Routes       []string      `json:"routes" yaml:"routes"`
	Database     string        `json:"database" yaml:"database"`
	Architecture []string      `json:"architecture" yaml:"architecture"`
	Security     []string      `json:"security" yaml:"security"`
}

// RunBlueprint executes `pogn blueprint`.
func RunBlueprint(args []string) error {
	projectDir := "."
	format := "mermaid"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--format=") {
			format = strings.TrimPrefix(arg, "--format=")
		} else if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	cfg, _ := config.Load(absDir)
	appName := "app"
	if cfg != nil && cfg.Name != "" {
		appName = cfg.Name
	}

	r, _ := router.New(filepath.Join(absDir, "app"))
	var routeList []string
	if r != nil {
		for _, route := range r.Routes {
			routeList = append(routeList, route.Path)
		}
	}

	bp := AppBlueprint{
		Name:    appName,
		Version: "0.1.0",
		Style:   "Modular Monolith (5-Tier)",
		Routes:  routeList,
		Database: "SQLite WAL / PostgreSQL (Driver Agnostic)",
		Architecture: []string{
			"Browser (Soft Navigation)",
			"Radix Route Trie",
			"HTTP Handlers",
			"Domain Services",
			"Repositories (SQL)",
			"Database",
		},
		Security: []string{
			"Argon2id Password Hashing",
			"Persistent Session Store",
			"CSRF Token Validation",
			"Token-Bucket Rate Limiter",
			"Panic Recovery",
		},
	}

	if format == "json" {
		data, _ := json.MarshalIndent(bp, "", "  ")
		fmt.Println(string(data))
		return nil
	} else if format == "yaml" {
		data, _ := yaml.Marshal(bp)
		fmt.Println(string(data))
		return nil
	}

	// Default: Mermaid Architecture Blueprint
	fmt.Println()
	fmt.Println("📐 \x1b[1mPogonian Application Architecture Blueprint\x1b[0m")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("```mermaid")
	fmt.Println("graph TD")
	fmt.Println("  Browser[\"🌐 Browser (runtime.js)\"] --> Router[\"⚡ Radix Route Trie\"]")
	for _, route := range routeList {
		safeID := strings.ReplaceAll(strings.Trim(route, "/"), "/", "_")
		if safeID == "" {
			safeID = "root"
		}
		fmt.Printf("  Router --> R_%s[\"Page: %s\"]\n", safeID, route)
		if safeID != "root" && safeID != "about" && safeID != "architecture" {
			fmt.Printf("  R_%s --> H_%s[\"Handler: %s\"]\n", safeID, safeID, safeID)
			fmt.Printf("  H_%s --> S_%s[\"Service: %s\"]\n", safeID, safeID, safeID)
			fmt.Printf("  S_%s --> Rep_%s[\"Repository: %s\"]\n", safeID, safeID, safeID)
			fmt.Printf("  Rep_%s --> DB[(\"💾 Database: %s\")]\n", safeID, safeID)
		}
	}
	fmt.Println("```")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

// RunDocs generates automatic API & route documentation.
func RunDocs(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	cfg, _ := config.Load(absDir)
	appName := "app"
	if cfg != nil && cfg.Name != "" {
		appName = cfg.Name
	}

	r, _ := router.New(filepath.Join(absDir, "app"))

	fmt.Println()
	fmt.Printf("📚 \x1b[1mPogonian Automatic API Documentation:\x1b[0m %s\n", appName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-24s %-10s %-25s %s\n", "ENDPOINT", "METHOD", "AUTH", "DESCRIPTION")
	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("%-24s %-10s %-25s %s\n", "/register", "POST", "Public", "User account registration")
	fmt.Printf("%-24s %-10s %-25s %s\n", "/login", "POST", "Public", "User authentication (Argon2id)")
	fmt.Printf("%-24s %-10s %-25s %s\n", "/logout", "POST", "Authenticated", "Invalidate active session")

	if r != nil {
		for _, route := range r.Routes {
			if route.Path != "/login" && route.Path != "/register" {
				fmt.Printf("%-24s %-10s %-25s %s\n", route.Path, "GET/POST", "Public/Guarded", "HTML view & form action")
			}
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}
