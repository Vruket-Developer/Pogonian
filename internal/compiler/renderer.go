package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pogonian/internal/router"
)

// RenderContext holds data available to HTML templates.
type RenderContext struct {
	AppName   string
	Title     string
	Params    map[string]string
	Query     map[string]string
	Data      interface{}
	CSRFToken string
	Content   template.HTML
	Meta      map[string]interface{}
}

// CachedTemplate holds a precompiled template chain for a route.
type CachedTemplate struct {
	PageTemplate    *template.Template
	LayoutTemplates []*template.Template
	LastModified    time.Time
}

// Renderer compiles and renders filesystem routes with their layout hierarchies.
type Renderer struct {
	AppName     string
	BaseDir     string
	DevMode     bool
	cacheMu     sync.RWMutex
	cache       map[string]*CachedTemplate
	customFuncs template.FuncMap
}

// NewRenderer creates a new HTML template renderer.
func NewRenderer(appName, baseDir string) *Renderer {
	return NewRendererWithMode(appName, baseDir, true)
}

// NewRendererWithMode creates a new HTML template renderer with dev/prod mode control.
func NewRendererWithMode(appName, baseDir string, devMode bool) *Renderer {
	r := &Renderer{
		AppName: appName,
		BaseDir: baseDir,
		DevMode: devMode,
		cache:   make(map[string]*CachedTemplate),
		customFuncs: template.FuncMap{
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
			"json": func(v interface{}) string {
				b, _ := json.Marshal(v)
				return string(b)
			},
			"formatDate": func(t time.Time, layout string) string {
				if layout == "" {
					layout = "Jan 02, 2006"
				}
				return t.Format(layout)
			},
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
		},
	}
	return r
}

// PrecompileAll walks all routes and precompiles their templates into memory (for production startup).
func (r *Renderer) PrecompileAll(routes []*router.Route) error {
	for _, rt := range routes {
		if err := r.precompileRoute(rt); err != nil {
			return fmt.Errorf("failed to precompile route %s: %w", rt.Path, err)
		}
	}
	return nil
}

func (r *Renderer) precompileRoute(route *router.Route) error {
	// Parse page with components
	pageBytes, err := os.ReadFile(route.PagePath)
	if err != nil {
		return fmt.Errorf("failed to read page %s: %w", route.PagePath, err)
	}

	pageTmpl := template.New("page").Funcs(r.customFuncs)
	pageTmpl = r.loadComponents(pageTmpl)
	pageTmpl, err = pageTmpl.Parse(string(pageBytes))
	if err != nil {
		return fmt.Errorf("failed to parse page template %s: %w", route.PagePath, err)
	}

	// Parse layout chain with components
	layoutTmpls := make([]*template.Template, 0, len(route.LayoutPaths))
	for _, layoutPath := range route.LayoutPaths {
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return fmt.Errorf("failed to read layout %s: %w", layoutPath, err)
		}

		lt := template.New(filepath.Base(layoutPath)).Funcs(r.customFuncs)
		lt = r.loadComponents(lt)
		lt, err = lt.Parse(string(layoutBytes))
		if err != nil {
			return fmt.Errorf("failed to parse layout template %s: %w", layoutPath, err)
		}
		layoutTmpls = append(layoutTmpls, lt)
	}

	r.cacheMu.Lock()
	r.cache[route.Path] = &CachedTemplate{
		PageTemplate:    pageTmpl,
		LayoutTemplates: layoutTmpls,
		LastModified:    time.Now(),
	}
	r.cacheMu.Unlock()

	return nil
}

// Render compiles the page and its parent layouts into a complete HTML response.
func (r *Renderer) Render(route *router.Route, params map[string]string, query map[string]string, data interface{}, csrfToken string) ([]byte, error) {
	pageTitle := derivePageTitle(route.Path, r.AppName)

	ctx := RenderContext{
		AppName:   r.AppName,
		Title:     pageTitle,
		Params:    params,
		Query:     query,
		Data:      data,
		CSRFToken: csrfToken,
		Meta:      make(map[string]interface{}),
	}

	// In production, use precompiled cache if present
	if !r.DevMode {
		r.cacheMu.RLock()
		cached, exists := r.cache[route.Path]
		r.cacheMu.RUnlock()

		if exists && cached != nil {
			return r.executeCached(cached, ctx)
		}
	}

	// Dynamic parse (in dev mode or cache miss)
	pageBytes, err := os.ReadFile(route.PagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read page %s: %w", route.PagePath, err)
	}

	pageTmpl := template.New("page").Funcs(r.customFuncs)
	pageTmpl = r.loadComponents(pageTmpl)
	pageTmpl, err = pageTmpl.Parse(string(pageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse page template %s: %w", route.PagePath, err)
	}

	var pageBuf bytes.Buffer
	if err := pageTmpl.Execute(&pageBuf, ctx); err != nil {
		return nil, fmt.Errorf("failed to execute page template %s: %w", route.PagePath, err)
	}

	currentHTML := pageBuf.String()

	// Wrap through the layout chain (from innermost to outermost)
	for i := len(route.LayoutPaths) - 1; i >= 0; i-- {
		layoutPath := route.LayoutPaths[i]
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read layout %s: %w", layoutPath, err)
		}

		layoutTmpl := template.New(filepath.Base(layoutPath)).Funcs(r.customFuncs)
		layoutTmpl = r.loadComponents(layoutTmpl)
		layoutTmpl, err = layoutTmpl.Parse(string(layoutBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to parse layout template %s: %w", layoutPath, err)
		}

		ctx.Content = template.HTML(currentHTML)

		var layoutBuf bytes.Buffer
		if err := layoutTmpl.Execute(&layoutBuf, ctx); err != nil {
			return nil, fmt.Errorf("failed to execute layout template %s: %w", layoutPath, err)
		}

		currentHTML = layoutBuf.String()
	}

	// Inject Client Runtimes cleanly based on environment
	currentHTML = r.injectScripts(currentHTML)

	return []byte(currentHTML), nil
}

func (r *Renderer) executeCached(cached *CachedTemplate, ctx RenderContext) ([]byte, error) {
	var pageBuf bytes.Buffer
	if err := cached.PageTemplate.Execute(&pageBuf, ctx); err != nil {
		return nil, err
	}

	currentHTML := pageBuf.String()

	for i := len(cached.LayoutTemplates) - 1; i >= 0; i-- {
		lt := cached.LayoutTemplates[i]
		ctx.Content = template.HTML(currentHTML)

		var layoutBuf bytes.Buffer
		if err := lt.Execute(&layoutBuf, ctx); err != nil {
			return nil, err
		}
		currentHTML = layoutBuf.String()
	}

	currentHTML = r.injectScripts(currentHTML)
	return []byte(currentHTML), nil
}

func (r *Renderer) loadComponents(t *template.Template) *template.Template {
	compDir := filepath.Join(r.BaseDir, "components")
	entries, err := os.ReadDir(compDir)
	if err != nil {
		return t
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			path := filepath.Join(compDir, entry.Name())
			data, err := os.ReadFile(path)
			if err == nil {
				// Parse template definition
				t, _ = t.Parse(string(data))
				// Also parse with names "components/xxx.html" and "xxx"
				relName := "components/" + entry.Name()
				shortName := strings.TrimSuffix(entry.Name(), ".html")
				t, _ = t.New(relName).Parse(string(data))
				t, _ = t.New(shortName).Parse(string(data))
			}
		}
	}
	return t
}

func (r *Renderer) injectScripts(html string) string {
	if !strings.Contains(html, "</body>") {
		return html
	}

	if r.DevMode {
		if !strings.Contains(html, "/__pogn/devbar.js") {
			html = strings.Replace(html, "</body>", "<script src=\"/__pogn/devbar.js\" defer></script>\n</body>", 1)
		}
		if !strings.Contains(html, "/_pogn/runtime.js") {
			html = strings.Replace(html, "</body>", "<script src=\"/_pogn/runtime.js\" defer></script>\n</body>", 1)
		}
	} else {
		if !strings.Contains(html, "/_pogn/runtime.js") {
			html = strings.Replace(html, "</body>", "<script src=\"/_pogn/runtime.js\" defer></script>\n</body>", 1)
		}
	}

	return html
}

func derivePageTitle(routePath, appName string) string {
	if routePath == "/" {
		return "Home"
	}
	clean := strings.Trim(routePath, "/")
	parts := strings.Split(clean, "/")
	last := parts[len(parts)-1]
	if after, ok := strings.CutPrefix(last, ":"); ok {
		last = after
	}
	if len(last) > 0 {
		return strings.ToUpper(last[:1]) + last[1:]
	}
	return appName
}
