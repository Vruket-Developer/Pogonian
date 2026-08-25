package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pogonian/internal/compiler"
	"pogonian/internal/config"
	"pogonian/internal/router"
	"pogonian/internal/version"
)

// BuildOptions holds flags and arguments for the production build pipeline.
type BuildOptions struct {
	ProjectDir string
	Standalone bool
	Minify     bool
	OutputDir  string
	TargetOS   string
	TargetArch string
}

// RunBuild executes `pogn build` / `pogn build --standalone`.
func RunBuild(args []string) error {
	opts := BuildOptions{
		ProjectDir: ".",
		OutputDir:  "dist",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--standalone" || arg == "-s":
			opts.Standalone = true
		case arg == "--minify" || arg == "-m":
			opts.Minify = true
		case strings.HasPrefix(arg, "--output="):
			opts.OutputDir = strings.TrimPrefix(arg, "--output=")
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				opts.OutputDir = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--target="):
			parts := strings.Split(strings.TrimPrefix(arg, "--target="), "/")
			if len(parts) == 2 {
				opts.TargetOS = parts[0]
				opts.TargetArch = parts[1]
			}
		case !strings.HasPrefix(arg, "-"):
			opts.ProjectDir = arg
		}
	}

	absDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return err
	}

	start := time.Now()
	cfg, err := config.Load(absDir)
	if err != nil {
		cfg = config.DefaultConfig(filepath.Base(absDir))
	}

	appName := cfg.Name
	if appName == "" {
		appName = "app"
	}

	fmt.Println()
	fmt.Printf("📦 \x1b[1;36mPogonian Production Compiler\x1b[0m \x1b[1;30mv%s\x1b[0m\n", version.Current)
	fmt.Printf("   Project:   \x1b[1m%s\x1b[0m\n", absDir)
	fmt.Printf("   Target:    %s\n\n", opts.OutputDir)

	// Step 1: Pre-build Architecture & Route Integrity Check
	fmt.Printf("  🔍 \x1b[1m[1/5] Verifying architecture & route integrity...\x1b[0m\n")
	if err := RunCheck([]string{absDir}); err != nil {
		return fmt.Errorf("build aborted: architecture checks failed: %w", err)
	}

	// Step 2: Route Scanning & Template Precompilation
	fmt.Printf("  ⚙️  \x1b[1m[2/5] Precompiling routes and template hierarchies...\x1b[0m\n")
	appDir := filepath.Join(absDir, "app")
	r, err := router.New(appDir)
	if err != nil {
		return fmt.Errorf("failed to scan routes: %w", err)
	}

	rend := compiler.NewRendererWithMode(appName, absDir, false)
	if err := rend.PrecompileAll(r.Routes); err != nil {
		return fmt.Errorf("template precompilation failed: %w", err)
	}
	fmt.Printf("     \x1b[1;32m✓\x1b[0m Precompiled %d filesystem routes\n", len(r.Routes))

	// Step 3: Prepare Isolated Build Cache and Dist Directory
	fmt.Printf("  📁 \x1b[1m[3/5] Preparing release artifacts...\x1b[0m\n")
	buildCacheDir := filepath.Join(absDir, ".pogonian", "build")
	distDir := filepath.Join(absDir, opts.OutputDir)

	if err := os.MkdirAll(buildCacheDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return err
	}
	// Step 4: Synthesize Production Release Entrypoint
	fmt.Printf("  🔨 \x1b[1m[4/5] Generating standalone production entrypoint...\x1b[0m\n")
	entrypointCode := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pogonian/pkg/config"
	"pogonian/pkg/server"
)

func main() {
	// Set production environment
	os.Setenv("APP_ENV", "production")

	cfg, err := config.Load(".")
	if err != nil {
		cfg = config.DefaultConfig("%s")
	}

	// Dynamic PORT binding for Cloud / Docker / Serverless
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		var p int
		if _, err := fmt.Sscanf(portEnv, "%%d", &p); err == nil && p > 0 {
			cfg.Server.Port = p
		}
	}

	// Initialize Production Server (DevMode = false: no DevTools overhead, precompiled templates)
	srv, err := server.New(".", false)
	if err != nil {
		log.Fatalf("Fatal Pogonian production startup error: %%v", err)
	}

	// Graceful Shutdown Listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Pogonian production server crashed: %%v", err)
		}
	}()

	sig := <-stop
	fmt.Printf("\n🛑 Received signal %%v. Initiating graceful shutdown...\n", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during graceful shutdown: %%v", err)
	} else {
		fmt.Println("✨ Server shut down cleanly.")
	}
}
`, appName)

	entryFile := filepath.Join(buildCacheDir, "main.go")
	if err := os.WriteFile(entryFile, []byte(entrypointCode), 0644); err != nil {
		return fmt.Errorf("failed to write build entrypoint: %w", err)
	}

	// Ensure project has a valid go.mod with pogonian dependency resolved
	if err := ensureProjectGoMod(absDir); err != nil {
		return fmt.Errorf("failed to prepare project dependencies: %w", err)
	}

	// Step 5: Compile Standalone Native Binary with Optimization Flags
	fmt.Printf("  🚀 \x1b[1m[5/5] Compiling native release binary with Go toolchain...\x1b[0m\n")
	outBin := filepath.Join(distDir, appName)

	buildArgs := []string{
		"build",
		"-trimpath",
		"-ldflags=-s -w -buildid=",
		"-o", outBin,
		"./.pogonian/build",
	}

	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = absDir

	env := os.Environ()
	if opts.TargetOS != "" {
		env = append(env, "GOOS="+opts.TargetOS)
	}
	if opts.TargetArch != "" {
		env = append(env, "GOARCH="+opts.TargetArch)
	}
	cmd.Env = env

	if out, err := cmd.CombinedOutput(); err != nil {
		// Strict zero-tolerance: never fake build success by copying pogn binary
		return fmt.Errorf("production build failed: go compilation error:\n%s", string(out))
	}

	// Compute build hash and size
	binData, _ := os.ReadFile(outBin)
	binHash := sha256.Sum256(binData)
	hashStr := hex.EncodeToString(binHash[:8])

	info, err := os.Stat(outBin)
	sizeMB := 0.0
	if err == nil {
		sizeMB = float64(info.Size()) / (1024 * 1024)
	}

	duration := time.Since(start)

	fmt.Println()
	fmt.Printf("✨ \x1b[1;32mStandalone production build complete! Release created in %s!\x1b[0m\n", duration.Round(time.Millisecond))
	fmt.Printf("   • Binary:       \x1b[1m%s\x1b[0m (%.1f MB)\n", outBin, sizeMB)
	fmt.Printf("   • Build Hash:   \x1b[1;30m%s\x1b[0m\n", hashStr)
	fmt.Printf("   • Environment:  Production (Zero Dev Overhead, Precompiled Templates)\n")
	fmt.Printf("   • To Run:       \x1b[1;34m./dist/%s\x1b[0m\n\n", appName)

	return nil
}

// ensureProjectGoMod ensures that the target project has a valid go.mod with pogonian dependency.
func ensureProjectGoMod(projectDir string) error {
	goModPath := filepath.Join(projectDir, "go.mod")

	repoRoot, err := findFrameworkRoot()
	if err != nil {
		return nil // standalone remote build
	}

	// If building inside the framework repo itself, go.mod is already present
	absProject, _ := filepath.Abs(projectDir)
	absRoot, _ := filepath.Abs(repoRoot)
	if absProject == absRoot {
		return nil
	}

	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		appName := filepath.Base(projectDir)
		modContent := fmt.Sprintf(`module %s

go 1.22

require pogonian v%s

replace pogonian => %s
`, appName, version.Current, absRoot)

		return os.WriteFile(goModPath, []byte(modContent), 0644)
	}

	// Ensure require and replace directives point to local framework if running in local environment
	data, err := os.ReadFile(goModPath)
	if err == nil {
		modStr := string(data)
		var newLines []string
		hasRequire := false
		hasReplace := false

		for _, line := range strings.Split(modStr, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "require pogonian") {
				hasRequire = true
			}
			if strings.HasPrefix(trimmed, "replace pogonian") {
				hasReplace = true
				line = fmt.Sprintf("replace pogonian => %s", absRoot)
			}
			newLines = append(newLines, line)
		}

		if !hasRequire {
			newLines = append(newLines, fmt.Sprintf("\nrequire pogonian v%s", version.Current))
		}
		if !hasReplace {
			newLines = append(newLines, fmt.Sprintf("\nreplace pogonian => %s", absRoot))
		}

		_ = os.WriteFile(goModPath, []byte(strings.Join(newLines, "\n")), 0644)
	}

	// Resolve and record all direct & indirect dependencies for the build entrypoint
	getCmd := exec.Command("go", "get", "./.pogonian/build")
	getCmd.Dir = projectDir
	_, _ = getCmd.CombinedOutput()

	return nil
}

func findFrameworkRoot() (string, error) {
	// 1. Try checking executable directory or walking up from cwd
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(candidate); err == nil {
			if strings.Contains(string(data), "module pogonian") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("pogonian framework root not found")
}
