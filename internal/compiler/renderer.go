package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	knownComps := r.discoverKnownComponents()

	pageBytes, err := os.ReadFile(route.PagePath)
	if err != nil {
		return fmt.Errorf("failed to read page %s: %w", route.PagePath, err)
	}

	rawPage := r.preprocessComponentTags(string(pageBytes), knownComps)
	pageTmpl := template.New("page").Funcs(r.customFuncs)
	pageTmpl = r.loadComponents(pageTmpl)
	pageTmpl, err = pageTmpl.Parse(rawPage)
	if err != nil {
		return fmt.Errorf("failed to parse page template %s: %w", route.PagePath, err)
	}

	layoutTmpls := make([]*template.Template, 0, len(route.LayoutPaths))
	for _, layoutPath := range route.LayoutPaths {
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return fmt.Errorf("failed to read layout %s: %w", layoutPath, err)
		}

		rawLayout := r.preprocessComponentTags(string(layoutBytes), knownComps)
		lt := template.New(filepath.Base(layoutPath)).Funcs(r.customFuncs)
		lt = r.loadComponents(lt)
		lt, err = lt.Parse(rawLayout)
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

	if !r.DevMode {
		r.cacheMu.RLock()
		cached, exists := r.cache[route.Path]
		r.cacheMu.RUnlock()

		if exists && cached != nil {
			return r.executeCached(cached, ctx)
		}
	}

	knownComps := r.discoverKnownComponents()

	pageBytes, err := os.ReadFile(route.PagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read page %s: %w", route.PagePath, err)
	}

	rawPage := r.preprocessComponentTags(string(pageBytes), knownComps)
	pageTmpl := template.New("page").Funcs(r.customFuncs)
	pageTmpl = r.loadComponents(pageTmpl)
	pageTmpl, err = pageTmpl.Parse(rawPage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page template %s: %w", route.PagePath, err)
	}

	var pageBuf bytes.Buffer
	if err := pageTmpl.Execute(&pageBuf, ctx); err != nil {
		return nil, fmt.Errorf("failed to execute page template %s: %w", route.PagePath, err)
	}

	currentHTML := pageBuf.String()

	for i := len(route.LayoutPaths) - 1; i >= 0; i-- {
		layoutPath := route.LayoutPaths[i]
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read layout %s: %w", layoutPath, err)
		}

		rawLayout := r.preprocessComponentTags(string(layoutBytes), knownComps)
		layoutTmpl := template.New(filepath.Base(layoutPath)).Funcs(r.customFuncs)
		layoutTmpl = r.loadComponents(layoutTmpl)
		layoutTmpl, err = layoutTmpl.Parse(rawLayout)
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

var pognTagRegex = regexp.MustCompile(`(?i)<(?:pogn:|component:)([a-zA-Z0-9_\-]+)\s*(?:/>|></(?:pogn:|component:)[a-zA-Z0-9_\-]+>)`)
var compNameTagRegex = regexp.MustCompile(`(?i)<component\s+name=["']([a-zA-Z0-9_\-]+)["']\s*(?:/>|></component>)`)
var selfClosingTagRegex = regexp.MustCompile(`<([a-zA-Z0-9_\-]+)\s*/>`)
var customElementRegex = regexp.MustCompile(`<([A-Z][a-zA-Z0-9_]*)\s*(?:/>|></[A-Z][a-zA-Z0-9_]*>)`)

// preprocessComponentTags automatically expands tags like <footer />, <Footer />, <pogn:footer />, and <component name="footer" /> into {{ template "footer" . }}
func (r *Renderer) preprocessComponentTags(html string, knownComponents map[string]bool) string {
	// 1. Expand <pogn:footer /> or <component:footer />
	html = pognTagRegex.ReplaceAllString(html, `{{ template "$1" . }}`)

	// 2. Expand <component name="footer" />
	html = compNameTagRegex.ReplaceAllString(html, `{{ template "$1" . }}`)

	// 3. Expand self-closing HTML tags like <footer />, <header />, <sidebar /> when component exists
	html = selfClosingTagRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := selfClosingTagRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			tagName := strings.ToLower(submatches[1])
			if knownComponents[tagName] || tagName == "footer" || tagName == "header" || tagName == "sidebar" {
				return fmt.Sprintf(`{{ template "%s" . }}`, tagName)
			}
		}
		return match
	})

	// 4. Expand PascalCase <Footer /> or <Header /> or <Footer></Footer>
	html = customElementRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := customElementRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			tagName := submatches[1]
			lowerName := strings.ToLower(tagName)
			if knownComponents[lowerName] || lowerName == "footer" || lowerName == "header" || lowerName == "sidebar" {
				return fmt.Sprintf(`{{ template "%s" . }}`, lowerName)
			}
		}
		return match
	})

	return html
}

// discoverKnownComponents discovers all component names across app/components, root components/, and module/ folders.
func (r *Renderer) discoverKnownComponents() map[string]bool {
	comps := make(map[string]bool)

	dirsToScan := []string{
		filepath.Join(r.BaseDir, "components"),
		filepath.Join(filepath.Dir(r.BaseDir), "components"),
		filepath.Join(r.BaseDir, "components", "module"),
		filepath.Join(filepath.Dir(r.BaseDir), "components", "module"),
	}

	for _, d := range dirsToScan {
		_ = filepath.WalkDir(d, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
				base := strings.TrimSuffix(strings.ToLower(entry.Name()), ".html")
				comps[base] = true
			}
			return nil
		})
	}

	return comps
}

// loadComponents imports and registers component templates from /app/components/, /components/, and subfolders like /module/.
func (r *Renderer) loadComponents(t *template.Template) *template.Template {
	dirsToScan := []string{
		filepath.Join(r.BaseDir, "components"),
		filepath.Join(filepath.Dir(r.BaseDir), "components"),
	}

	visited := make(map[string]bool)

	for _, compDir := range dirsToScan {
		if _, err := os.Stat(compDir); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(compDir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				return nil
			}

			if visited[path] {
				return nil
			}
			visited[path] = true

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			content := string(data)
			rel, _ := filepath.Rel(compDir, path)
			relUnix := filepath.ToSlash(rel)

			shortName := strings.TrimSuffix(entry.Name(), ".html")
			lowerShort := strings.ToLower(shortName)

			// Parse base template
			t, _ = t.Parse(content)

			// Register aliases:
			// 1. "footer"
			t, _ = t.New(lowerShort).Parse(content)
			t, _ = t.New(shortName).Parse(content)

			// 2. "components/footer.html" or "components/module/footer.html"
			t, _ = t.New("components/" + relUnix).Parse(content)

			// 3. "module/footer" or "module/footer.html"
			relNoExt := strings.TrimSuffix(relUnix, ".html")
			t, _ = t.New(relUnix).Parse(content)
			t, _ = t.New(relNoExt).Parse(content)

			return nil
		})
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
