package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pogonian/internal/architecture"
	"pogonian/tests/harness"
)

func TestAST_StructFieldExtraction(t *testing.T) {
	tempDir := t.TempDir()
	cartEntityFile := filepath.Join(tempDir, "entity.go")

	goCode := `package cart

import "time"

type CartItem struct {
	ID        int64     ` + "`json:\"id\"`" + `
	ItemName  string    ` + "`json:\"item_name\"`" + `
	Quantity  int       ` + "`json:\"quantity\"`" + `
	Price     float64   ` + "`json:\"price\"`" + `
	InStock   bool      ` + "`json:\"in_stock\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`
	if err := os.WriteFile(cartEntityFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("Failed to write test entity: %v", err)
	}

	structName, fields, err := architecture.ParseEntityFile(cartEntityFile)
	if err != nil {
		t.Fatalf("ParseEntityFile failed: %v", err)
	}

	if structName != "CartItem" {
		t.Errorf("Expected struct name 'CartItem', got '%s'", structName)
	}

	if len(fields) != 6 {
		t.Fatalf("Expected 6 struct fields, got %d", len(fields))
	}

	// Verify field mapping
	for _, f := range fields {
		switch f.Name {
		case "ItemName":
			if f.HTMLInputType != "text" || f.JSONTag != "item_name" {
				t.Errorf("ItemName mapped incorrectly: %+v", f)
			}
		case "Quantity":
			if f.HTMLInputType != "number" || f.SQLType != "INTEGER" {
				t.Errorf("Quantity mapped incorrectly: %+v", f)
			}
		case "Price":
			if f.HTMLInputType != "number" || f.Step != `step="0.01"` || f.SQLType != "REAL" {
				t.Errorf("Price mapped incorrectly: %+v", f)
			}
		case "InStock":
			if f.HTMLInputType != "checkbox" {
				t.Errorf("InStock mapped incorrectly: %+v", f)
			}
		}
	}

	// Verify HTML generation
	html := architecture.GenerateHTMLFromFields("cart", structName, fields)
	if !strings.Contains(html, `name="item_name"`) {
		t.Errorf("Expected item_name form input in HTML")
	}
	if !strings.Contains(html, `name="quantity"`) {
		t.Errorf("Expected quantity form input in HTML")
	}
	if !strings.Contains(html, `step="0.01"`) {
		t.Errorf("Expected step='0.01' for price in HTML")
	}
	if !strings.Contains(html, `>QUANTITY<`) {
		t.Errorf("Expected QUANTITY table header in HTML")
	}
}

func TestCLI_SchemaDrivenRouteGeneration(t *testing.T) {
	app := harness.NewTestRunner(t, "schema-driven-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Create custom entity in server/cart/entity.go
	cartDir := filepath.Join(projectDir, "server", "cart")
	_ = os.MkdirAll(cartDir, 0755)

	cartCode := `package cart

import "time"

type Cart struct {
	ID        int64     ` + "`json:\"id\"`" + `
	ItemName  string    ` + "`json:\"item_name\"`" + `
	Quantity  int       ` + "`json:\"quantity\"`" + `
	Price     float64   ` + "`json:\"price\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`
	_ = os.WriteFile(filepath.Join(cartDir, "entity.go"), []byte(cartCode), 0644)

	// 2. Run pogn add route /cart
	out, err := app.RunCLI("add", "route", "/cart", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn add route /cart failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Schema-Driven AST Generator") {
		t.Errorf("Expected Schema-Driven AST Generator in output, got:\n%s", out)
	}

	// 3. Verify generated app/cart/page.html contains the custom fields!
	pageHTML, err := os.ReadFile(filepath.Join(projectDir, "app", "cart", "page.html"))
	if err != nil {
		t.Fatalf("Failed to read generated app/cart/page.html: %v", err)
	}

	htmlStr := string(pageHTML)
	if !strings.Contains(htmlStr, `name="item_name"`) {
		t.Errorf("Expected custom field item_name in generated HTML")
	}
	if !strings.Contains(htmlStr, `name="quantity"`) {
		t.Errorf("Expected custom field quantity in generated HTML")
	}
	if !strings.Contains(htmlStr, `>PRICE<`) {
		t.Errorf("Expected PRICE column in generated HTML")
	}
}
