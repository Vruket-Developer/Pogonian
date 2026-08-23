package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FrameworkType identifies the legacy framework of a source project.
type FrameworkType string

const (
	FrameworkReact   FrameworkType = "React"
	FrameworkNextJS  FrameworkType = "Next.js"
	FrameworkVue     FrameworkType = "Vue"
	FrameworkNuxt    FrameworkType = "Nuxt"
	FrameworkSvelte  FrameworkType = "Svelte"
	FrameworkExpress FrameworkType = "Express"
	FrameworkGeneric FrameworkType = "Generic Web"
)

// SourceProject represents a detected legacy application.
type SourceProject struct {
	Path       string        `json:"path"`
	Name       string        `json:"name"`
	Framework  FrameworkType `json:"framework"`
	HasBackend bool          `json:"has_backend"`
	HasRouter  bool          `json:"has_router"`
	FileCount  int           `json:"file_count"`
}

// DetectFramework analyzes package.json and directory structures of a target codebase.
func DetectFramework(sourceDir string) (*SourceProject, error) {
	proj := &SourceProject{
		Path:      sourceDir,
		Name:      filepath.Base(sourceDir),
		Framework: FrameworkGeneric,
	}

	pkgPath := filepath.Join(sourceDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Name         string            `json:"name"`
			Dependencies map[string]string `json:"dependencies"`
			DevDeps      map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err == nil {
			if pkg.Name != "" {
				proj.Name = pkg.Name
			}
			deps := make(map[string]bool)
			for k := range pkg.Dependencies {
				deps[k] = true
			}
			for k := range pkg.DevDeps {
				deps[k] = true
			}

			if deps["next"] {
				proj.Framework = FrameworkNextJS
				proj.HasRouter = true
			} else if deps["nuxt"] {
				proj.Framework = FrameworkNuxt
				proj.HasRouter = true
			} else if deps["svelte"] || deps["@sveltejs/kit"] {
				proj.Framework = FrameworkSvelte
				proj.HasRouter = true
			} else if deps["vue"] {
				proj.Framework = FrameworkVue
				if deps["vue-router"] {
					proj.HasRouter = true
				}
			} else if deps["react"] {
				proj.Framework = FrameworkReact
				if deps["react-router"] || deps["react-router-dom"] {
					proj.HasRouter = true
				}
			} else if deps["express"] {
				proj.Framework = FrameworkExpress
				proj.HasBackend = true
			}
		}
	}

	// Fallback filesystem heuristics
	if proj.Framework == FrameworkGeneric {
		if _, err := os.Stat(filepath.Join(sourceDir, "pages")); err == nil {
			proj.Framework = FrameworkNextJS
			proj.HasRouter = true
		} else if _, err := os.Stat(filepath.Join(sourceDir, "src", "App.vue")); err == nil {
			proj.Framework = FrameworkVue
		} else if _, err := os.Stat(filepath.Join(sourceDir, "src", "App.svelte")); err == nil {
			proj.Framework = FrameworkSvelte
		} else if _, err := os.Stat(filepath.Join(sourceDir, "src", "App.jsx")); err == nil {
			proj.Framework = FrameworkReact
		}
	}

	return proj, nil
}
