package cli_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pogonian/tests/harness"
)

func TestCLI_BuildStandalone(t *testing.T) {
	app := harness.NewTestRunner(t, "build-standalone-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()
	out, err := app.RunCLI("build", "--standalone", projectDir)
	if err != nil {
		t.Fatalf("pogn build --standalone failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Standalone production build complete") {
		t.Errorf("Expected build success summary, got:\n%s", out)
	}

	distBin := filepath.Join(projectDir, "dist", "build-standalone-app")
	if _, err := os.Stat(distBin); err != nil {
		t.Fatalf("Expected compiled binary at %s, but file not found: %v", distBin, err)
	}

	// Verify that the binary is a real, executable production server
	testPort := 3991
	cmd := exec.Command(distBin)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", testPort), "APP_ENV=production")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start compiled standalone binary: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Wait for server to become ready
	var resp *http.Response
	var getErr error
	targetURL := fmt.Sprintf("http://127.0.0.1:%d/", testPort)

	for attempt := 0; attempt < 20; attempt++ {
		time.Sleep(100 * time.Millisecond)
		client := &http.Client{Timeout: 1 * time.Second}
		resp, getErr = client.Get(targetURL)
		if getErr == nil && resp != nil && resp.StatusCode == http.StatusOK {
			break
		}
	}

	if getErr != nil || resp == nil {
		t.Fatalf("Standalone production binary failed to serve requests at %s: %v", targetURL, getErr)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	body := string(bodyBytes)

	if !strings.Contains(body, "Pogonian") && !strings.Contains(body, "html") {
		t.Errorf("Unexpected response body from standalone binary:\n%s", body)
	}

	// Verify client runtime asset is served in production
	runtimeURL := fmt.Sprintf("http://127.0.0.1:%d/_pogn/runtime.js", testPort)
	rtResp, rtErr := http.Get(runtimeURL)
	if rtErr != nil || rtResp.StatusCode != http.StatusOK {
		t.Errorf("Failed to retrieve client runtime from standalone binary: %v (Status: %v)", rtErr, rtResp)
	}
	if rtResp != nil {
		_ = rtResp.Body.Close()
	}
}
