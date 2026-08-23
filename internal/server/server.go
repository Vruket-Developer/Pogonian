package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pogonian/internal/compiler"
	"pogonian/internal/config"
	"pogonian/internal/database"
	"pogonian/internal/devtools"
	"pogonian/internal/middleware"
	"pogonian/internal/router"
	"pogonian/runtime"
)

// DataLoaderFunc loads dynamic data for a route before rendering.
type DataLoaderFunc func(r *http.Request, params map[string]string) (interface{}, error)

// ActionHandlerFunc handles form submissions (POST/PUT/DELETE) for a route.
type ActionHandlerFunc func(w http.ResponseWriter, r *http.Request, params map[string]string) error

// Server handles HTTP requests and backend architecture for a Pogonian project.
type Server struct {
	ProjectDir  string
	Config      *config.Config
	Router      *router.Router
	Renderer    *compiler.Renderer
	DB          *database.DB
	DataLoaders map[string]DataLoaderFunc
	Actions     map[string]ActionHandlerFunc
	DevMode     bool
	httpServer  *http.Server
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

// New creates a new Pogonian HTTP server instance with DB and middleware.
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

	rend := compiler.NewRenderer(cfg.Name, projectDir)

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
	}

	s.setupBuiltinModules()

	return s, nil
}

// RegisterLoader registers a data loader for a specific route path (e.g. "/products" or "/products/:id").
func (s *Server) RegisterLoader(path string, loader DataLoaderFunc) {
	s.DataLoaders[path] = loader
}

// RegisterAction registers a form POST action handler for a specific route path.
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
		middleware.CSRF(false), // CSRF validation (soft-nav friendly)
	)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	defer func() {
		duration := time.Since(start)
		s.logRequest(r.Method, r.URL.Path, rec.statusCode, duration)
		devtools.GlobalRecorder.Record(r.Method, r.URL.Path, rec.statusCode, duration)
	}()

	path := r.URL.Path

	// 1. Serve embedded client runtime & devtools
	if path == "/_pogn/runtime.js" {
		runtime.Handler().ServeHTTP(rec, r)
		return
	}
	if path == "/__pogn/devtools" {
		devtools.ServeDevTools(s.ProjectDir).ServeHTTP(rec, r)
		return
	}
	if path == "/__pogn/devbar.js" {
		devtools.ServeDevBar().ServeHTTP(rec, r)
		return
	}

	// 2. Serve static project files (globals.css, globals.js, favicon.svg, public/*)
	if s.handleStatic(rec, r) {
		return
	}

	// 3. In dev mode, re-scan routes to pick up newly added/removed pages automatically
	if s.DevMode {
		_ = s.Router.Scan()
	}

	// 4. Match filesystem route
	route, params := s.Router.Match(path)
	if route != nil {
		// Handle POST Form Actions
		if r.Method == http.MethodPost {
			if action, exists := s.Actions[route.Path]; exists {
				if err := action(rec, r, params); err != nil {
					rec.WriteHeader(http.StatusBadRequest)
					rec.Header().Set("Content-Type", "text/html; charset=utf-8")
					fmt.Fprintf(rec, "<h1>Action Error</h1><p>%v</p>", err)
					return
				}
				if rec.statusCode != http.StatusOK {
					return
				}
			} else if s.DB != nil {
				// Fallback generic SQLite action handler for AI-generated modules
				tableName := strings.Trim(route.Path, "/")
				_ = r.ParseMultipartForm(32 << 20)
				_ = r.ParseForm()

				title := r.FormValue("title")
				if title == "" {
					title = r.FormValue("name")
				}
				content := r.FormValue("content")
				if content == "" {
					content = r.FormValue("description")
				}

				if title != "" {
					// Ensure table exists
					_, _ = s.DB.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						title TEXT NOT NULL,
						content TEXT,
						created_at DATETIME DEFAULT CURRENT_TIMESTAMP
					)`, tableName))

					_, _ = s.DB.Exec(
						fmt.Sprintf("INSERT INTO %s (title, content, created_at) VALUES (?, ?, ?)", tableName),
						title, content, time.Now().UTC().Format(time.RFC3339),
					)
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
			// Fallback generic SQLite data loader for AI-generated tables
			tableName := strings.Trim(route.Path, "/")
			type GenericItem struct {
				ID      int64
				Title   string
				Content string
			}
			var items []*GenericItem
			rows, err := s.DB.Query(fmt.Sprintf("SELECT id, title, content FROM %s ORDER BY id DESC", tableName))
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					item := &GenericItem{}
					if err := rows.Scan(&item.ID, &item.Title, &item.Content); err == nil {
						items = append(items, item)
					}
				}
			}
			if items == nil {
				items = make([]*GenericItem, 0)
			}
			data = map[string]interface{}{
				"Items": items,
				"Total": len(items),
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

		csrfToken := middleware.GenerateToken()

		html, err := s.Renderer.Render(route, params, query, data, csrfToken)
		if err != nil {
			rec.WriteHeader(http.StatusInternalServerError)
			rec.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(rec, "<h1>500 Internal Server Error</h1><pre>%s</pre>", err.Error())
			return
		}

		rec.Header().Set("Content-Type", "text/html; charset=utf-8")
		rec.Header().Set("X-Powered-By", "Pogonian")
		rec.Write(html)
		return
	}

	// 5. Route Not Found (404)
	s.handleNotFound(rec, r)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return false
	}

	// Direct files in project root (e.g. globals.css, globals.js, favicon.svg)
	rootCandidate := filepath.Join(s.ProjectDir, path)
	if info, err := os.Stat(rootCandidate); err == nil && !info.IsDir() {
		http.ServeFile(w, r, rootCandidate)
		return true
	}

	// Files inside public/
	publicCandidate := filepath.Join(s.ProjectDir, "public", path)
	if info, err := os.Stat(publicCandidate); err == nil && !info.IsDir() {
		http.ServeFile(w, r, publicCandidate)
		return true
	}

	return false
}

func (s *Server) handleNotFound(w *responseRecorder, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if user has custom 404.html
	custom404 := filepath.Join(s.ProjectDir, "app", "404.html")
	if data, err := os.ReadFile(custom404); err == nil {
		w.Write(data)
		return
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

// Start launches the HTTP server listening on the configured host and port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Config.Server.Host, s.Config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	fmt.Println()
	fmt.Printf("  \x1b[1;36mPogonian Dev Server\x1b[0m \x1b[1;30mv0.1.0\x1b[0m\n")
	fmt.Printf("  ➜  \x1b[1mLocal:\x1b[0m   \x1b[1;32mhttp://%s\x1b[0m\n", addr)
	fmt.Printf("  ➜  \x1b[1mApp:\x1b[0m     %s\n", s.Config.Name)
	fmt.Printf("  ➜  \x1b[1mRoutes:\x1b[0m  %d discovered\n", len(s.Router.Routes))
	for _, route := range s.Router.Routes {
		fmt.Printf("     \x1b[1;30m•\x1b[0m %-20s \x1b[1;30m(%s)\x1b[0m\n", route.Path, filepath.Base(filepath.Dir(route.PagePath)))
	}
	fmt.Println()
	fmt.Printf("  \x1b[1;30mReady for requests. Press Ctrl+C to stop.\x1b[0m\n\n")

	return s.httpServer.ListenAndServe()
}

// Close closes the server and database connections.
func (s *Server) Close() error {
	if s.DB != nil {
		_ = s.DB.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}
