package migration

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"pogonian/pkg/ir"
	"pogonian/templates"
)

// GenerateFromIR scaffolds a fresh Pogonian application matching the provided IR.
func GenerateFromIR(appIR *ir.Application, targetDir string) error {
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	ctx := struct {
		AppName   string
		CreatedAt string
		Version   string
	}{
		AppName:   appIR.Name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Version:   "0.1.0",
	}

	// 1. Scaffold base templates
	err = fs.WalkDir(templates.DefaultFS, "default", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel("default", path)
		if err != nil || relPath == "." {
			return nil
		}

		destRelPath := relPath
		if filepath.Base(destRelPath) == "gitignore" {
			destRelPath = filepath.Join(filepath.Dir(destRelPath), ".gitignore")
		}
		destPath := filepath.Join(absPath, destRelPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := templates.DefaultFS.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(path).Parse(string(content))
		var buf bytes.Buffer
		if err == nil {
			_ = tmpl.Execute(&buf, ctx)
		} else {
			buf.Write(content)
		}

		_ = os.MkdirAll(filepath.Dir(destPath), 0755)
		return os.WriteFile(destPath, buf.Bytes(), 0644)
	})
	if err != nil {
		return fmt.Errorf("failed to scaffold template files: %w", err)
	}

	// 2. Generate additional routes discovered from migration
	for _, r := range appIR.Routes {
		if r.Path == "/" || r.Path == "/about" || r.Path == "/architecture" || r.Path == "/products" {
			continue
		}

		routeDir := filepath.Join(targetDir, "app", strings.TrimPrefix(r.Path, "/"))
		_ = os.MkdirAll(routeDir, 0755)

		pageFile := filepath.Join(routeDir, "page.html")
		title := filepath.Base(routeDir)
		content := fmt.Sprintf(`<section class="content-section">
    <h1>Migrated Page: %s</h1>
    <p>Migrated automatically from legacy application.</p>
    <div class="card">
        <p>Path: <code>%s</code></p>
    </div>
</section>`, title, r.Path)

		_ = os.WriteFile(pageFile, []byte(content), 0644)
	}

	return nil
}
