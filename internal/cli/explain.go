package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"pogonian/internal/architecture"
	"pogonian/pkg/config"
	"pogonian/pkg/router"
)

// RunExplain executes `pogn explain [target]`.
func RunExplain(args []string) error {
	target := ""
	projectDir := "."

	for _, arg := range args {
		if strings.HasPrefix(arg, "--dir=") {
			projectDir = strings.TrimPrefix(arg, "--dir=")
		} else if target == "" && !strings.HasPrefix(arg, "-") {
			target = arg
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

	fmt.Println()
	if target == "" {
		// Explain whole application
		r, _ := router.New(filepath.Join(absDir, "app"))
		routeCount := 0
		if r != nil {
			routeCount = len(r.Routes)
		}

		fmt.Println("🧠 \x1b[1mPogonian Application Architecture Explanation\x1b[0m")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  Application:    \x1b[1;36m%s\x1b[0m (Modular Monolith)\n", appName)
		fmt.Printf("  Routes:         \x1b[1m%d frontend pages\x1b[0m\n", routeCount)
		fmt.Printf("  Database:       \x1b[1mSQLite WAL Mode\x1b[0m (Zero Global State)\n")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("\x1b[1mHOW THE APPLICATION RUNS:\x1b[0m")
		fmt.Println("  1. \x1b[1;34mBrowser\x1b[0m makes a soft-navigation request with \x1b[1m_pogn/runtime.js\x1b[0m")
		fmt.Println("  2. \x1b[1;33mRadix Route Trie\x1b[0m matches the URL in O(k) time")
		fmt.Println("  3. \x1b[1;35mHTTP Handler\x1b[0m validates CSRF & calls the domain \x1b[1mService\x1b[0m")
		fmt.Println("  4. \x1b[1;36mService\x1b[0m enforces business rules & calls the \x1b[1mRepository\x1b[0m")
		fmt.Println("  5. \x1b[1;32mRepository\x1b[0m queries SQLite via parameterized SQL")
		fmt.Println("  6. \x1b[1;37mHTML Renderer\x1b[0m injects data into the layout with zero full-page reload")
		fmt.Println()
		fmt.Println("💡 \x1b[1mTip:\x1b[0m Run '\x1b[1;36mpogn explain <feature>\x1b[0m' (e.g. \x1b[1;36mpogn explain products\x1b[0m) to inspect a specific module flow.")
	} else {
		// Explain specific module or file
		cleanTarget := strings.ToLower(strings.Trim(target, "/"))
		report, _ := architecture.AnalyzeImpact(absDir, cleanTarget)

		fmt.Printf("🧠 \x1b[1mFeature Architecture:\x1b[0m \x1b[1;36m%s\x1b[0m\n", cleanTarget)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("\x1b[1mARCHITECTURAL FLOW:\x1b[0m")
		fmt.Printf("  🌐 \x1b[1;34mRoute (Frontend):\x1b[0m      app/%s/page.html\n", cleanTarget)
		fmt.Printf("  ⚡ \x1b[1;33mHTTP Handler:\x1b[0m          server/%s/handler.go\n", cleanTarget)
		fmt.Printf("  📦 \x1b[1;35mService (Logic):\x1b[0m       server/%s/service.go\n", cleanTarget)
		fmt.Printf("  🗄️  \x1b[1;36mRepository (SQL):\x1b[0m      server/%s/repository.go\n", cleanTarget)
		fmt.Printf("  💾 \x1b[1;32mDatabase Table:\x1b[0m        table '%s' in data.db\n", cleanTarget)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("\x1b[1mWHY THIS ARCHITECTURE EXISTS:\x1b[0m")
		fmt.Println("  • The \x1b[1mHandler\x1b[0m handles HTTP parameters, cookies, and CSRF tokens.")
		fmt.Println("  • The \x1b[1mService\x1b[0m isolates business logic so handlers never touch SQL directly.")
		fmt.Println("  • The \x1b[1mRepository\x1b[0m encapsulates database queries preventing SQL injection.")
		if report != nil && len(report.PotentialBreaks) > 0 {
			fmt.Println()
			fmt.Println("⚠️  \x1b[1;31mLayer Boundary Notes:\x1b[0m")
			for _, b := range report.PotentialBreaks {
				fmt.Printf("   ↳ %s\n", b)
			}
		}
	}
	fmt.Println()
	return nil
}
