package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"pogonian/templates"
)

var validProjectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// CreateOptions holds flags for project generation.
type CreateOptions struct {
	Name     string
	Template string
	Force    bool
	DryRun   bool
	TargetDir string
}

// TemplateContext is passed to template files during rendering.
type TemplateContext struct {
	AppName   string
	CreatedAt string
	Version   string
}

// RunCreate executes the `pogn create` command.
func RunCreate(opts CreateOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("project name is required: pogn create <project-name>")
	}

	if !validProjectNameRegex.MatchString(opts.Name) {
		return fmt.Errorf("invalid project name '%s': only letters, numbers, hyphens, and underscores are allowed", opts.Name)
	}

	if opts.Template == "" {
		opts.Template = "default"
	}

	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = opts.Name
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Check if directory exists and is non-empty
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return fmt.Errorf("failed to inspect directory %s: %w", absPath, err)
		}
		if len(entries) > 0 && !opts.Force {
			return fmt.Errorf("directory '%s' is not empty. Use --force to overwrite", targetDir)
		}
	}

	if opts.DryRun {
		fmt.Printf("🔍 [Dry Run] Would create Pogonian project '%s' in: %s\n", opts.Name, absPath)
		return nil
	}

	fmt.Printf("🚀 Creating Pogonian application '%s' in %s...\n", opts.Name, targetDir)

	ctx := TemplateContext{
		AppName:   opts.Name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Version:   "0.1.0",
	}

	templateRoot := opts.Template
	err = fs.WalkDir(templates.DefaultFS, templateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Handle renamed files like gitignore -> .gitignore
		destRelPath := relPath
		if filepath.Base(destRelPath) == "gitignore" {
			destRelPath = filepath.Join(filepath.Dir(destRelPath), ".gitignore")
		}

		destPath := filepath.Join(absPath, destRelPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read and execute template
		content, err := templates.DefaultFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		tmpl, err := template.New(path).Parse(string(content))
		if err != nil {
			// If template parse fails (e.g. binary/raw file), write directly
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			return os.WriteFile(destPath, content, 0644)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			// On execution error, write raw content
			buf.Reset()
			buf.Write(content)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(destPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}

		fmt.Printf("  ✓ Created %s\n", destRelPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}

	// Create initial pogonian.lock
	lockContent := fmt.Sprintf(`# Pogonian Lockfile
version: 1
framework_version: 0.1.0
created_at: "%s"
generator: "pogn create"
`, ctx.CreatedAt)

	lockPath := filepath.Join(absPath, "pogonian.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0644); err != nil {
		return fmt.Errorf("failed to write pogonian.lock: %w", err)
	}
	fmt.Printf("  ✓ Created pogonian.lock\n")

	printSuccessMessage(opts.Name, targetDir)
	return nil
}

func printSuccessMessage(appName, targetDir string) {
	fmt.Println()
	fmt.Println("✨ \x1b[1;32mSuccess! Created\x1b[0m " + appName + " at " + targetDir)
	fmt.Println()
	fmt.Println("Inside that directory, you can run several commands:")
	fmt.Println()
	fmt.Println("  \x1b[1;36mpogn dev\x1b[0m")
	fmt.Println("    Starts the local development server with soft navigation and live reload.")
	fmt.Println()
	fmt.Println("  \x1b[1;36mpogn check\x1b[0m")
	fmt.Println("    Validates filesystem routes, architecture boundaries, and .ai specifications.")
	fmt.Println()
	fmt.Println("  \x1b[1;36mpogn build\x1b[0m")
	fmt.Println("    Compiles your HTML, CSS, assets, and Go backend into an optimized binary.")
	fmt.Println()
	fmt.Println("To get started:")
	if targetDir != "." {
		fmt.Printf("  cd %s\n", targetDir)
	}
	fmt.Println("  pogn dev")
	fmt.Println()
}

// ExtractCreateOptions parses command line flags for create.
func ExtractCreateOptions(args []string) (CreateOptions, error) {
	opts := CreateOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--force" || arg == "-f":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case strings.HasPrefix(arg, "--template="):
			opts.Template = strings.TrimPrefix(arg, "--template=")
		case arg == "--template" || arg == "-t":
			if i+1 < len(args) {
				opts.Template = args[i+1]
				i++
			} else {
				return opts, fmt.Errorf("--template requires a template name")
			}
		case !strings.HasPrefix(arg, "-"):
			if opts.Name == "" {
				opts.Name = arg
			} else if opts.TargetDir == "" {
				opts.TargetDir = arg
			}
		}
	}
	return opts, nil
}
