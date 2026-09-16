package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pogonian/internal/architecture"
	"pogonian/pkg/config"
	"pogonian/pkg/router"
)

// OpenAPI 3.0.0 Spec Structures

type Spec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []ServerObject         `json:"servers,omitempty"`
	Tags       []Tag                  `json:"tags,omitempty"`
	Paths      map[string]*PathItem   `json:"paths"`
	Components *Components            `json:"components,omitempty"`
}

type Info struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Version        string `json:"version"`
	TermsOfService string `json:"termsOfService,omitempty"`
}

type ServerObject struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
}

type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "path", "query", "header", "cookie"
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Example     interface{}        `json:"example,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	In   string `json:"in,omitempty"`
}

var (
	formTagRegex    = regexp.MustCompile(`(?i)<form[^>]*\bmethod=["'](post|put|delete|patch|get)["'][^>]*>`)
	formActionRegex = regexp.MustCompile(`(?i)<form[^>]*\baction=["']([^"']+)["'][^>]*>`)
	inputNameRegex  = regexp.MustCompile(`(?i)<input[^>]*\bname=["']([a-zA-Z0-9_\-]+)["'][^>]*>`)
)

// GenerateSpec scans the project directory, routes, database tables, and API handlers to produce an OpenAPI 3.0 specification.
func GenerateSpec(projectDir string, port int) (*Spec, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	cfg, _ := config.Load(absDir)
	appName := "Pogonian App"
	if cfg != nil && cfg.Name != "" {
		appName = cfg.Name
	}

	if port <= 0 {
		if cfg != nil && cfg.Server.Port > 0 {
			port = cfg.Server.Port
		} else {
			port = 3000
		}
	}

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	spec := &Spec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       appName + " API & Routing Specification",
			Description: "Automatically generated full-stack API, route, and dynamic entity documentation for Pogonian.",
			Version:     "1.0.0",
		},
		Servers: []ServerObject{
			{
				URL:         serverURL,
				Description: "Local Development Server",
			},
		},
		Tags: []Tag{
			{Name: "Pages & Views", Description: "Filesystem HTML-first UI pages & layout routes"},
			{Name: "Dynamic Entities", Description: "Auto-generated CRUD REST endpoints & SQLite entities"},
			{Name: "Authentication", Description: "Session and user authentication actions"},
			{Name: "API Endpoints", Description: "Backend JSON API routes and custom Go handlers"},
			{Name: "DevTools & System", Description: "Pogonian framework diagnostics and client runtime"},
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas: make(map[string]*Schema),
			SecuritySchemes: map[string]SecurityScheme{
				"CSRFProtection": {
					Type: "apiKey",
					Name: "X-CSRF-Token",
					In:   "header",
				},
			},
		},
	}

	// Add Standard Schemas
	spec.Components.Schemas["ErrorResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"error": {Type: "string", Example: "Resource not found"},
			"code":  {Type: "integer", Example: 404},
		},
	}

	spec.Components.Schemas["AuthCredentials"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"username": {Type: "string", Example: "user@example.com"},
			"password": {Type: "string", Example: "secret123"},
		},
		Required: []string{"username", "password"},
	}

	// 1. Discover filesystem routes from app/
	appDir := filepath.Join(absDir, "app")
	r, err := router.New(appDir)
	if err == nil && r != nil {
		for _, rt := range r.Routes {
			openAPIPath := convertPogonianPathToOpenAPI(rt.Path)
			pathItem := spec.ensurePathItem(openAPIPath)

			pageFile := rt.PagePath
			pageContent, _ := os.ReadFile(pageFile)
			htmlStr := string(pageContent)

			var pathParams []Parameter
			for _, paramName := range rt.Params {
				pathParams = append(pathParams, Parameter{
					Name:        paramName,
					In:          "path",
					Required:    true,
					Description: fmt.Sprintf("Route parameter for %s", paramName),
					Schema:      &Schema{Type: "string"},
				})
			}

			// Add GET (Page Render)
			summary := fmt.Sprintf("Render %s page", rt.Path)
			pathItem.Get = &Operation{
				Tags:        []string{"Pages & Views"},
				Summary:     summary,
				Description: fmt.Sprintf("Returns HTML content rendered from `%s` with layout hierarchy.", filepath.ToSlash(filepath.Clean(rt.PagePath))),
				OperationID: "getRoute_" + sanitizeOpID(rt.Path),
				Parameters:  pathParams,
				Responses: map[string]Response{
					"200": {
						Description: "HTML Page Response",
						Content: map[string]MediaType{
							"text/html": {Schema: &Schema{Type: "string"}},
						},
					},
					"404": {Description: "Page not found"},
					"500": {Description: "Template rendering error"},
				},
			}

			// Check for Form POST/PUT/DELETE actions in HTML
			if formMatches := formTagRegex.FindAllStringSubmatch(htmlStr, -1); len(formMatches) > 0 {
				var inputNames []string
				if inputs := inputNameRegex.FindAllStringSubmatch(htmlStr, -1); len(inputs) > 0 {
					for _, inp := range inputs {
						name := inp[1]
						if name != "_csrf" && name != "_method" && !containsString(inputNames, name) {
							inputNames = append(inputNames, name)
						}
					}
				}

				formProps := make(map[string]*Schema)
				for _, name := range inputNames {
					formProps[name] = &Schema{Type: "string", Example: "value"}
				}

				formSchema := &Schema{
					Type:       "object",
					Properties: formProps,
				}

				for _, m := range formMatches {
					method := strings.ToUpper(m[1])
					switch method {
					case "POST":
						pathItem.Post = &Operation{
							Tags:        []string{"Pages & Views"},
							Summary:     fmt.Sprintf("Handle form submission for %s", rt.Path),
							Description: "Processes form submission, mutation, and redirect.",
							OperationID: "postRoute_" + sanitizeOpID(rt.Path),
							Parameters:  pathParams,
							RequestBody: &RequestBody{
								Required: true,
								Content: map[string]MediaType{
									"application/x-www-form-urlencoded": {Schema: formSchema},
									"multipart/form-data":               {Schema: formSchema},
								},
							},
							Responses: map[string]Response{
								"303": {Description: "Redirect to updated route state"},
								"400": {Description: "Validation or CSRF error"},
							},
						}
					}
				}
			}
		}
	}

	// 2. Discover Dynamic Database Entities / Server Domains (server/ or app/api/)
	analyzer, err := architecture.NewAnalyzer(absDir)
	if err == nil {
		if appIR, _, err := analyzer.Analyze(); err == nil {
			for _, e := range appIR.Entities {
				schemaName := e.Name
				entityProps := make(map[string]*Schema)
				for _, f := range e.Fields {
					fType := "string"
					switch strings.ToLower(f.Type) {
					case "int", "int64", "int32", "uint":
						fType = "integer"
					case "float64", "float32":
						fType = "number"
					case "bool":
						fType = "boolean"
					}
					entityProps[f.Name] = &Schema{
						Type:        fType,
						Description: fmt.Sprintf("Field %s (%s)", f.Name, f.Type),
					}
				}

				spec.Components.Schemas[schemaName] = &Schema{
					Type:       "object",
					Properties: entityProps,
				}

				entityPath := "/" + strings.ToLower(e.Name)
				pathItem := spec.ensurePathItem(entityPath)
				pathItem.Summary = fmt.Sprintf("CRUD operations for %s entity", schemaName)

				// GET list
				pathItem.Get = &Operation{
					Tags:        []string{"Dynamic Entities"},
					Summary:     fmt.Sprintf("List all %s items", schemaName),
					Description: fmt.Sprintf("Fetches all records from the SQLite `%s` table or renders the entity view.", strings.ToLower(e.Name)),
					OperationID: "list" + schemaName,
					Responses: map[string]Response{
						"200": {
							Description: fmt.Sprintf("List of %s items", schemaName),
							Content: map[string]MediaType{
								"application/json": {
									Schema: &Schema{
										Type:  "array",
										Items: &Schema{Type: "object", Properties: entityProps},
									},
								},
								"text/html": {Schema: &Schema{Type: "string"}},
							},
						},
					},
				}

				// POST create
				pathItem.Post = &Operation{
					Tags:        []string{"Dynamic Entities"},
					Summary:     fmt.Sprintf("Create a new %s", schemaName),
					Description: fmt.Sprintf("Inserts a new record into the `%s` database table.", strings.ToLower(e.Name)),
					OperationID: "create" + schemaName,
					RequestBody: &RequestBody{
						Required: true,
						Content: map[string]MediaType{
							"application/json":                  {Schema: &Schema{Type: "object", Properties: entityProps}},
							"application/x-www-form-urlencoded": {Schema: &Schema{Type: "object", Properties: entityProps}},
						},
					},
					Responses: map[string]Response{
						"201": {Description: fmt.Sprintf("%s created successfully", schemaName)},
						"303": {Description: "Redirect after form submission"},
						"400": {Description: "Invalid input data"},
					},
				}

				// Single item path /{entity}/{id}
				itemPath := entityPath + "/{id}"
				itemPathItem := spec.ensurePathItem(itemPath)
				itemPathParams := []Parameter{
					{
						Name:        "id",
						In:          "path",
						Required:    true,
						Description: fmt.Sprintf("Unique ID of the %s record", schemaName),
						Schema:      &Schema{Type: "string"},
					},
				}

				itemPathItem.Get = &Operation{
					Tags:        []string{"Dynamic Entities"},
					Summary:     fmt.Sprintf("Get %s by ID", schemaName),
					Description: fmt.Sprintf("Fetches a single %s item from the database.", schemaName),
					OperationID: "get" + schemaName + "ByID",
					Parameters:  itemPathParams,
					Responses: map[string]Response{
						"200": {
							Description: fmt.Sprintf("%s details", schemaName),
							Content: map[string]MediaType{
								"application/json": {Schema: &Schema{Type: "object", Properties: entityProps}},
								"text/html":        {Schema: &Schema{Type: "string"}},
							},
						},
						"404": {Description: fmt.Sprintf("%s not found", schemaName)},
					},
				}
			}
		}
	}

	// 3. Add Built-in Authentication Endpoints
	authLogin := spec.ensurePathItem("/login")
	authLogin.Post = &Operation{
		Tags:        []string{"Authentication"},
		Summary:     "User Login",
		Description: "Authenticates user credentials and establishes a persistent session.",
		OperationID: "authLogin",
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"email":    {Type: "string", Example: "admin@example.com"},
							"password": {Type: "string", Example: "password123"},
						},
						Required: []string{"email", "password"},
					},
				},
			},
		},
		Responses: map[string]Response{
			"303": {Description: "Redirect to dashboard upon successful login"},
			"401": {Description: "Invalid email or password"},
		},
	}

	authRegister := spec.ensurePathItem("/register")
	authRegister.Post = &Operation{
		Tags:        []string{"Authentication"},
		Summary:     "User Registration",
		Description: "Registers a new user with cryptographically hashed password.",
		OperationID: "authRegister",
		RequestBody: &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/x-www-form-urlencoded": {
					Schema: &Schema{
						Type: "object",
						Properties: map[string]*Schema{
							"username": {Type: "string", Example: "newuser"},
							"email":    {Type: "string", Example: "user@example.com"},
							"password": {Type: "string", Example: "securepass"},
						},
						Required: []string{"username", "email", "password"},
					},
				},
			},
		},
		Responses: map[string]Response{
			"303": {Description: "Redirect after account creation"},
			"400": {Description: "Validation error or email already exists"},
		},
	}

	authLogout := spec.ensurePathItem("/logout")
	authLogout.Post = &Operation{
		Tags:        []string{"Authentication"},
		Summary:     "User Logout",
		Description: "Clears the active session and logs out the current user.",
		OperationID: "authLogout",
		Responses: map[string]Response{
			"303": {Description: "Redirect to login or home page"},
		},
	}

	// 4. Add Standard API & Dev System Routes
	apiHealth := spec.ensurePathItem("/api/health")
	apiHealth.Get = &Operation{
		Tags:        []string{"API Endpoints"},
		Summary:     "API Health Check",
		Description: "Returns status and version health check for the Pogonian server.",
		OperationID: "getHealthCheck",
		Responses: map[string]Response{
			"200": {
				Description: "Server is healthy",
				Content: map[string]MediaType{
					"application/json": {
						Schema: &Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"status":    {Type: "string", Example: "ok"},
								"framework": {Type: "string", Example: "pogonian"},
							},
						},
					},
				},
			},
		},
	}

	runtimeJS := spec.ensurePathItem("/_pogn/runtime.js")
	runtimeJS.Get = &Operation{
		Tags:        []string{"DevTools & System"},
		Summary:     "Client SPA Runtime JS",
		Description: "Serves the high-performance Pogonian client soft navigation, form hydration, and API client runtime.",
		OperationID: "getClientRuntime",
		Responses: map[string]Response{
			"200": {Description: "JavaScript library bundle"},
		},
	}

	devtoolsPath := spec.ensurePathItem("/__pogn/devtools")
	devtoolsPath.Get = &Operation{
		Tags:        []string{"DevTools & System"},
		Summary:     "Architecture DevTools Dashboard",
		Description: "Live development dashboard with real-time request metrics, IR dependency graph, and spec editor.",
		OperationID: "getDevToolsDashboard",
		Responses: map[string]Response{
			"200": {Description: "DevTools dashboard HTML"},
		},
	}

	return spec, nil
}

func (s *Spec) ensurePathItem(path string) *PathItem {
	if item, ok := s.Paths[path]; ok {
		return item
	}
	item := &PathItem{}
	s.Paths[path] = item
	return item
}

func convertPogonianPathToOpenAPI(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	var newParts []string
	for _, p := range parts {
		if strings.HasPrefix(p, ":") {
			newParts = append(newParts, "{"+strings.TrimPrefix(p, ":")+"}")
		} else if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			newParts = append(newParts, "{"+strings.TrimSuffix(strings.TrimPrefix(p, "["), "]")+"}")
		} else {
			newParts = append(newParts, p)
		}
	}
	res := strings.Join(newParts, "/")
	if !strings.HasPrefix(res, "/") {
		res = "/" + res
	}
	return res
}

func sanitizeOpID(path string) string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return "Root"
	}
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, ":", "Param_")
	clean = strings.ReplaceAll(clean, "[", "")
	clean = strings.ReplaceAll(clean, "]", "")
	clean = strings.ReplaceAll(clean, "-", "_")
	return clean
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ServeSpecHandler returns an http.HandlerFunc that serves the generated OpenAPI 3.0 JSON specification.
func ServeSpecHandler(projectDir string, port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec, err := GenerateSpec(projectDir, port)
		if err != nil {
			http.Error(w, `{"error":"failed to generate openapi spec: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")
		_ = json.NewEncoder(w).Encode(spec)
	}
}
