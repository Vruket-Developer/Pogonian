package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pogonian/internal/server"
)

// DevOptions configures the dev server CLI execution.
type DevOptions struct {
	Port       int
	Host       string
	ProjectDir string
	KillPort   bool
	StrictPort bool
}

// RunDev executes `pogn dev` / `pogn run`.
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

	return srv.Start()
}
