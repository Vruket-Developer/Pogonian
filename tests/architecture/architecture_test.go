package architecture_test

import (
	"strings"
	"testing"

	"pogonian/internal/architecture"
	"pogonian/pkg/ir"
	"pogonian/tests/harness"
)

func TestArchitecture_GraphAndMermaid(t *testing.T) {
	g := architecture.NewGraph()

	g.AddNode("route:/products", "/products", architecture.NodeRoute)
	g.AddNode("handler:products.Handler", "ProductHandler", architecture.NodeHandler)
	g.AddNode("service:products.Service", "ProductService", architecture.NodeService)
	g.AddNode("repository:products.Repository", "ProductRepository", architecture.NodeRepository)
	g.AddNode("database:sqlite", "SQLite", architecture.NodeDatabase)

	g.AddEdge("route:/products", "handler:products.Handler", "dispatches")
	g.AddEdge("handler:products.Handler", "service:products.Service", "calls")
	g.AddEdge("service:products.Service", "repository:products.Repository", "calls")
	g.AddEdge("repository:products.Repository", "database:sqlite", "queries")

	mermaid := g.ToMermaid()
	if !strings.Contains(mermaid, "flowchart TD") {
		t.Errorf("Expected mermaid header, got:\n%s", mermaid)
	}
	if !strings.Contains(mermaid, "ProductHandler") || !strings.Contains(mermaid, "ProductService") {
		t.Errorf("Expected nodes in mermaid graph")
	}

	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("Expected 0 cycles in clean flow, got: %v", cycles)
	}
}

func TestArchitecture_CycleDetection(t *testing.T) {
	g := architecture.NewGraph()

	g.AddNode("svc:A", "ServiceA", architecture.NodeService)
	g.AddNode("svc:B", "ServiceB", architecture.NodeService)

	g.AddEdge("svc:A", "svc:B", "calls")
	g.AddEdge("svc:B", "svc:A", "calls") // Circular!

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Fatalf("Expected circular dependency detected, got 0 cycles")
	}
}

func TestArchitecture_LayerViolationDetection(t *testing.T) {
	app := &ir.Application{
		Handlers: []*ir.HandlerIR{
			{
				Name:           "BadHandler",
				File:           "server/bad/handler.go",
				DirectDBAccess: true, // Layer Violation!
			},
		},
	}

	g := architecture.NewGraph()
	validator := architecture.NewValidator(true)
	violations := validator.Validate(app, g)

	if len(violations) == 0 {
		t.Fatalf("Expected layer violation for direct DB access, but got 0")
	}

	v := violations[0]
	if v.Severity != "ERROR" || !strings.Contains(v.Message, "database") {
		t.Errorf("Expected DB layer violation error, got: %v", v)
	}
}

func TestArchitecture_AnalyzeDefaultProject(t *testing.T) {
	runner := harness.NewTestRunner(t, "analyze-test-app")
	err := runner.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	analyzer, err := architecture.NewAnalyzer(runner.AppPath())
	if err != nil {
		t.Fatalf("Failed to initialize analyzer: %v", err)
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if len(appIR.Routes) < 3 {
		t.Errorf("Expected at least 3 routes, got %d", len(appIR.Routes))
	}
	if len(graph.Nodes) < 4 {
		t.Errorf("Expected at least 4 nodes in graph, got %d", len(graph.Nodes))
	}

	validator := architecture.NewValidator(true)
	violations := validator.Validate(appIR, graph)
	for _, v := range violations {
		if v.Severity == "ERROR" {
			t.Errorf("Default template has architectural error: %v", v)
		}
	}
}
