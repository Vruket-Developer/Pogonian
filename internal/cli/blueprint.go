package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pogonian/internal/architecture"
	"pogonian/pkg/config"
	"pogonian/pkg/router"
	"gopkg.in/yaml.v3"
)

// RunBlueprint exports the architecture blueprint.
func RunBlueprint(args []string) error {
	format := "mermaid"
	projectDir := "."

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
	appName := "PogonianApp"
	if cfg != nil && cfg.Name != "" {
		appName = cfg.Name
	}

	analyzer, err := architecture.NewAnalyzer(absDir)
	if err != nil {
		return err
	}
	spec, _, err := analyzer.Analyze()
	if err != nil {
		return err
	}

	switch strings.ToLower(format) {
	case "json":
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil

	case "yaml", "yml":
		data, err := yaml.Marshal(spec)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil

	case "mermaid", "mmd":
		return printMermaidBlueprint(appName, absDir)

	default:
		return fmt.Errorf("unknown blueprint format '%s'. Supported formats: mermaid, json, yaml", format)
	}
}

func printMermaidBlueprint(appName, projectDir string) error {
	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		return err
	}
	spec, _, err := analyzer.Analyze()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("📐 \x1b[1;36mPogonian Architecture Blueprint (Mermaid Diagram):\x1b[0m %s\n", appName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("```mermaid")
	fmt.Println("graph TD")
	fmt.Println("  Client([Browser / Client Runtime])")
	fmt.Println("  Router[Radix Trie Router]")
	fmt.Println("  Client -->|HTTP / Soft Navigation| Router")
	fmt.Println()

	// Dynamic Pages from real IR
	fmt.Println("  subgraph AppLayer [App & Pages]")
	if len(spec.Routes) == 0 {
		fmt.Println("    HomePage[app/page.html]")
	} else {
		for i, r := range spec.Routes {
			nodeID := fmt.Sprintf("Page%d", i)
			cleanPath := strings.ReplaceAll(r.Path, "/", "_")
			cleanPath = strings.ReplaceAll(cleanPath, ":", "param_")
			if cleanPath == "_" || cleanPath == "" {
				cleanPath = "Home"
			}
			fmt.Printf("    %s[\"%s  (%s)\"]\n", nodeID, r.Path, r.Handler)
			fmt.Printf("    Router --> %s\n", nodeID)
		}
	}
	fmt.Println("  end")
	fmt.Println()

	// Dynamic Server Domains from real IR
	fmt.Println("  subgraph ServerLayer [5-Tier Server Domains]")
	if len(spec.Entities) == 0 {
		fmt.Println("    DefaultDomain[Core Services]")
	} else {
		for _, e := range spec.Entities {
			domPascal := e.Name
			fmt.Printf("    H_%s[\"%s Handler\"] --> S_%s[\"%s Service\"]\n", domPascal, domPascal, domPascal, domPascal)
			fmt.Printf("    S_%s --> R_%s[\"%s Repository\"]\n", domPascal, domPascal, domPascal)
			fmt.Printf("    R_%s --> DB[(Database / SQLite)]\n", domPascal)
		}
	}
	fmt.Println("  end")

	fmt.Println("```")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

var formTagRegex = regexp.MustCompile(`(?i)<form[^>]*action=["']([^"']*)["'][^>]*method=["']([^"']*)["']`)
var formMethodRegex = regexp.MustCompile(`(?i)<form[^>]*method=["']([^"']*)["']`)

// RunDocs generates 100% dynamic, AST & filesystem-driven API and route documentation.
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
	fmt.Printf("📚 \x1b[1;36mPogonian Automatic API Documentation:\x1b[0m \x1b[1m%s\x1b[0m\n", appName)
	fmt.Printf("   Directory: %s\n", absDir)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-24s %-12s %-18s %s\n", "ENDPOINT / ROUTE", "METHODS", "DATA CONTRACT", "DESCRIPTION")
	fmt.Println("────────────────────────────────────────────────────────────────────────────")

	type RouteDoc struct {
		Endpoint    string
		Methods     string
		Contract    string
		Description string
	}
	var docs []RouteDoc

	// 1. Inspect all filesystem routes in app/
	if r != nil && len(r.Routes) > 0 {
		for _, rt := range r.Routes {
			pagePath := filepath.Join(absDir, "app", strings.TrimPrefix(rt.Path, "/"), "page.html")
			if rt.Path == "/" {
				pagePath = filepath.Join(absDir, "app", "page.html")
			}

			methods := []string{"GET"}
			contract := "HTML View"
			desc := fmt.Sprintf("View page for route %s", rt.Path)

			// Scan HTML for forms and actions
			if content, err := os.ReadFile(pagePath); err == nil {
				htmlStr := string(content)
				if formMatches := formTagRegex.FindAllStringSubmatch(htmlStr, -1); len(formMatches) > 0 {
					for _, m := range formMatches {
						mMethod := strings.ToUpper(m[2])
						if !containsString(methods, mMethod) {
							methods = append(methods, mMethod)
						}
					}
					desc = "Interactive view & form action handler"
				} else if formMethodRegex.MatchString(htmlStr) {
					if !containsString(methods, "POST") {
						methods = append(methods, "POST")
					}
					desc = "Interactive view & form action handler"
				}
			}

			// Check matching domain in server/
			domainName := strings.Trim(rt.Path, "/")
			if domainName != "" && !strings.Contains(domainName, "/") {
				entityFile := filepath.Join(absDir, "server", domainName, "entity.go")
				if structName, fields, err := architecture.ParseEntityFile(entityFile); err == nil && len(fields) > 0 {
					contract = fmt.Sprintf("%s (%d fields)", structName, len(fields))
					desc = fmt.Sprintf("CRUD interface for %s entity", structName)
				}
			}

			docs = append(docs, RouteDoc{
				Endpoint:    rt.Path,
				Methods:     strings.Join(methods, "/"),
				Contract:    contract,
				Description: desc,
			})
		}
	}

	// 2. Discover any additional backend domains in server/ without frontend pages
	serverDir := filepath.Join(absDir, "server")
	if entries, err := os.ReadDir(serverDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				domName := entry.Name()
				entityFile := filepath.Join(serverDir, domName, "entity.go")
				if structName, fields, err := architecture.ParseEntityFile(entityFile); err == nil && len(fields) > 0 {
					endpointPath := "/" + domName
					alreadyListed := false
					for _, d := range docs {
						if d.Endpoint == endpointPath {
							alreadyListed = true
							break
						}
					}
					if !alreadyListed {
						docs = append(docs, RouteDoc{
							Endpoint:    endpointPath,
							Methods:     "GET/POST/PUT/DELETE",
							Contract:    fmt.Sprintf("%s (%d fields)", structName, len(fields)),
							Description: fmt.Sprintf("Backend service domain for %s", structName),
						})
					}
				}
			}
		}
	}

	if len(docs) == 0 {
		fmt.Println("  (No routes or server domains found in project)")
	} else {
		for _, d := range docs {
			fmt.Printf("%-24s %-12s %-18s %s\n", "\x1b[1;36m"+d.Endpoint+"\x1b[0m", "\x1b[1;32m"+d.Methods+"\x1b[0m", d.Contract, d.Description)
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Total Endpoints Discovered: %d\n\n", len(docs))
	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
