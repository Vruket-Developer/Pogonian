package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"pogonian/internal/compiler"
	"pogonian/internal/config"
	"pogonian/internal/database"
	"pogonian/internal/devtools"
	"pogonian/internal/middleware"
	"pogonian/internal/openapi"
	"pogonian/internal/router"
	"pogonian/internal/version"
	"pogonian/runtime"
)

// DataLoaderFunc loads dynamic data for a route before rendering.
type DataLoaderFunc func(r *http.Request, params map[string]string) (interface{}, error)

// ActionHandlerFunc handles form submissions (POST/PUT/DELETE) for a route.
type ActionHandlerFunc func(w http.ResponseWriter, r *http.Request, params map[string]string) error

// Server handles HTTP requests and backend architecture for a Pogonian project.
type Server struct {
	ProjectDir         string
	Config             *config.Config
	Router             *router.Router
	Renderer           *compiler.Renderer
	DB                 *database.DB
	DataLoaders        map[string]DataLoaderFunc
	Actions            map[string]ActionHandlerFunc
	DevMode            bool
	KillPortOnConflict bool
	StrictPort         bool
	httpServer         *http.Server
	liveClients        map[chan string]bool
	liveMu             sync.Mutex
}

// responseRecorder captures the HTTP status code for logging.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// New creates a new Pogonian HTTP server instance with DB, template engine, and middleware.
func New(projectDir string, devMode bool) (*Server, error) {
	cfg, err := config.Load(projectDir)
	if err != nil {
		cfg = config.DefaultConfig(filepath.Base(projectDir))
	}

	appDir := filepath.Join(projectDir, "app")
	r, err := router.New(appDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize router: %w", err)
	}

	rend := compiler.NewRendererWithMode(cfg.Name, projectDir, devMode)

	// In production, precompile all template chains for maximum performance and zero per-request parsing
	if !devMode {
		if err := rend.PrecompileAll(r.Routes); err != nil {
			return nil, fmt.Errorf("failed to precompile production templates: %w", err)
		}
	}

	var db *database.DB
	if cfg.Database.Driver == "sqlite" && cfg.Database.Path != "" {
		dbPath := cfg.Database.Path
		if !filepath.IsAbs(dbPath) && dbPath != ":memory:" {
			dbPath = filepath.Join(projectDir, dbPath)
		}
		dbConn, err := database.Open(dbPath)
		if err == nil {
			db = dbConn
		}
	}

	s := &Server{
		ProjectDir:  projectDir,
		Config:      cfg,
		Router:      r,
		Renderer:    rend,
		DB:          db,
		DataLoaders: make(map[string]DataLoaderFunc),
		Actions:     make(map[string]ActionHandlerFunc),
		DevMode:     devMode,
		liveClients: make(map[chan string]bool),
	}

	s.setupBuiltinModules()

	return s, nil
}

// RegisterLoader registers a data loader for a specific route path (e.g. "/products" or "/products/:id").
func (s *Server) RegisterLoader(path string, loader DataLoaderFunc) {
	s.DataLoaders[path] = loader
}

// RegisterAction registers a form POST/PUT/DELETE action handler for a specific route path.
func (s *Server) RegisterAction(path string, action ActionHandlerFunc) {
	s.Actions[path] = action
}

// Handler returns the fully wrapped HTTP handler with middleware.
func (s *Server) Handler() http.Handler {
	baseHandler := http.HandlerFunc(s.handleRequest)

	return middleware.Chain(
		baseHandler,
		middleware.Recovery(s.DevMode),
		middleware.RateLimit(500, time.Minute),
		middleware.CSRF(true), // Enabled fail-closed CSRF protection
	)
}

// resolveEffectiveMethod inspects headers and form body to support method spoofing (PUT, PATCH, DELETE).
func resolveEffectiveMethod(r *http.Request) string {
	if r.Method == http.MethodPost {
		// 1. Check Header
		if override := r.Header.Get("X-HTTP-Method-Override"); override != "" {
			upper := strings.ToUpper(strings.TrimSpace(override))
			if upper == "PUT" || upper == "PATCH" || upper == "DELETE" {
				return upper
			}
		}
		// 2. Check Form value
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()
		if formMethod := r.FormValue("_method"); formMethod != "" {
			upper := strings.ToUpper(strings.TrimSpace(formMethod))
			if upper == "PUT" || upper == "PATCH" || upper == "DELETE" {
				return upper
			}
		}
	}
	return r.Method
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	defer func() {
		duration := time.Since(start)
		s.logRequest(r.Method, r.URL.Path, rec.statusCode, duration)
		if s.DevMode {
			devtools.GlobalRecorder.Record(r.Method, r.URL.Path, rec.statusCode, duration)
		}
	}()

	// 1. DevTools, Live Reload & DevBar routes (Dev Mode only)
	if s.DevMode {
		if r.URL.Path == "/__pogn/devtools" {
			devtools.ServeDevTools(s.ProjectDir).ServeHTTP(rec, r)
			return
		}
		if r.URL.Path == "/__pogn/devbar.js" {
			devtools.ServeDevBar().ServeHTTP(rec, r)
			return
		}
		if r.URL.Path == "/__pogn/live-reload" {
			s.serveLiveReload(rec, r)
			return
		}
	}

	// 2. Interactive Swagger UI & OpenAPI Specification Endpoints
	if r.URL.Path == "/swagger" || r.URL.Path == "/swagger/" || r.URL.Path == "/api/docs" || r.URL.Path == "/__pogn/swagger" {
		openapi.ServeSwaggerUI(s.ProjectDir, "/swagger.json").ServeHTTP(rec, r)
		return
	}
	if r.URL.Path == "/swagger.json" || r.URL.Path == "/api/openapi.json" || r.URL.Path == "/__pogn/swagger.json" {
		openapi.ServeSpecHandler(s.ProjectDir, s.Config.Server.Port).ServeHTTP(rec, r)
		return
	}

	// 3. Serve Static Assets (Public, app/ & project static files with explicit path traversal protection)
	if s.handleStatic(rec, r) {
		return
	}

	// 4. Serve Builtin Client Runtime
	if r.URL.Path == "/_pogn/runtime.js" {
		rec.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		if s.DevMode {
			rec.Header().Set("Cache-Control", "no-cache")
		} else {
			rec.Header().Set("Cache-Control", "public, max-age=86400")
		}
		rec.Write(runtime.RuntimeJS)
		return
	}

	// 5. Direct API Endpoint Dispatcher (/api/...)
	if s.handleAPI(rec, r) {
		return
	}

	// 6. Match Filesystem Route (with dynamic dev-mode rescan for zero-restart route discovery)
	route, params := s.Router.Match(r.URL.Path)
	if route == nil && s.DevMode {
		_ = s.Router.Scan()
		route, params = s.Router.Match(r.URL.Path)
	}

	if route != nil {
		effectiveMethod := resolveEffectiveMethod(r)

		// Handle Action Submissions (Form POST/PUT/PATCH/DELETE)
		if effectiveMethod == http.MethodPost || effectiveMethod == http.MethodPut ||
			effectiveMethod == http.MethodPatch || effectiveMethod == http.MethodDelete {

			if action, exists := s.Actions[route.Path]; exists {
				if err := action(rec, r, params); err != nil {
					rec.WriteHeader(http.StatusInternalServerError)
					rec.Header().Set("Content-Type", "text/html; charset=utf-8")
					fmt.Fprintf(rec, "<h1>500 Action Handler Error</h1><p>%v</p>", err)
				}
				return
			}

			// Fallback dynamic action handler for SQLite dynamic entities
			if s.DB != nil {
				tableName := strings.Trim(route.Path, "/")
				_ = r.ParseMultipartForm(32 << 20)
				_ = r.ParseForm()

				formMap := make(map[string]string)
				for k, v := range r.PostForm {
					if len(v) > 0 && k != "_csrf" && k != "_method" {
						formMap[k] = v[0]
					}
				}

				if len(formMap) > 0 {
					_ = s.DB.GenericInsertForm(tableName, formMap)
				}
				http.Redirect(rec, r, route.Path, http.StatusSeeOther)
				return
			}
		}

		// Handle Data Loading
		var data interface{}
		if loader, exists := s.DataLoaders[route.Path]; exists {
			loadedData, err := loader(r, params)
			if err != nil {
				rec.WriteHeader(http.StatusInternalServerError)
				rec.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(rec, "<h1>Data Loading Error</h1><p>%v</p>", err)
				return
			}
			data = loadedData
		} else if s.DB != nil {
			// Fallback generic SQLite data loader for dynamic tables and single records
			cleanPath := strings.Trim(route.Path, "/")
			tableName := strings.Trim(strings.Split(cleanPath, "/")[0], "/")

			if idStr, ok := params["id"]; ok && idStr != "" {
				var item map[string]interface{}
				if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
					item, _ = s.DB.GenericSelectByID(tableName, id)
				}
				data = map[string]interface{}{
					"Product": item,
					"Item":    item,
					"Record":  item,
				}
			} else {
				items, _ := s.DB.GenericSelect(tableName)
				if items == nil {
					items = make([]map[string]interface{}, 0)
				}
				data = map[string]interface{}{
					"Products": items,
					"Items":    items,
					"Total":    len(items),
				}
			}
		} else {
			data = map[string]interface{}{
				"Items": make([]interface{}, 0),
				"Total": 0,
			}
		}

		query := make(map[string]string)
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}

		csrfToken, _ := middleware.GenerateToken()

		html, err := s.Renderer.Render(route, params, query, data, csrfToken)
		if err != nil {
			rec.WriteHeader(http.StatusInternalServerError)
			rec.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(rec, "<h1>500 Internal Server Error</h1><pre>failed to execute page template %s: %v</pre>", route.PagePath, err)
			return
		}

		rec.Header().Set("Content-Type", "text/html; charset=utf-8")
		rec.Header().Set("X-Powered-By", "Pogonian")
		rec.Write(html)
		return
	}

	// 7. Route Not Found (404)
	s.handleNotFound(rec, r)
}

// handleAPI handles backend JSON API requests, health checks, and SQLite entity CRUD endpoints.
func (s *Server) handleAPI(w *responseRecorder, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" {
		return false
	}

	// 1. Health check: /api/health
	if r.URL.Path == "/api/health" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"framework": "pogonian",
			"version":   version.Current,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return true
	}

	// 2. Discovered routes listing: /api/routes
	if r.URL.Path == "/api/routes" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		var routeList []map[string]interface{}
		for _, rt := range s.Router.Routes {
			routeList = append(routeList, map[string]interface{}{
				"path":   rt.Path,
				"page":   rt.PagePath,
				"params": rt.Params,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"routes": routeList,
			"total":  len(routeList),
		})
		return true
	}

	// 3. Dynamic REST CRUD for SQLite tables (/api/:entity or /api/:entity/:id)
	if s.DB != nil {
		relAPI := strings.TrimPrefix(r.URL.Path, "/api/")
		parts := strings.Split(relAPI, "/")
		tableName := parts[0]

		if tableName != "" {
			switch r.Method {
			case http.MethodGet:
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if len(parts) > 1 && parts[1] != "" {
					if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						item, err := s.DB.GenericSelectByID(tableName, id)
						if err != nil || item == nil {
							w.WriteHeader(http.StatusNotFound)
							_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "not found"})
							return true
						}
						_ = json.NewEncoder(w).Encode(item)
						return true
					}
				} else {
					items, err := s.DB.GenericSelect(tableName)
					if err == nil && items != nil {
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"data":  items,
							"total": len(items),
						})
						return true
					}
				}

			case http.MethodPost:
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				formMap := make(map[string]string)
				if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
					var jsonBody map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&jsonBody); err == nil {
						for k, v := range jsonBody {
							formMap[k] = fmt.Sprintf("%v", v)
						}
					}
				} else {
					_ = r.ParseMultipartForm(32 << 20)
					_ = r.ParseForm()
					for k, v := range r.PostForm {
						if len(v) > 0 && k != "_csrf" && k != "_method" {
							formMap[k] = v[0]
						}
					}
				}
				if len(formMap) > 0 {
					if err := s.DB.GenericInsertForm(tableName, formMap); err == nil {
						w.WriteHeader(http.StatusCreated)
						_ = json.NewEncoder(w).Encode(map[string]interface{}{
							"success": true,
							"message": fmt.Sprintf("Record created in %s", tableName),
						})
						return true
					}
				}

			case http.MethodDelete:
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				if len(parts) > 1 && parts[1] != "" {
					if id, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
						if err := s.DB.GenericDeleteByID(tableName, id); err == nil {
							_ = json.NewEncoder(w).Encode(map[string]interface{}{
								"success": true,
								"message": "Record deleted successfully",
							})
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// serveLiveReload provides Server-Sent Events (SSE) stream for browser auto-reload.
func (s *Server) serveLiveReload(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgChan := make(chan string, 10)
	s.liveMu.Lock()
	if s.liveClients == nil {
		s.liveClients = make(map[chan string]bool)
	}
	s.liveClients[msgChan] = true
	s.liveMu.Unlock()

	defer func() {
		s.liveMu.Lock()
		delete(s.liveClients, msgChan)
		s.liveMu.Unlock()
	}()

	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// BroadcastLiveReload broadcasts a reload signal to all connected browser clients.
func (s *Server) BroadcastLiveReload(event string) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	for ch := range s.liveClients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// handleStatic serves static files with strict path traversal prevention.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) bool {
	// Block null bytes and control characters
	if strings.Contains(r.URL.Path, "\x00") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return true
	}

	cleaned := filepath.Clean(r.URL.Path)
	if cleaned == "/" || cleaned == "." {
		return false
	}
	relPath := strings.TrimPrefix(cleaned, "/")

	absProject, err := filepath.Abs(s.ProjectDir)
	if err != nil {
		return false
	}

	// 1. Check inside app/ directory (e.g. app/globals.css, app/globals.js)
	appCandidate, err := filepath.Abs(filepath.Join(absProject, "app", relPath))
	if err == nil && isSubPath(filepath.Join(absProject, "app"), appCandidate) {
		if info, err := os.Stat(appCandidate); err == nil && !info.IsDir() {
			http.ServeFile(w, r, appCandidate)
			return true
		}
	}

	// 2. Direct files in project root (e.g. globals.css, globals.js, favicon.svg)
	rootCandidate, err := filepath.Abs(filepath.Join(absProject, relPath))
	if err == nil && isSubPath(absProject, rootCandidate) {
		if info, err := os.Stat(rootCandidate); err == nil && !info.IsDir() {
			http.ServeFile(w, r, rootCandidate)
			return true
		}
	}

	// 3. Files inside public/ or app/public/
	publicCandidates := []string{
		filepath.Join(absProject, "public"),
		filepath.Join(absProject, "app", "public"),
	}
	for _, pDir := range publicCandidates {
		pCand, err := filepath.Abs(filepath.Join(pDir, relPath))
		if err == nil && isSubPath(pDir, pCand) {
			if info, err := os.Stat(pCand); err == nil && !info.IsDir() {
				http.ServeFile(w, r, pCand)
				return true
			}
		}
	}

	return false
}

func isSubPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != ".."
}

func (s *Server) handleNotFound(w *responseRecorder, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if user has custom 404.html in app/ or root
	for _, p := range []string{filepath.Join(s.ProjectDir, "app", "404.html"), filepath.Join(s.ProjectDir, "404.html")} {
		if data, err := os.ReadFile(p); err == nil {
			w.Write(data)
			return
		}
	}

	// Default sleek 404
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>404 - Page Not Found | Pogonian</title>
    <link rel="stylesheet" href="/globals.css">
</head>
<body class="pogn-body" style="display:flex;align-items:center;justify-content:center;height:100vh;text-align:center;">
    <div style="max-width:500px;padding:2rem;">
        <h1 style="font-size:4rem;background:linear-gradient(135deg,#38bdf8,#818cf8);-webkit-background-clip:text;-webkit-text-fill-color:transparent;">404</h1>
        <h2 style="margin:1rem 0;">Page Not Found</h2>
        <p style="color:#9aa5b5;margin-bottom:2rem;">No route matched <code>%s</code> in your <code>app/</code> directory.</p>
        <a href="/" class="btn btn-primary">&larr; Return Home</a>
    </div>
</body>
</html>`, r.URL.Path)
}

func (s *Server) logRequest(method, path string, status int, duration time.Duration) {
	statusColor := "\x1b[1;32m" // green
	if status >= 400 && status < 500 {
		statusColor = "\x1b[1;33m" // yellow
	} else if status >= 500 {
		statusColor = "\x1b[1;31m" // red
	}

	fmt.Printf("  \x1b[1;34m[%s]\x1b[0m %-24s %s%d\x1b[0m \x1b[1;30m(%s)\x1b[0m\n",
		method, path, statusColor, status, duration.Round(time.Microsecond))
}

// Start launches the HTTP server listening on the configured host and port with automatic fallback.
func (s *Server) Start() error {
	host := s.Config.Server.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := s.Config.Server.Port
	if port <= 0 {
		port = 3000
	}

	// 1. If user passed --kill / -k, clear any zombie process occupying the port
	if s.KillPortOnConflict {
		if killed := killProcessOnPort(port); killed {
			fmt.Println()
			fmt.Printf("  \x1b[1;33m⚡ Cleared process occupying port %d\x1b[0m\n", port)
			time.Sleep(150 * time.Millisecond)
		}
	}

	// 2. Attempt to bind listener with automatic fallback to next available port (3000 -> 3001 -> 3002...)
	var listener net.Listener
	var err error
	originalPort := port

	for attempts := 0; attempts < 20; attempts++ {
		addr := fmt.Sprintf("%s:%d", host, port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}

		if s.StrictPort {
			return fmt.Errorf("port %d is in use and --strict-port is enabled: %w", port, err)
		}

		port++
	}

	if listener == nil {
		return fmt.Errorf("failed to bind to any open port starting from %d: %w", originalPort, err)
	}

	s.Config.Server.Port = port
	actualAddr := listener.Addr().String()

	s.httpServer = &http.Server{
		Addr:    actualAddr,
		Handler: s.Handler(),
	}

	modeName := "Dev Server"
	if !s.DevMode {
		modeName = "Production Server"
	}

	fmt.Println()
	fmt.Printf("  \x1b[1;36mPogonian %s\x1b[0m \x1b[1;30m%s\x1b[0m\n", modeName, version.Current)
	if port != originalPort {
		fmt.Printf("  \x1b[1;33m⚠️  Port %d was busy. Automatically switched to port %d\x1b[0m\n", originalPort, port)
		fmt.Printf("  \x1b[1;30m↳ Tip: Run with '-k' or '--kill' to clear port %d: pogn dev -k\x1b[0m\n", originalPort)
	}
	fmt.Printf("  ➜  \x1b[1mLocal:\x1b[0m     \x1b[1;32mhttp://%s\x1b[0m\n", actualAddr)
	fmt.Printf("  ➜  \x1b[1mSwagger:\x1b[0m   \x1b[1;35mhttp://%s/swagger\x1b[0m\n", actualAddr)
	fmt.Printf("  ➜  \x1b[1mApp:\x1b[0m       %s\n", s.Config.Name)
	fmt.Printf("  ➜  \x1b[1mRoutes:\x1b[0m    %d discovered\n", len(s.Router.Routes))
	for _, route := range s.Router.Routes {
		fmt.Printf("     \x1b[1;30m•\x1b[0m %-20s \x1b[1;30m(%s)\x1b[0m\n", route.Path, filepath.Base(filepath.Dir(route.PagePath)))
	}
	fmt.Println()
	fmt.Printf("  \x1b[1;30mReady for requests. Press Ctrl+C to stop.\x1b[0m\n\n")

	return s.httpServer.Serve(listener)
}

func killProcessOnPort(port int) bool {
	// 1. Try fuser -k on linux
	if err := exec.Command("fuser", "-k", fmt.Sprintf("%d/tcp", port)).Run(); err == nil {
		return true
	}
	// 2. Try lsof + kill
	cmd := fmt.Sprintf("lsof -ti tcp:%d | xargs -r kill -9", port)
	if err := exec.Command("sh", "-c", cmd).Run(); err == nil {
		return true
	}
	return false
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Close forcefully closes the server and database connections.
func (s *Server) Close() error {
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}
