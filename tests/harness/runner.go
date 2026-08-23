package harness

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestApp represents a disposable Pogonian application created for testing.
type TestApp struct {
	Name       string
	Dir        string
	BinaryPath string
	ServerCmd  *exec.Cmd
	Port       int
	t          *testing.T
	mu         sync.Mutex
}

// NewTestRunner creates a disposable workspace inside a temporary directory or .test-work.
func NewTestRunner(t *testing.T, appName string) *TestApp {
	t.Helper()

	// Ensure .test-work or OS temp dir is used
	baseDir := filepath.Join(os.TempDir(), "pogonian-tests")
	_ = os.MkdirAll(baseDir, 0755)

	appDir, err := os.MkdirTemp(baseDir, fmt.Sprintf("pogn-%s-*", appName))
	if err != nil {
		t.Fatalf("Failed to create temporary directory for test app: %v", err)
	}

	app := &TestApp{
		Name: appName,
		Dir:  appDir,
		Port: 3999, // default test port
		t:    t,
	}

	// Register automatic cleanup upon test completion
	t.Cleanup(func() {
		app.Cleanup()
	})

	return app
}

// RunCLI executes a command using the locally built pogn CLI or go run.
func (app *TestApp) RunCLI(args ...string) (string, error) {
	app.t.Helper()

	// Find Pogonian root by walking up
	root, err := findRepoRoot()
	if err != nil {
		return "", fmt.Errorf("could not find pogonian repo root: %w", err)
	}

	cmdArgs := append([]string{"run", "./cmd/pogn"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String() + stderr.String()

	return output, err
}

// CreateApp scaffolds a new project in the test app directory.
func (app *TestApp) CreateApp(flags ...string) error {
	app.t.Helper()

	args := []string{"create", app.Name, "--template=default"}
	// Append additional flags
	args = append(args, flags...)

	// We pass the target directory inside app.Dir
	targetPath := filepath.Join(app.Dir, app.Name)
	args = append(args, targetPath)

	out, err := app.RunCLI(args...)
	if err != nil {
		return fmt.Errorf("pogn create failed: %v\nOutput:\n%s", err, out)
	}

	return nil
}

// AppPath returns the path to a file or directory within the generated project.
func (app *TestApp) AppPath(subPaths ...string) string {
	elems := append([]string{app.Dir, app.Name}, subPaths...)
	return filepath.Join(elems...)
}

// AssertFileExists checks that a file exists within the generated project.
func (app *TestApp) AssertFileExists(subPath string) {
	app.t.Helper()
	fullPath := app.AppPath(subPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		app.t.Fatalf("Expected file %s to exist at %s, but got error: %v", subPath, fullPath, err)
	}
	if info.IsDir() {
		app.t.Fatalf("Expected %s to be a file, but it is a directory", subPath)
	}
}

// AssertDirExists checks that a directory exists within the generated project.
func (app *TestApp) AssertDirExists(subPath string) {
	app.t.Helper()
	fullPath := app.AppPath(subPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		app.t.Fatalf("Expected directory %s to exist at %s, but got error: %v", subPath, fullPath, err)
	}
	if !info.IsDir() {
		app.t.Fatalf("Expected %s to be a directory, but it is a file", subPath)
	}
}

// AssertFileContains checks that a file contains an expected string.
func (app *TestApp) AssertFileContains(subPath, expectedContent string) {
	app.t.Helper()
	fullPath := app.AppPath(subPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		app.t.Fatalf("Failed to read file %s: %v", fullPath, err)
	}

	if !strings.Contains(string(data), expectedContent) {
		app.t.Fatalf("File %s does not contain expected substring:\nExpected:\n%s\nActual content:\n%s", subPath, expectedContent, string(data))
	}
}

// Request performs an HTTP GET request against the running test server.
func (app *TestApp) Request(path string) (*http.Response, string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", app.Port, path)
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res, "", err
	}

	return res, string(body), nil
}

// Cleanup terminates any running background server and removes the disposable project directory.
func (app *TestApp) Cleanup() {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.ServerCmd != nil && app.ServerCmd.Process != nil {
		_ = app.ServerCmd.Process.Kill()
		_ = app.ServerCmd.Wait()
		app.ServerCmd = nil
	}

	if app.Dir != "" {
		_ = os.RemoveAll(app.Dir)
	}
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return cwd, nil
}
