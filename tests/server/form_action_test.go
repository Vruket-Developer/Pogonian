package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"pogonian/internal/server"
	"pogonian/tests/harness"
)

func TestServer_FormActionAndDynamicRoute(t *testing.T) {
	app := harness.NewTestRunner(t, "form-action-test")

	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	projectDir := app.AppPath()
	srv, err := server.New(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to initialize server: %v", err)
	}
	defer srv.Close()

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Initial GET /products (Should show empty inventory)
	res, err := http.Get(ts.URL + "/products")
	if err != nil {
		t.Fatalf("GET /products failed: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if !strings.Contains(string(body), "No products in database yet") {
		t.Errorf("Expected empty inventory message, got:\n%s", string(body))
	}

	// Extract CSRF cookie
	csrfCookie := ""
	for _, c := range res.Cookies() {
		if c.Name == "_pogn_csrf" {
			csrfCookie = c.Value
		}
	}

	// 2. Submit Form: POST /products to create a new product
	formData := url.Values{
		"name":        {"Mechanical Keyboard"},
		"price":       {"129.99"},
		"description": {"Custom mechanical keyboard with RGB switches"},
		"_csrf":       {csrfCookie},
	}

	postReq, err := http.NewRequest("POST", ts.URL+"/products", strings.NewReader(formData.Encode()))
	if err != nil {
		t.Fatalf("Failed to create POST request: %v", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRF-Token", csrfCookie)
	if csrfCookie != "" {
		postReq.AddCookie(&http.Cookie{Name: "_pogn_csrf", Value: csrfCookie})
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	postRes, err := client.Do(postReq)
	if err != nil {
		t.Fatalf("POST /products failed: %v", err)
	}
	postBody, _ := io.ReadAll(postRes.Body)
	postRes.Body.Close()

	// After following redirect, we should see the new product in the table!
	postBodyStr := string(postBody)
	if !strings.Contains(postBodyStr, "Mechanical Keyboard") {
		t.Errorf("Expected 'Mechanical Keyboard' in rendered list after POST, got:\n%s", postBodyStr)
	}
	if !strings.Contains(postBodyStr, "$129.99") {
		t.Errorf("Expected '$129.99' price in table, got:\n%s", postBodyStr)
	}

	// 3. Test Dynamic Route: GET /products/1
	detailRes, err := http.Get(ts.URL + "/products/1")
	if err != nil {
		t.Fatalf("GET /products/1 failed: %v", err)
	}
	detailBody, _ := io.ReadAll(detailRes.Body)
	detailRes.Body.Close()

	detailBodyStr := string(detailBody)
	if !strings.Contains(detailBodyStr, "Mechanical Keyboard") || !strings.Contains(detailBodyStr, "Custom mechanical keyboard") {
		t.Errorf("Expected product details on /products/1, got:\n%s", detailBodyStr)
	}
	if !strings.Contains(detailBodyStr, "id = 1") {
		t.Errorf("Expected dynamic route parameter 'id = 1' displayed, got:\n%s", detailBodyStr)
	}
}
