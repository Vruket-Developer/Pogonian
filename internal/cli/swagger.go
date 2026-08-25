package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/openapi"
)

// RunSwagger handles the `pogn swagger` command.
func RunSwagger(args []string) error {
	projectDir := "."
	exportPath := ""
	outputJSON := false
	openBrowser := false
	port := 3000

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			outputJSON = true
		case strings.HasPrefix(arg, "--export="):
			exportPath = strings.TrimPrefix(arg, "--export=")
		case arg == "--export" || arg == "-e":
			if i+1 < len(args) {
				exportPath = args[i+1]
				i++
			}
		case arg == "--open" || arg == "-o":
			openBrowser = true
		case strings.HasPrefix(arg, "--port="):
			if p, err := strconv.Atoi(strings.TrimPrefix(arg, "--port=")); err == nil {
				port = p
			}
		case arg == "-p" || arg == "--port":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil {
					port = p
					i++
				}
			}
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	cfg, _ := config.Load(absDir)
	if cfg != nil && cfg.Server.Port > 0 && port == 3000 {
		port = cfg.Server.Port
	}

	spec, err := openapi.GenerateSpec(absDir, port)
	if err != nil {
		return fmt.Errorf("failed to generate Swagger / OpenAPI specification: %w", err)
	}

	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode specification JSON: %w", err)
	}

	if exportPath != "" {
		outPath := exportPath
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(absDir, exportPath)
		}
		if err := os.WriteFile(outPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write OpenAPI spec to %s: %w", exportPath, err)
		}
		fmt.Println()
		fmt.Printf("✨ \x1b[1;32mExported OpenAPI 3.0 Spec:\x1b[0m %s\n", outPath)
		fmt.Printf("   • Total Paths: %d\n", len(spec.Paths))
		fmt.Printf("   • Schemas: %d\n\n", len(spec.Components.Schemas))
		return nil
	}

	if outputJSON {
		fmt.Println(string(jsonData))
		return nil
	}

	swaggerURL := fmt.Sprintf("http://localhost:%d/swagger", port)

	fmt.Println()
	fmt.Printf("📚 \x1b[1;36mPogonian Swagger & OpenAPI Engine\x1b[0m\n")
	fmt.Printf("   Project: %s\n", absDir)
	fmt.Printf("   Version: OpenAPI %s\n", spec.OpenAPI)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  ➜  \x1b[1mInteractive Swagger UI:\x1b[0m   \x1b[1;32m%s\x1b[0m\n", swaggerURL)
	fmt.Printf("  ➜  \x1b[1mOpenAPI 3.0 JSON Spec:\x1b[0m    \x1b[1;34mhttp://localhost:%d/swagger.json\x1b[0m\n", port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  \x1b[1mDiscovered API Routes & Endpoints (%d):\x1b[0m\n", len(spec.Paths))

	for path, item := range spec.Paths {
		var methods []string
		if item.Get != nil {
			methods = append(methods, "GET")
		}
		if item.Post != nil {
			methods = append(methods, "POST")
		}
		if item.Put != nil {
			methods = append(methods, "PUT")
		}
		if item.Delete != nil {
			methods = append(methods, "DELETE")
		}
		if item.Patch != nil {
			methods = append(methods, "PATCH")
		}

		tag := "General"
		summary := item.Summary
		if item.Get != nil && len(item.Get.Tags) > 0 {
			tag = item.Get.Tags[0]
			if summary == "" {
				summary = item.Get.Summary
			}
		} else if item.Post != nil && len(item.Post.Tags) > 0 {
			tag = item.Post.Tags[0]
			if summary == "" {
				summary = item.Post.Summary
			}
		}

		fmt.Printf("  • \x1b[1;36m%-26s\x1b[0m \x1b[1;32m[%-12s]\x1b[0m \x1b[1;30m(%s)\x1b[0m %s\n", path, strings.Join(methods, ","), tag, summary)
	}

	fmt.Println()
	fmt.Println("  \x1b[1;30mTip: Start dev server with 'pogn dev' and navigate to /swagger in your browser.\x1b[0m")
	fmt.Println("  \x1b[1;30mExport spec: 'pogn swagger --export=openapi.json' or 'pogn swagger --json'\x1b[0m")
	fmt.Println()

	if openBrowser {
		openURL(swaggerURL)
	}

	return nil
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
