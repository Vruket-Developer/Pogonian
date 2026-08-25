package router

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Route represents a discovered filesystem route.
type Route struct {
	Path        string   // e.g. "/", "/about", "/products/:id"
	Pattern     string   // Regex pattern for matching
	PagePath    string   // Absolute or relative path to page.html
	LayoutPaths []string // Chain of layout.html paths from root to page
	Params      []string // Named route parameters, e.g. ["id"]
	Regex       *regexp.Regexp
}

// Router maintains the tree of registered filesystem routes.
type Router struct {
	AppDir string
	Routes []*Route
	Trie   *TrieRouter
}

// New creates a new filesystem router for the given app directory.
func New(appDir string) (*Router, error) {
	r := &Router{
		AppDir: appDir,
		Routes: make([]*Route, 0),
		Trie:   NewTrieRouter(),
	}

	if err := r.Scan(); err != nil {
		return nil, err
	}

	return r, nil
}

// Scan traverses the app/ directory and builds the route list.
func (r *Router) Scan() error {
	if _, err := os.Stat(r.AppDir); os.IsNotExist(err) {
		return fmt.Errorf("app directory '%s' not found", r.AppDir)
	}

	routes := make([]*Route, 0)

	err := filepath.WalkDir(r.AppDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fileName := d.Name()
		if fileName != "page.html" {
			return nil
		}

		// Calculate relative path from AppDir
		relDir, err := filepath.Rel(r.AppDir, filepath.Dir(path))
		if err != nil {
			return err
		}

		var urlPath string
		if relDir == "." || relDir == "" {
			urlPath = "/"
		} else {
			normalized := strings.ReplaceAll(relDir, "\\", "/")
			parts := strings.Split(normalized, "/")
			for i, part := range parts {
				if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
					parts[i] = ":" + strings.Trim(part, "[]")
				}
			}
			urlPath = "/" + strings.Join(parts, "/")
		}

		layouts := r.findLayouts(filepath.Dir(path))
		params := extractParams(urlPath)
		pattern := buildPattern(urlPath)
		compiledRegex, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("failed to compile route regex for %s: %w", urlPath, err)
		}

		route := &Route{
			Path:        urlPath,
			Pattern:     pattern,
			PagePath:    path,
			LayoutPaths: layouts,
			Params:      params,
			Regex:       compiledRegex,
		}

		routes = append(routes, route)
		r.Trie.Insert(urlPath, route)
		return nil
	})

	if err != nil {
		return err
	}

	sort.Slice(routes, func(i, j int) bool {
		cI := strings.Count(routes[i].Path, ":")
		cJ := strings.Count(routes[j].Path, ":")
		if cI != cJ {
			return cI < cJ
		}
		return len(routes[i].Path) > len(routes[j].Path)
	})

	r.Routes = routes
	return nil
}

// Match tests a requested URL path against the Radix Trie.
func (r *Router) Match(path string) (*Route, map[string]string) {
	if route, params := r.Trie.Match(path); route != nil {
		return route, params
	}

	for _, route := range r.Routes {
		if route.Regex == nil {
			continue
		}
		matches := route.Regex.FindStringSubmatch(path)
		if matches != nil {
			params := make(map[string]string)
			for i, name := range route.Params {
				if i+1 < len(matches) {
					params[name] = matches[i+1]
				}
			}
			return route, params
		}
	}

	return nil, nil
}

func (r *Router) findLayouts(pageDir string) []string {
	var layouts []string
	curr := pageDir

	for {
		candidate := filepath.Join(curr, "layout.html")
		if _, err := os.Stat(candidate); err == nil {
			layouts = append([]string{candidate}, layouts...)
		}

		if curr == r.AppDir || curr == filepath.Dir(curr) {
			break
		}
		curr = filepath.Dir(curr)
	}

	return layouts
}

func extractParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, strings.TrimPrefix(part, ":"))
		}
	}
	return params
}

func buildPattern(path string) string {
	if path == "/" {
		return `^/?$`
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	var regexParts []string

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			regexParts = append(regexParts, `([^/]+)`)
		} else {
			regexParts = append(regexParts, regexp.QuoteMeta(part))
		}
	}

	return `^/` + strings.Join(regexParts, `/`) + `/?$`
}
