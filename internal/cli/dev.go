package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"pogonian/pkg/server"
)

// DevOptions configures the dev server CLI execution.
type DevOptions struct {
	Port       int
	Host       string
	ProjectDir string
	KillPort   bool
	StrictPort bool
}

// RunDev executes `pogn dev` / `pogn run` with dual frontend/backend orchestration and fast refresh.
func RunDev(args []string) error {
	opts := DevOptions{
		ProjectDir: ".",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--port="):
			p, err := strconv.Atoi(strings.TrimPrefix(arg, "--port="))
			if err == nil {
				opts.Port = p
			}
		case arg == "-p" || arg == "--port":
			if i+1 < len(args) {
				p, err := strconv.Atoi(args[i+1])
				if err == nil {
					opts.Port = p
					i++
				}
			}
		case strings.HasPrefix(arg, "--host="):
			opts.Host = strings.TrimPrefix(arg, "--host=")
		case arg == "--kill" || arg == "--kill-port" || arg == "-k" || arg == "--clear":
			opts.KillPort = true
		case arg == "--strict" || arg == "--strict-port":
			opts.StrictPort = true
		case !strings.HasPrefix(arg, "-"):
			opts.ProjectDir = arg
		}
	}

	absDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Verify it is a Pogonian project
	yamlPath := filepath.Join(absDir, "pogonian.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return fmt.Errorf("no pogonian.yaml found in %s. Are you inside a Pogonian project?", absDir)
	}

	srv, err := server.New(absDir, true)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	if opts.Port > 0 {
		srv.Config.Server.Port = opts.Port
	}
	if opts.Host != "" {
		srv.Config.Server.Host = opts.Host
	}

	srv.KillPortOnConflict = opts.KillPort
	srv.StrictPort = opts.StrictPort

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Setup Signal Intercept for Graceful Exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 2. Start Frontend Dev Process if present in package.json
	frontendCmd := detectFrontendCommand(absDir)
	if frontendCmd != nil {
		fmt.Println("  \x1b[1;35m➜  Frontend:\x1b[0m  Orchestrating frontend dev process alongside Go backend...")
		go startProcess(ctx, frontendCmd, "\x1b[1;35m[frontend]\x1b[0m")
	}

	// 3. Start Intelligent File Watcher for Fast Refresh (HTML, CSS, JS, Components, Go)
	go startDevWatcher(ctx, absDir, srv)

	// 4. Start HTTP Server in background
	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- srv.Start()
	}()

	// 5. Await Shutdown Signal or Fatal Server Error
	select {
	case <-sigChan:
		fmt.Println("\n  \x1b[1;33m🛑 Gracefully shutting down Pogonian dev environment...\x1b[0m")
		cancel()
		_ = srv.Shutdown(context.Background())
		return nil
	case err := <-serverErrChan:
		cancel()
		return err
	}
}

// detectFrontendCommand inspects package.json for dedicated frontend scripts.
func detectFrontendCommand(projectDir string) *exec.Cmd {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	// Look for specific frontend dev scripts
	preferredScripts := []string{"dev:frontend", "frontend", "vite", "watch", "client", "build:watch"}
	for _, s := range preferredScripts {
		if _, exists := pkg.Scripts[s]; exists {
			pm := detectPackageManager(projectDir)
			cmd := exec.Command(pm, "run", s)
			cmd.Dir = projectDir
			return cmd
		}
	}

	// Check if "dev" exists and doesn't just call pogn / make
	if devScript, exists := pkg.Scripts["dev"]; exists {
		lower := strings.ToLower(devScript)
		if !strings.Contains(lower, "pogn") && !strings.Contains(lower, "make") && !strings.Contains(lower, "go run") {
			pm := detectPackageManager(projectDir)
			cmd := exec.Command(pm, "run", "dev")
			cmd.Dir = projectDir
			return cmd
		}
	}

	return nil
}

func detectPackageManager(projectDir string) string {
	if _, err := os.Stat(filepath.Join(projectDir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(projectDir, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(projectDir, "bun.lockb")); err == nil {
		return "bun"
	}
	return "npm"
}

func startProcess(ctx context.Context, cmd *exec.Cmd, prefix string) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	go pipeOutput(stdout, prefix)
	go pipeOutput(stderr, prefix)

	<-ctx.Done()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func pipeOutput(r io.Reader, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			fmt.Printf("  %s %s\n", prefix, line)
		}
	}
}

// startDevWatcher tracks filesystem changes and triggers Live Reload or Go Fast Refresh.
func startDevWatcher(ctx context.Context, projectDir string, srv *server.Server) {
	fileModTimes := make(map[string]time.Time)
	var mu sync.Mutex

	// Initial scan to populate file map
	scanFiles := func() map[string]time.Time {
		times := make(map[string]time.Time)
		dirsToWatch := []string{
			filepath.Join(projectDir, "app"),
			filepath.Join(projectDir, "components"),
			filepath.Join(projectDir, "server"),
			filepath.Join(projectDir, "api"),
			filepath.Join(projectDir, "public"),
		}

		for _, d := range dirsToWatch {
			if _, err := os.Stat(d); os.IsNotExist(err) {
				continue
			}
			_ = filepath.WalkDir(d, func(path string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return nil
				}
				// Skip hidden files/dirs
				if strings.HasPrefix(entry.Name(), ".") {
					return nil
				}
				if info, err := entry.Info(); err == nil {
					times[path] = info.ModTime()
				}
				return nil
			})
		}

		// Root files
		for _, f := range []string{"globals.css", "globals.js", "pogonian.yaml"} {
			fp := filepath.Join(projectDir, f)
			if info, err := os.Stat(fp); err == nil {
				times[fp] = info.ModTime()
			}
		}

		return times
	}

	mu.Lock()
	fileModTimes = scanFiles()
	mu.Unlock()

	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentTimes := scanFiles()

			var modifiedFile string
			var isCSS, isGo, isHTML bool

			mu.Lock()
			// Check for modified or new files
			for path, modTime := range currentTimes {
				prevTime, exists := fileModTimes[path]
				if !exists || modTime.After(prevTime) {
					modifiedFile = path
					ext := strings.ToLower(filepath.Ext(path))
					if ext == ".css" {
						isCSS = true
					} else if ext == ".go" {
						isGo = true
					} else if ext == ".html" || ext == ".js" || ext == ".json" {
						isHTML = true
					}
					break
				}
			}

			// Check for deleted files
			if modifiedFile == "" {
				for path := range fileModTimes {
					if _, exists := currentTimes[path]; !exists {
						modifiedFile = path
						isHTML = true
						break
					}
				}
			}

			fileModTimes = currentTimes
			mu.Unlock()

			if modifiedFile != "" {
				rel, _ := filepath.Rel(projectDir, modifiedFile)

				if isCSS {
					fmt.Printf("  \x1b[1;36m[hot-reload]\x1b[0m CSS updated: \x1b[1m%s\x1b[0m\n", rel)
					srv.BroadcastLiveReload("css-update")
				} else if isGo {
					fmt.Printf("  \x1b[1;35m[fast-refresh]\x1b[0m Go backend change detected: \x1b[1m%s\x1b[0m (live updating)\n", rel)
					// Rescan routes and notify browser
					_ = srv.Router.Scan()
					srv.BroadcastLiveReload("go-reloaded")
				} else if isHTML || modifiedFile != "" {
					fmt.Printf("  \x1b[1;32m[live-reload]\x1b[0m Template / component updated: \x1b[1m%s\x1b[0m\n", rel)
					// Rescan routes dynamically so any new route/component is instantly active
					_ = srv.Router.Scan()
					srv.BroadcastLiveReload("html-update")
				}
			}
		}
	}
}
