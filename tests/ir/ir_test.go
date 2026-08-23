package ir_test

import (
	"strings"
	"testing"

	"pogonian/internal/ir"
)

func TestIR_Serialization(t *testing.T) {
	app := &ir.Application{
		Name:         "ecommerce-store",
		Version:      "1.0.0",
		Framework:    "pogonian",
		Architecture: "modular-monolith",
		Routes: []*ir.RouteIR{
			{Path: "/", Methods: []string{"GET"}},
			{Path: "/products", Methods: []string{"GET", "POST"}, Handler: "ProductHandler"},
		},
		Entities: []*ir.EntityIR{
			{
				Name: "Product",
				Fields: []ir.FieldIR{
					{Name: "ID", Type: "int64", Required: true, PK: true},
					{Name: "Name", Type: "string", Required: true},
					{Name: "Price", Type: "float64", Required: true},
				},
			},
		},
		Handlers: []*ir.HandlerIR{
			{Name: "ProductHandler", Package: "products", CallsServices: []string{"ProductService"}},
		},
		Services: []*ir.ServiceIR{
			{Name: "ProductService", Package: "products", CallsRepos: []string{"ProductRepository"}},
		},
		Repositories: []*ir.RepositoryIR{
			{Name: "ProductRepository", Package: "products", TargetTable: "products"},
		},
		Database: &ir.DatabaseIR{
			Driver: "sqlite",
		},
	}

	// 1. Test JSON Serialization
	jsonData, err := app.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if !strings.Contains(string(jsonData), "ecommerce-store") {
		t.Errorf("Expected app name in JSON output")
	}

	// 2. Test JSON Deserialization
	recovered, err := ir.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}
	if recovered.Name != app.Name || len(recovered.Routes) != 2 {
		t.Errorf("Recovered IR does not match original: %v", recovered)
	}

	// 3. Test YAML Serialization
	yamlData, err := app.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}
	if !strings.Contains(string(yamlData), "architecture: modular-monolith") {
		t.Errorf("Expected YAML output")
	}
}
