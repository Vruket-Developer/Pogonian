package compiler

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

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

// Renderer compiles and renders filesystem routes with their layout hierarchies.
type Renderer struct {
	AppName string
	BaseDir string
}

// NewRenderer creates a new HTML template renderer.
func NewRenderer(appName, baseDir string) *Renderer {
	return &Renderer{
		AppName: appName,
		BaseDir: baseDir,
	}
}

// Render compiles the page and its parent layouts into a complete HTML response.
func (r *Renderer) Render(route *router.Route, params map[string]string, query map[string]string, data interface{}, csrfToken string) ([]byte, error) {
	// 1. Read page content
	pageBytes, err := os.ReadFile(route.PagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read page %s: %w", route.PagePath, err)
	}

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

	// Render the page template
	pageTmpl, err := template.New("page").Parse(string(pageBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse page template %s: %w", route.PagePath, err)
	}

	var pageBuf bytes.Buffer
	if err := pageTmpl.Execute(&pageBuf, ctx); err != nil {
		return nil, fmt.Errorf("failed to execute page template %s: %w", route.PagePath, err)
	}

	currentHTML := pageBuf.String()

	// 2. Wrap through the layout chain (from innermost to outermost)
	// If LayoutPaths is [root/layout.html, sub/layout.html],
	// sub/layout wraps page, then root/layout wraps sub/layout.
	for i := len(route.LayoutPaths) - 1; i >= 0; i-- {
		layoutPath := route.LayoutPaths[i]
		layoutBytes, err := os.ReadFile(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read layout %s: %w", layoutPath, err)
		}

		layoutTmpl, err := template.New(filepath.Base(layoutPath)).Parse(string(layoutBytes))
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

	// 3. Ensure runtime script tag is included
	if !strings.Contains(currentHTML, "/_pogn/runtime.js") && strings.Contains(currentHTML, "</body>") {
		currentHTML = strings.Replace(currentHTML, "</body>", "<script src=\"/_pogn/runtime.js\" defer></script>\n</body>", 1)
	}

	return []byte(currentHTML), nil
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
