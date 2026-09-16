package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pogonian/pkg/capability"
	"pogonian/pkg/compiler"
	"pogonian/pkg/ir"
)

// RunIR outputs or validates the Universal Pogonian IR for a project.
func RunIR(args []string) error {
	projectDir := "."
	format := "json"
	targetPlatform := ""

	for _, arg := range args {
		switch {
		case arg == "--yaml" || arg == "-y":
			format = "yaml"
		case arg == "--json" || arg == "-j":
			format = "json"
		case strings.HasPrefix(arg, "--target="):
			targetPlatform = strings.TrimPrefix(arg, "--target=")
		case !strings.HasPrefix(arg, "-"):
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	pipeline, err := compiler.NewPipeline(absDir)
	if err != nil {
		return fmt.Errorf("failed to initialize compiler pipeline: %w", err)
	}

	var targets []ir.TargetPlatform
	if targetPlatform != "" {
		targets = []ir.TargetPlatform{ir.TargetPlatform(targetPlatform)}
	} else {
		targets = []ir.TargetPlatform{ir.TargetWeb}
	}

	res, err := pipeline.Compile(targets)
	if err != nil {
		return fmt.Errorf("compiler pipeline failed: %w", err)
	}

	if format == "yaml" {
		yamlData, err := res.AppIR.ToYAML()
		if err != nil {
			return err
		}
		fmt.Println(string(yamlData))
	} else {
		jsonData, err := res.AppIR.ToJSON()
		if err != nil {
			return err
		}
		fmt.Println(string(jsonData))
	}

	return nil
}

// RunCapabilities prints the capability matrix and checks the current project.
func RunCapabilities(args []string) error {
	projectDir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			projectDir = arg
		}
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("⚡ \x1b[1;36mPogonian Cross-Platform Capability Matrix\x1b[0m")
	fmt.Println("   Auditing supported hardware & OS capabilities across all targets:")
	fmt.Println()

	matrix := capability.DefaultCapabilityMatrix()
	targets := ir.AllTargetPlatforms()

	fmt.Printf("%-24s", "CAPABILITY")
	for _, t := range targets {
		fmt.Printf("%-12s", strings.ToUpper(string(t)))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 24+12*len(targets)))

	caps := []ir.CapabilityType{
		ir.CapabilityNetwork,
		ir.CapabilityKeyboard,
		ir.CapabilityMouse,
		ir.CapabilityTouch,
		ir.CapabilityTVRemote,
		ir.CapabilityFileSystem,
		ir.CapabilityNotifications,
		ir.CapabilityLocation,
		ir.CapabilityCamera,
		ir.CapabilityMicrophone,
		ir.CapabilityBiometrics,
		ir.CapabilityBluetooth,
		ir.CapabilitySensors,
		ir.CapabilityClipboard,
		ir.CapabilityGameController,
		ir.CapabilityBackgroundExec,
	}

	for _, c := range caps {
		fmt.Printf("%-24s", c)
		for _, t := range targets {
			level := matrix[t][c]
			switch level {
			case capability.Supported:
				fmt.Printf("\x1b[1;32m%-12s\x1b[0m", "✓ YES")
			case capability.Conditional:
				fmt.Printf("\x1b[1;33m%-12s\x1b[0m", "⚡ PERM")
			case capability.Unsupported:
				fmt.Printf("\x1b[1;30m%-12s\x1b[0m", "✗ NO")
			}
		}
		fmt.Println()
	}
	fmt.Println()

	// Check current project capabilities if pogonian.yaml exists
	if _, err := os.Stat(filepath.Join(absDir, "pogonian.yaml")); err == nil {
		pipeline, err := compiler.NewPipeline(absDir)
		if err == nil {
			res, err := pipeline.Compile([]ir.TargetPlatform{ir.TargetWeb})
			if err == nil && len(res.AppIR.Capabilities) > 0 {
				fmt.Printf("📋 Project declared %d device capabilities. Validating compatibility...\n", len(res.AppIR.Capabilities))
				validator := capability.NewValidator()
				violations := validator.CheckCompatibility(res.AppIR, targets)
				fmt.Println(capability.FormatViolations(violations))
			}
		}
	}

	return nil
}
