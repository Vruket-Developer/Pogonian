package compiler

import (
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"pogonian/pkg/ir"
	"pogonian/runtime"
)

func init() {
	RegisterLowerer(&WebLowerer{})
}

// WebLowerer compiles the universal application IR into server-rendered HTML, HTMX attributes, and runtime.js.
type WebLowerer struct{}

func (w *WebLowerer) Target() ir.TargetPlatform {
	return ir.TargetWeb
}

func (w *WebLowerer) Lower(app *ir.Application, projectDir string) (*LoweredArtifact, error) {
	files := make(map[string][]byte)

	// 1. Embed Pogonian Soft Navigation Runtime
	files["public/runtime.js"] = runtime.RuntimeJS

	// 2. Generate or compile HTML pages from routes/UI trees
	var routeManifest []string
	for _, r := range app.Routes {
		routeManifest = append(routeManifest, fmt.Sprintf("%s ➔ %s", r.Path, r.Handler))
		if r.UIRoot != nil {
			htmlContent := lowerUIToHTML(r.UIRoot)
			pageName := strings.Trim(r.Path, "/")
			if pageName == "" {
				pageName = "index"
			}
			files[fmt.Sprintf("app/%s.html", pageName)] = []byte(htmlContent)
		}
	}

	// 3. Web Route Registration Manifest
	manifestContent := fmt.Sprintf("// Auto-generated Web Route Registry for Pogonian\n// Routes: %d\n%s\n",
		len(app.Routes), strings.Join(routeManifest, "\n"))
	files["server/web_routes.go"] = []byte(manifestContent)

	summary := fmt.Sprintf("Web target lowered successfully: %d routes, HTMX-compatible server HTML, and 3.2KB soft-nav runtime.", len(app.Routes))

	return &LoweredArtifact{
		Target:             ir.TargetWeb,
		AppName:            app.Name,
		EntrypointFile:     filepath.Join("app", "layout.html"),
		Files:              files,
		Summary:            summary,
		NativeCapabilities: []string{"network", "keyboard", "mouse", "touch", "webauthn"},
	}, nil
}

func lowerUIToHTML(elem *ir.UIElementIR) string {
	if elem == nil {
		return ""
	}

	var sb strings.Builder
	escapedText := html.EscapeString(elem.Text)

	switch elem.Kind {
	case ir.UIElementStackVertical:
		sb.WriteString(`<div class="pogn-stack-vertical">`)
	case ir.UIElementStackHorizontal:
		sb.WriteString(`<div class="pogn-stack-horizontal">`)
	case ir.UIElementHeading:
		sb.WriteString(`<h2>`)
		sb.WriteString(escapedText)
		sb.WriteString(`</h2>`)
	case ir.UIElementText:
		sb.WriteString(`<p>`)
		sb.WriteString(escapedText)
		sb.WriteString(`</p>`)
	case ir.UIElementButton:
		sb.WriteString(`<button class="btn btn-primary" hx-boost="true">`)
		sb.WriteString(escapedText)
		sb.WriteString(`</button>`)
	case ir.UIElementInput:
		sb.WriteString(fmt.Sprintf(`<input class="input-field" placeholder="%s" />`, escapedText))
	case ir.UIElementCard:
		sb.WriteString(`<div class="card pogn-card">`)
	default:
		elemID := html.EscapeString(elem.ID)
		sb.WriteString(fmt.Sprintf(`<div id="%s" class="pogn-container">`, elemID))
	}

	for _, child := range elem.Children {
		sb.WriteString(lowerUIToHTML(child))
	}

	switch elem.Kind {
	case ir.UIElementStackVertical, ir.UIElementStackHorizontal, ir.UIElementCard, ir.UIElementContainer:
		sb.WriteString(`</div>`)
	}

	return sb.String()
}
