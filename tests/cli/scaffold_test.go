package cli_test

import (
	"strings"
	"testing"

	"pogonian/tests/harness"
)

func TestCLI_AddAndRemoveTable(t *testing.T) {
	app := harness.NewTestRunner(t, "scaffold-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn add table customer
	out, err := app.RunCLI("add", "table", "customer", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn add table failed: %v\nOutput: %s", err, out)
	}

	if !strings.Contains(out, "Successfully added table 'customer'") {
		t.Errorf("Expected success message, got:\n%s", out)
	}

	app.AssertFileExists("server/customer/entity.go")
	app.AssertFileExists("server/customer/repository.go")
	app.AssertFileExists("server/customer/service.go")
	app.AssertFileExists("server/customer/handler.go")
	app.AssertFileExists("app/customer/page.html")

	// Verify architecture check passes
	checkOut, err := app.RunCLI("check", projectDir)
	if err != nil {
		t.Fatalf("pogn check failed after add table: %v\nOutput: %s", err, checkOut)
	}

	// 2. Test pogn remove table customer
	rmOut, err := app.RunCLI("remove", "table", "customer", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn remove table failed: %v\nOutput: %s", err, rmOut)
	}

	if !strings.Contains(rmOut, "Removed table 'customer'") {
		t.Errorf("Expected remove message, got:\n%s", rmOut)
	}
}

func TestCLI_AddAndRemoveRoute(t *testing.T) {
	app := harness.NewTestRunner(t, "scaffold-route-app")
	err := app.CreateApp()
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	projectDir := app.AppPath()

	// 1. Test pogn add route /faq
	out, err := app.RunCLI("add", "route", "/faq", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn add route failed: %v\nOutput: %s", err, out)
	}

	if (!strings.Contains(out, "Added route") && !strings.Contains(out, "Generated route")) || !strings.Contains(out, "faq") {
		t.Errorf("Expected route added message, got:\n%s", out)
	}
	app.AssertFileExists("app/faq/page.html")

	// 2. Test pogn remove route /faq
	rmOut, err := app.RunCLI("remove", "route", "/faq", "--dir="+projectDir)
	if err != nil {
		t.Fatalf("pogn remove route failed: %v\nOutput: %s", err, rmOut)
	}

	if !strings.Contains(rmOut, "Removed route") || !strings.Contains(rmOut, "faq") {
		t.Errorf("Expected route removed message, got:\n%s", rmOut)
	}
}
