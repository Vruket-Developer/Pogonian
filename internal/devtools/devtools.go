package devtools

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"pogonian/internal/architecture"
)

//go:embed devtools.html
var DevToolsHTML []byte

//go:embed devbar.js
var DevBarJS []byte

// DevToolsContext holds data passed to the DevTools HTML dashboard.
type DevToolsContext struct {
	TotalRequests int
	RoutesCount   int
	MermaidGraph  template.HTML
	Metrics       []*RequestMetric
	AISpecYAML    string
}

// ServeDevTools renders the live architecture DevTools dashboard.
func ServeDevTools(projectDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		analyzer, err := architecture.NewAnalyzer(projectDir)
		mermaid := "flowchart TD\n  app([\"App\"])"
		routesCount := 0

		if err == nil {
			appIR, graph, err := analyzer.Analyze()
			if err == nil {
				mermaid = graph.ToMermaid()
				routesCount = len(appIR.Routes)
			}
		}

		aiSpecPath := filepath.Join(projectDir, ".pogonian", "architecture", "app.ai")
		aiData, _ := os.ReadFile(aiSpecPath)

		metrics := GlobalRecorder.GetMetrics()

		ctx := DevToolsContext{
			TotalRequests: len(metrics),
			RoutesCount:   routesCount,
			MermaidGraph:  template.HTML(mermaid),
			Metrics:       metrics,
			AISpecYAML:    string(aiData),
		}

		tmpl, err := template.New("devtools").Parse(string(DevToolsHTML))
		if err != nil {
			http.Error(w, "Failed to load devtools template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			http.Error(w, "Failed to render devtools: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	}
}

// ServeDevBar serves the in-page floating DevBar script.
func ServeDevBar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(DevBarJS)
	}
}
