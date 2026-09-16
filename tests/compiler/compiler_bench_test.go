package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pogonian/pkg/capability"
	"pogonian/pkg/compiler"
	"pogonian/pkg/dataruntime"
	"pogonian/pkg/ir"
	"pogonian/pkg/state"
)

func BenchmarkCompiler_MultiTargetLowering(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "pogn-bench-comp-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	_ = os.MkdirAll(filepath.Join(tempDir, "app"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "app", "page.html"), []byte("<h1>Bench</h1>"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "pogonian.yaml"), []byte("name: bench-app\nversion: 1.0.0\n"), 0644)

	pipeline, err := compiler.NewPipeline(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	targets := []ir.TargetPlatform{
		ir.TargetWeb,
		ir.TargetAndroid,
		ir.TargetIOS,
		ir.TargetLinux,
		ir.TargetAndroidTV,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Compile(targets)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkState_EnvelopeSerialization(b *testing.B) {
	engine := state.NewEngine("bench-secret-key-1234567890123456")
	app := &ir.Application{Name: "bench", Version: "1.0.0"}
	data := map[string]any{
		"user_id":   1042,
		"theme":     "dark",
		"volume":    75,
		"is_active": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snap, err := engine.CreateSnapshot(app, ir.TargetWeb, "/dash", "main", data)
		if err != nil {
			b.Fatal(err)
		}
		env, err := engine.PackEnvelope(snap, time.Minute)
		if err != nil {
			b.Fatal(err)
		}
		_, err = engine.UnpackEnvelope(env)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCapability_MatrixCheck(b *testing.B) {
	validator := capability.NewValidator()
	app := &ir.Application{
		Name:    "bench",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{Type: ir.CapabilityCamera, Required: true},
			{Type: ir.CapabilityNotifications, Required: false},
			{Type: ir.CapabilityTVRemote, Required: false},
		},
	}
	targets := ir.AllTargetPlatforms()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.CheckCompatibility(app, targets)
	}
}

func BenchmarkDataRuntime_InsertAndFind(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "pogn-bench-db-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	dr, err := dataruntime.Default(filepath.Join(tempDir, "bench.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer dr.Close()

	ctx := context.Background()
	_ = dr.EnsureEntity(ctx, "items", map[string]string{
		"title": "TEXT NOT NULL",
		"count": "INTEGER NOT NULL",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id, err := dr.Insert(ctx, "items", dataruntime.Record{
			"title": "Item",
			"count": i,
		})
		if err != nil {
			b.Fatal(err)
		}
		_, err = dr.FindByID(ctx, "items", id)
		if err != nil {
			b.Fatal(err)
		}
	}
}
