package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// EntityField describes a single struct field extracted from Go AST.
type EntityField struct {
	Name          string
	Type          string
	JSONTag       string
	HTMLInputType string
	Step          string
	SQLType       string
	IsFormInput   bool
}

// ParseEntityFile parses a Go entity file and extracts its struct definition and fields.
func ParseEntityFile(filePath string) (string, []EntityField, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse entity file %s: %w", filePath, err)
	}

	var structName string
	var fields []EntityField

	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName = ts.Name.Name

		for _, f := range st.Fields.List {
			if len(f.Names) == 0 {
				continue
			}

			fieldName := f.Names[0].Name
			fieldType := extractTypeString(f.Type)
			jsonTag := strings.ToLower(fieldName)

			if f.Tag != nil {
				tagVal := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
				if jsonVal := tagVal.Get("json"); jsonVal != "" {
					jsonTag = strings.Split(jsonVal, ",")[0]
				}
			}

			htmlType := "text"
			step := ""
			sqlType := "TEXT"
			isForm := true

			switch fieldType {
			case "int", "int64", "int32", "uint", "uint64":
				htmlType = "number"
				sqlType = "INTEGER"
			case "float64", "float32":
				htmlType = "number"
				step = `step="0.01"`
				sqlType = "REAL"
			case "bool":
				htmlType = "checkbox"
				sqlType = "BOOLEAN"
			case "time.Time":
				htmlType = "datetime-local"
				sqlType = "DATETIME"
			}

			if strings.EqualFold(fieldName, "id") || strings.EqualFold(fieldName, "createdat") || strings.EqualFold(fieldName, "created_at") {
				isForm = false
			}

			fields = append(fields, EntityField{
				Name:          fieldName,
				Type:          fieldType,
				JSONTag:       jsonTag,
				HTMLInputType: htmlType,
				Step:          step,
				SQLType:       sqlType,
				IsFormInput:   isForm,
			})
		}

		return false
	})

	if structName == "" {
		return "", nil, fmt.Errorf("no struct definition found in %s", filePath)
	}

	return structName, fields, nil
}

// GenerateHTMLFromFields synthesizes an HTML page tailored to the struct's fields.
func GenerateHTMLFromFields(moduleName, structName string, fields []EntityField) string {
	mod := strings.ToLower(moduleName)
	pascal := strings.ToUpper(mod[:1]) + mod[1:]

	var formInputs strings.Builder
	for _, f := range fields {
		if !f.IsFormInput {
			continue
		}

		label := formatFieldLabel(f.Name)
		stepAttr := ""
		if f.Step != "" {
			stepAttr = " " + f.Step
		}

		if strings.Contains(strings.ToLower(f.Name), "description") || strings.Contains(strings.ToLower(f.Name), "content") || strings.Contains(strings.ToLower(f.Name), "notes") {
			formInputs.WriteString(fmt.Sprintf(`                <div>
                    <label style="display: block; margin-bottom: 0.35rem; font-weight: 500; font-size: 0.9rem;">%s</label>
                    <textarea name="%s" rows="3" placeholder="Enter %s..."
                              style="width: 100%%; padding: 0.65rem 0.85rem; border-radius: 6px; border: 1px solid var(--border-color); background: var(--bg-surface-elevated); color: var(--text-primary); font-size: 0.95rem;"></textarea>
                </div>
`, label, f.JSONTag, strings.ToLower(label)))
		} else if f.HTMLInputType == "checkbox" {
			formInputs.WriteString(fmt.Sprintf(`                <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <input type="checkbox" name="%s" id="%s" value="true">
                    <label for="%s" style="font-weight: 500; font-size: 0.9rem;">%s</label>
                </div>
`, f.JSONTag, f.JSONTag, f.JSONTag, label))
		} else {
			formInputs.WriteString(fmt.Sprintf(`                <div>
                    <label style="display: block; margin-bottom: 0.35rem; font-weight: 500; font-size: 0.9rem;">%s</label>
                    <input type="%s"%s name="%s" placeholder="Enter %s..." required
                           style="width: 100%%; padding: 0.65rem 0.85rem; border-radius: 6px; border: 1px solid var(--border-color); background: var(--bg-surface-elevated); color: var(--text-primary); font-size: 0.95rem;">
                </div>
`, label, f.HTMLInputType, stepAttr, f.JSONTag, strings.ToLower(label)))
		}
	}

	var tableHeaders strings.Builder
	var tableCells strings.Builder
	colCount := 0

	for _, f := range fields {
		colCount++
		label := strings.ToUpper(formatFieldLabel(f.Name))
		tableHeaders.WriteString(fmt.Sprintf(`                            <th style="padding: 0.75rem 0.5rem;">%s</th>`+"\n", label))

		if strings.EqualFold(f.Name, "id") {
			tableCells.WriteString(`                                <td style="padding: 0.75rem 0.5rem; color: var(--text-muted);">#{{ .ID }}</td>` + "\n")
		} else if strings.EqualFold(f.Name, "price") || strings.EqualFold(f.Name, "totalprice") || strings.EqualFold(f.Name, "amount") {
			tableCells.WriteString(fmt.Sprintf(`                                <td style="padding: 0.75rem 0.5rem; font-weight: 600; color: #34d399;">${{ .%s }}</td>`+"\n", f.Name))
		} else {
			tableCells.WriteString(fmt.Sprintf(`                                <td style="padding: 0.75rem 0.5rem; color: var(--text-secondary);">{{ .%s }}</td>`+"\n", f.Name))
		}
	}

	return fmt.Sprintf(`<section class="content-section">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem;">
        <div>
            <h1>%s</h1>
            <p style="color: var(--text-secondary);">Schema-driven Pogonian Flow: <code>Page &rarr; Form Action &rarr; Handler &rarr; Service &rarr; Repository &rarr; SQLite</code></p>
        </div>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1.5fr; gap: 2rem;">
        <!-- Form Action -->
        <div class="card">
            <h3>Add New %s</h3>
            <p style="color: var(--text-muted); margin-bottom: 1.25rem; font-size: 0.85rem;">Generated from Go struct <code>server/%s/entity.go</code></p>
            
            <form action="/%s" method="post" style="display: flex; flex-direction: column; gap: 1rem;">
                <input type="hidden" name="_csrf" value="{{ .CSRFToken }}">
%s
                <button type="submit" class="btn btn-primary" style="width: 100%%; margin-top: 0.5rem;">
                    + Create %s
                </button>
            </form>
        </div>

        <!-- Inventory / Collection Table -->
        <div class="card">
            <h3>%s Records</h3>
            <div style="margin-top: 1rem; overflow-x: auto;">
                <table style="width: 100%%; border-collapse: collapse; text-align: left;">
                    <thead>
                        <tr style="border-bottom: 1px solid var(--border-color); color: var(--text-muted); font-size: 0.85rem;">
%s
                        </tr>
                    </thead>
                    <tbody>
                        {{ if and .Data .Data.Items }}
                            {{ range .Data.Items }}
                            <tr style="border-bottom: 1px solid var(--border-color);">
%s
                            </tr>
                            {{ end }}
                        {{ else }}
                            <tr>
                                <td colspan="%d" style="padding: 2rem; text-align: center; color: var(--text-muted);">
                                    No %s records found. Use the form on the left to add one!
                                </td>
                            </tr>
                        {{ end }}
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</section>
`, pascal, pascal, mod, mod, formInputs.String(), pascal, pascal, tableHeaders.String(), tableCells.String(), colCount, pascal)
}

func extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return extractTypeString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + extractTypeString(t.X)
	default:
		return "string"
	}
}

func formatFieldLabel(name string) string {
	var result []rune
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	return string(result)
}
