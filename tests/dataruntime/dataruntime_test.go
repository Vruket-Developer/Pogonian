package dataruntime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pogonian/pkg/dataruntime"
)

func TestDataRuntime_CRUD(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pogn-dataruntime-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	dr, err := dataruntime.Default(dbPath)
	if err != nil {
		t.Fatalf("Failed to create default data runtime: %v", err)
	}
	defer dr.Close()

	ctx := context.Background()

	// 1. Ensure Entity Schema
	schema := map[string]string{
		"name":  "TEXT NOT NULL",
		"price": "REAL NOT NULL",
		"stock": "INTEGER NOT NULL",
	}
	if err := dr.EnsureEntity(ctx, "products", schema); err != nil {
		t.Fatalf("EnsureEntity failed: %v", err)
	}

	// 2. Insert records
	id1, err := dr.Insert(ctx, "products", dataruntime.Record{
		"name":  "Ergonomic Keyboard",
		"price": 129.99,
		"stock": 45,
	})
	if err != nil {
		t.Fatalf("Insert 1 failed: %v", err)
	}

	id2, err := dr.Insert(ctx, "products", dataruntime.Record{
		"name":  "Wireless Mouse",
		"price": 59.50,
		"stock": 100,
	})
	if err != nil {
		t.Fatalf("Insert 2 failed: %v", err)
	}

	// 3. Find By ID
	rec, err := dr.FindByID(ctx, "products", id1)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if rec["name"] != "Ergonomic Keyboard" {
		t.Errorf("Expected 'Ergonomic Keyboard', got %v", rec["name"])
	}

	// 4. Query with Filter
	filtered, err := dr.Find(ctx, "products", dataruntime.Filter{
		Field:    "price",
		Operator: dataruntime.OpLt,
		Value:    100.0,
	})
	if err != nil {
		t.Fatalf("Find with filter failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("Expected 1 filtered record, got %d", len(filtered))
	}
	if filtered[0]["name"] != "Wireless Mouse" {
		t.Errorf("Expected 'Wireless Mouse', got %v", filtered[0]["name"])
	}

	// 5. Update record
	err = dr.Update(ctx, "products", id2, dataruntime.Record{
		"price": 49.99,
		"stock": 95,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, err := dr.FindByID(ctx, "products", id2)
	if err != nil {
		t.Fatalf("FindByID updated failed: %v", err)
	}
	if updated["price"] != 49.99 {
		t.Errorf("Expected price 49.99, got %v", updated["price"])
	}

	// 6. Count
	count, err := dr.Count(ctx, "products")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 records, got %d", count)
	}

	// 7. Delete
	if err := dr.Delete(ctx, "products", id1); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	afterDeleteCount, err := dr.Count(ctx, "products")
	if err != nil {
		t.Fatalf("Count after delete failed: %v", err)
	}
	if afterDeleteCount != 1 {
		t.Errorf("Expected 1 record after delete, got %d", afterDeleteCount)
	}
}
