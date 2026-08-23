package testrunner

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pogonian/internal/architecture"
	"pogonian/internal/config"
	"pogonian/internal/database"
	"pogonian/internal/router"
	"pogonian/internal/server"
)

// DisposableTestResult holds results of the end-to-end disposable project run.
type DisposableTestResult struct {
	ProjectName string
	Duration    time.Duration
	StepsPassed int
	TotalSteps  int
	Errors      []error
}

// RunDisposableProjectTest scaffolds a fresh temporary Pogonian project, tests routes, data loading,
// forms, and architecture contracts, then cleans up.
func RunDisposableProjectTest() (*DisposableTestResult, error) {
	start := time.Now()
	res := &DisposableTestResult{
		ProjectName: fmt.Sprintf("disposable-pogn-%d", time.Now().UnixNano()),
		TotalSteps:  6,
	}

	tempDir, err := os.MkdirTemp("", "pogn_disposable_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir) // Automatic cleanup

	projectDir := filepath.Join(tempDir, res.ProjectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, err
	}

	// Step 1: Scaffold fresh project
	cfg := config.DefaultConfig(res.ProjectName)
	if err := config.Save(projectDir, cfg); err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 1 failed: %w", err))
		return res, nil
	}
	if _, err := config.EnsureEnvFile(projectDir, res.ProjectName); err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 1 env failed: %w", err))
		return res, nil
	}

	// Create app layout and home page
	appDir := filepath.Join(projectDir, "app")
	_ = os.MkdirAll(appDir, 0755)

	layoutHTML := `<!DOCTYPE html><html><head><title>{{ .Title }}</title></head><body>{{ .Content }}</body></html>`
	_ = os.WriteFile(filepath.Join(appDir, "layout.html"), []byte(layoutHTML), 0644)

	pageHTML := `<main id="pogn-root"><h1>Welcome to Pogonian</h1><a href="/products">Products</a></main>`
	_ = os.WriteFile(filepath.Join(appDir, "page.html"), []byte(pageHTML), 0644)

	// Create products route
	prodDir := filepath.Join(appDir, "products")
	_ = os.MkdirAll(prodDir, 0755)
	prodHTML := `<main id="pogn-root"><h1>Products</h1><form action="/products" method="post"><input type="text" name="name" value="Laptop"><button type="submit">Add</button></form></main>`
	_ = os.WriteFile(filepath.Join(prodDir, "page.html"), []byte(prodHTML), 0644)

	res.StepsPassed++

	// Step 2: Test Database Migration Engine
	migDir := filepath.Join(projectDir, "migrations")
	upFile, _, err := database.GenerateMigration(migDir, "create_products")
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 2 migration generation failed: %w", err))
		return res, nil
	}
	upSQL := `CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);`
	_ = os.WriteFile(upFile, []byte(upSQL), 0644)

	dbPath := filepath.Join(projectDir, "data.db")
	dbDriver := &database.SQLiteDriver{Path: dbPath}
	dbConn, err := dbDriver.Open()
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 2 db open failed: %w", err))
		return res, nil
	}
	defer dbConn.Close()

	migEngine := database.NewMigrationEngine(dbConn, dbDriver)
	applied, err := migEngine.MigrateUp(migDir)
	if err != nil || len(applied) == 0 {
		res.Errors = append(res.Errors, fmt.Errorf("step 2 migration execute failed: %w", err))
		return res, nil
	}
	res.StepsPassed++

	// Step 3: Test Filesystem Router & Trie Matching
	r, err := router.New(appDir)
	if err != nil || len(r.Routes) < 2 {
		res.Errors = append(res.Errors, fmt.Errorf("step 3 router failed: %w", err))
		return res, nil
	}
	matchedRoute, _ := r.Match("/products")
	if matchedRoute == nil || matchedRoute.Path != "/products" {
		res.Errors = append(res.Errors, fmt.Errorf("step 3 route match failed for /products"))
		return res, nil
	}
	res.StepsPassed++

	// Step 4: Test HTTP Server Rendering & Soft Navigation Header
	srv, err := server.New(projectDir, true)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 4 server init failed: %w", err))
		return res, nil
	}

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL + "/")
	if err != nil || resp.StatusCode != http.StatusOK {
		res.Errors = append(res.Errors, fmt.Errorf("step 4 GET / failed: %v", err))
		return res, nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if !strings.Contains(string(bodyBytes), "Welcome to Pogonian") || !strings.Contains(string(bodyBytes), "/_pogn/runtime.js") {
		res.Errors = append(res.Errors, fmt.Errorf("step 4 body verification failed"))
		return res, nil
	}
	res.StepsPassed++

	// Step 5: Test Form Post with CSRF
	csrfCookie := ""
	for _, c := range resp.Cookies() {
		if c.Name == "_pogn_csrf" {
			csrfCookie = c.Value
		}
	}

	postForm := url.Values{"name": {"Super Gadget"}, "_csrf": {csrfCookie}}
	req, _ := http.NewRequest("POST", ts.URL+"/products", strings.NewReader(postForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrfCookie)
	req.AddCookie(&http.Cookie{Name: "_pogn_csrf", Value: csrfCookie})

	postResp, err := client.Do(req)
	if err != nil || (postResp.StatusCode != http.StatusOK && postResp.StatusCode != http.StatusSeeOther) {
		res.Errors = append(res.Errors, fmt.Errorf("step 5 POST /products failed: %v", err))
		return res, nil
	}
	res.StepsPassed++

	// Step 6: Test Architecture Static Analysis & Verification
	analyzer, err := architecture.NewAnalyzer(projectDir)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 6 analyzer failed: %w", err))
		return res, nil
	}

	appIR, graph, err := analyzer.Analyze()
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("step 6 analyze failed: %w", err))
		return res, nil
	}

	validator := architecture.NewValidator(false)
	violations := validator.Validate(appIR, graph)
	for _, v := range violations {
		if v.Severity == "ERROR" {
			res.Errors = append(res.Errors, fmt.Errorf("step 6 architecture violation: %s", v.Message))
			return res, nil
		}
	}
	res.StepsPassed++

	res.Duration = time.Since(start)
	return res, nil
}
