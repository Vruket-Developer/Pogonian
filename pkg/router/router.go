package router

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Route represents a discovered filesystem route.
type Route struct {
	Path        string   // e.g. "/", "/about", "/products/:id"
	Pattern     string   // Normalized pattern
	PagePath    string   // Absolute or relative path to page.html
	LayoutPaths []string // Chain of layout.html paths from root to page
	Params      []string // Named route parameters, e.g. ["id"]
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

// Scan traverses the app/ directory and builds the route list and Radix Trie.
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
		var params []string
		if relDir == "." || relDir == "" {
			urlPath = "/"
		} else {
			// Convert windows backslashes
			normalized := strings.ReplaceAll(relDir, "\\", "/")
			parts := strings.Split(normalized, "/")
			var urlParts []string
			for _, p := range parts {
				if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
					paramName := p[1 : len(p)-1]
					if strings.Contains(paramName, ":") {
						paramName = strings.SplitN(paramName, ":", 2)[0]
					}
					params = append(params, paramName)
					urlParts = append(urlParts, ":"+paramName)
				} else {
					urlParts = append(urlParts, p)
				}
			}
			urlPath = "/" + strings.Join(urlParts, "/")
		}

		// Find layout chain from root app/ down to this route's directory
		layouts := r.findLayoutChain(filepath.Dir(path))

		route := &Route{
			Path:        urlPath,
			Pattern:     urlPath,
			PagePath:    path,
			LayoutPaths: layouts,
			Params:      params,
		}

		routes = append(routes, route)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan routes: %w", err)
	}

	// Sort routes: static routes first, then deep routes
	sort.Slice(routes, func(i, j int) bool {
		iHasParam := strings.Contains(routes[i].Path, ":")
		jHasParam := strings.Contains(routes[j].Path, ":")
		if iHasParam != jHasParam {
			return !iHasParam
		}
		return len(routes[i].Path) > len(routes[j].Path)
	})

	// Build authoritative Radix Route Trie
	trie := NewTrieRouter()
	for _, route := range routes {
		trie.Insert(route.Path, route)
	}

	r.Routes = routes
	r.Trie = trie
	return nil
}

// Match matches an incoming request path to a Route and extracts path params using the authoritative Radix Trie.
func (r *Router) Match(requestPath string) (*Route, map[string]string) {
	if r.Trie == nil {
		return nil, nil
	}

	// Normalize trailing slashes and clean path
	cleanPath := filepath.Clean(requestPath)
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	return r.Trie.Match(cleanPath)
}

// findLayoutChain traverses from appDir down to targetDir collecting layout.html files in order.
func (r *Router) findLayoutChain(targetDir string) []string {
	var layouts []string
	curr := targetDir
	appAbs, _ := filepath.Abs(r.AppDir)

	var dirChain []string
	for {
		dirChain = append([]string{curr}, dirChain...)
		currAbs, _ := filepath.Abs(curr)
		if currAbs == appAbs || curr == "." || curr == "/" {
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	for _, dir := range dirChain {
		layoutPath := filepath.Join(dir, "layout.html")
		if _, err := os.Stat(layoutPath); err == nil {
			layouts = append(layouts, layoutPath)
		}
	}

	return layouts
}
