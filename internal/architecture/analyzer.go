package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"pogonian/internal/config"
	"pogonian/internal/ir"
	"pogonian/internal/router"
)

// Analyzer inspects project source code and generates IR and dependency graphs.
type Analyzer struct {
	ProjectDir string
	Config     *config.Config
}

// NewAnalyzer creates a new project architecture analyzer.
func NewAnalyzer(projectDir string) (*Analyzer, error) {
	cfg, err := config.Load(projectDir)
	if err != nil {
		cfg = config.DefaultConfig(filepath.Base(projectDir))
	}
	return &Analyzer{
		ProjectDir: projectDir,
		Config:     cfg,
	}, nil
}

// Analyze performs static AST analysis and returns both the Pogonian IR and the DependencyGraph.
func (a *Analyzer) Analyze() (*ir.Application, *DependencyGraph, error) {
	appIR := &ir.Application{
		Name:         a.Config.Name,
		Version:      a.Config.Version,
		Framework:    "pogonian",
		Architecture: a.Config.Architecture.Style,
		Routes:       make([]*ir.RouteIR, 0),
		Entities:     make([]*ir.EntityIR, 0),
		Handlers:     make([]*ir.HandlerIR, 0),
		Services:     make([]*ir.ServiceIR, 0),
		Repositories: make([]*ir.RepositoryIR, 0),
		Database: &ir.DatabaseIR{
			Driver: a.Config.Database.Driver,
			Tables: make([]ir.TableIR, 0),
		},
	}

	graph := NewGraph()

	// Add Database Node
	dbNodeID := "database:" + a.Config.Database.Driver
	graph.AddNode(dbNodeID, fmt.Sprintf("Database (%s)", a.Config.Database.Driver), NodeDatabase)

	// 1. Scan Frontend Routes
	appDir := filepath.Join(a.ProjectDir, "app")
	if r, err := router.New(appDir); err == nil {
		for _, rt := range r.Routes {
			routeIR := &ir.RouteIR{
				Path:        rt.Path,
				Methods:     []string{"GET"},
				PagePath:    rt.PagePath,
				LayoutChain: rt.LayoutPaths,
				Params:      rt.Params,
			}
			appIR.Routes = append(appIR.Routes, routeIR)

			routeNodeID := "route:" + rt.Path
			graph.AddNode(routeNodeID, rt.Path, NodeRoute)
		}
	}

	// 2. Parse Go Backend AST (server/ directory)
	serverDir := filepath.Join(a.ProjectDir, "server")
	if _, err := os.Stat(serverDir); err == nil {
		fset := token.NewFileSet()

		err := filepath.WalkDir(serverDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return nil
			}

			packageName := node.Name.Name
			relFile, _ := filepath.Rel(a.ProjectDir, path)

			// Inspect AST declarations
			ast.Inspect(node, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}

				structName := ts.Name.Name

				switch {
				case strings.HasSuffix(structName, "Handler"):
					handler := &ir.HandlerIR{
						Name:          structName,
						Package:       packageName,
						File:          relFile,
						CallsServices: make([]string, 0),
					}

					// Inspect struct fields for dependencies
					for _, field := range st.Fields.List {
						fieldTypeStr := fmt.Sprintf("%v", field.Type)
						if strings.Contains(fieldTypeStr, "Service") {
							serviceName := extractTypeName(field.Type)
							handler.CallsServices = append(handler.CallsServices, serviceName)
						}
						if strings.Contains(fieldTypeStr, "sql.DB") || strings.Contains(fieldTypeStr, "database.DB") {
							handler.DirectDBAccess = true
						}
					}

					appIR.Handlers = append(appIR.Handlers, handler)
					hNodeID := fmt.Sprintf("handler:%s.%s", packageName, structName)
					graph.AddNode(hNodeID, structName, NodeHandler)

					// Connect corresponding Route -> Handler
					moduleName := strings.ToLower(strings.TrimSuffix(structName, "Handler"))
					routeNodeID := fmt.Sprintf("route:/%s", moduleName)
					if _, exists := graph.Nodes[routeNodeID]; exists {
						graph.AddEdge(routeNodeID, hNodeID, "dispatches")
					}

					for _, svc := range handler.CallsServices {
						sNodeID := fmt.Sprintf("service:%s.%s", packageName, svc)
						graph.AddEdge(hNodeID, sNodeID, "calls")
					}

					if handler.DirectDBAccess {
						graph.AddEdge(hNodeID, dbNodeID, "❌ VIOLATION: direct db")
					}

				case strings.HasSuffix(structName, "Service"):
					service := &ir.ServiceIR{
						Name:       structName,
						Package:    packageName,
						File:       relFile,
						CallsRepos: make([]string, 0),
					}

					for _, field := range st.Fields.List {
						fieldTypeStr := fmt.Sprintf("%v", field.Type)
						if strings.Contains(fieldTypeStr, "Repository") || strings.Contains(fieldTypeStr, "Repo") {
							repoName := extractTypeName(field.Type)
							service.CallsRepos = append(service.CallsRepos, repoName)
						}
					}

					appIR.Services = append(appIR.Services, service)
					sNodeID := fmt.Sprintf("service:%s.%s", packageName, structName)
					graph.AddNode(sNodeID, structName, NodeService)

					for _, repo := range service.CallsRepos {
						rNodeID := fmt.Sprintf("repository:%s.%s", packageName, repo)
						graph.AddEdge(sNodeID, rNodeID, "calls")
					}

				case strings.HasSuffix(structName, "Repository") || strings.HasSuffix(structName, "Repo"):
					repo := &ir.RepositoryIR{
						Name:        structName,
						Package:     packageName,
						File:        relFile,
						TargetTable: strings.ToLower(strings.TrimSuffix(structName, "Repository")),
					}

					appIR.Repositories = append(appIR.Repositories, repo)
					rNodeID := fmt.Sprintf("repository:%s.%s", packageName, structName)
					graph.AddNode(rNodeID, structName, NodeRepository)
					graph.AddEdge(rNodeID, dbNodeID, "queries")

				default:
					// Entity model
					entity := &ir.EntityIR{
						Name:   structName,
						Fields: make([]ir.FieldIR, 0),
					}
					for _, field := range st.Fields.List {
						for _, fname := range field.Names {
							entity.Fields = append(entity.Fields, ir.FieldIR{
								Name: fname.Name,
								Type: fmt.Sprintf("%v", field.Type),
							})
						}
					}
					appIR.Entities = append(appIR.Entities, entity)
				}

				return true
			})

			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	return appIR, graph, nil
}

func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return fmt.Sprintf("%v", expr)
	}
}
