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
	regex       *regexp.Regexp
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
			// Convert windows backslashes
			normalized := strings.ReplaceAll(relDir, "\\", "/")
			// Convert [param] to :param
			parts := strings.Split(normalized, "/")
			var urlParts []string
			for _, p := range parts {
				if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
					paramName := p[1 : len(p)-1]
					urlParts = append(urlParts, ":"+paramName)
				} else {
					urlParts = append(urlParts, p)
				}
			}
			urlPath = "/" + strings.Join(urlParts, "/")
		}

		// Find layout chain from root app/ down to this route's directory
		layouts := r.findLayoutChain(filepath.Dir(path))

		// Build route regex pattern and param list
		pattern, params := buildRouteRegex(urlPath)
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			return fmt.Errorf("failed to compile route regex for %s: %w", urlPath, err)
		}

		route := &Route{
			Path:        urlPath,
			Pattern:     pattern,
			PagePath:    path,
			LayoutPaths: layouts,
			Params:      params,
			regex:       re,
		}

		routes = append(routes, route)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan routes: %w", err)
	}

	// Sort routes so static routes are checked before parameterized routes,
	// and deeper routes before shallower routes
	sort.Slice(routes, func(i, j int) bool {
		iHasParam := strings.Contains(routes[i].Path, ":")
		jHasParam := strings.Contains(routes[j].Path, ":")
		if iHasParam != jHasParam {
			return !iHasParam // static routes first
		}
		return len(routes[i].Path) > len(routes[j].Path)
	})

	// Build high-performance Radix Route Trie
	trie := NewTrieRouter()
	for _, route := range routes {
		trie.Insert(route.Path, route)
	}

	r.Routes = routes
	r.Trie = trie
	return nil
}

// Match matches an incoming request path to a Route and extracts path params using the Radix Trie.
func (r *Router) Match(requestPath string) (*Route, map[string]string) {
	// Normalize path
	cleanPath := filepath.Clean(requestPath)
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	// 1. Fast Trie lookup
	if r.Trie != nil {
		if route, params := r.Trie.Match(cleanPath); route != nil {
			return route, params
		}
	}

	// 2. Fallback regex matcher
	for _, route := range r.Routes {
		matches := route.regex.FindStringSubmatch(cleanPath)
		if len(matches) > 0 {
			params := make(map[string]string)
			for idx, name := range route.Params {
				if idx+1 < len(matches) {
					params[name] = matches[idx+1]
				}
			}
			return route, params
		}
	}

	return nil, nil
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

	for _, d := range dirChain {
		layoutCandidate := filepath.Join(d, "layout.html")
		if _, err := os.Stat(layoutCandidate); err == nil {
			layouts = append(layouts, layoutCandidate)
		}
	}

	return layouts
}

func buildRouteRegex(urlPath string) (string, []string) {
	if urlPath == "/" {
		return "/", nil
	}

	segments := strings.Split(strings.Trim(urlPath, "/"), "/")
	var regexSegments []string
	var paramNames []string

	for _, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			paramName := strings.TrimPrefix(seg, ":")
			paramNames = append(paramNames, paramName)
			regexSegments = append(regexSegments, `([^/]+)`)
		} else {
			regexSegments = append(regexSegments, regexp.QuoteMeta(seg))
		}
	}

	pattern := "/" + strings.Join(regexSegments, "/")
	return pattern, paramNames
}
