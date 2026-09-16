package capability_test

import (
	"testing"

	"pogonian/pkg/capability"
	"pogonian/pkg/ir"
)

func TestCapabilityMatrix_Defaults(t *testing.T) {
	matrix := capability.DefaultCapabilityMatrix()

	// Web should not support TVRemote or BackgroundExec natively
	if matrix[ir.TargetWeb][ir.CapabilityTVRemote] != capability.Unsupported {
		t.Errorf("Expected Web TVRemote to be Unsupported")
	}

	// Android TV should support TVRemote
	if matrix[ir.TargetAndroidTV][ir.CapabilityTVRemote] != capability.Supported {
		t.Errorf("Expected Android TV TVRemote to be Supported")
	}

	// Android TV should NOT support Touch
	if matrix[ir.TargetAndroidTV][ir.CapabilityTouch] != capability.Unsupported {
		t.Errorf("Expected Android TV Touch to be Unsupported")
	}

	// Linux Desktop should support Keyboard & Mouse
	if matrix[ir.TargetLinux][ir.CapabilityKeyboard] != capability.Supported ||
		matrix[ir.TargetLinux][ir.CapabilityMouse] != capability.Supported {
		t.Errorf("Expected Linux Desktop to support Keyboard and Mouse")
	}
}

func TestCapabilityValidator_UnsupportedRequiredFlagsError(t *testing.T) {
	validator := capability.NewValidator()

	app := &ir.Application{
		Name:    "tv-media-app",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{
				Type:     ir.CapabilityTVRemote,
				Required: true,
			},
		},
	}

	// Auditing for Web target should produce an ERROR because TVRemote is unsupported on Web
	violations := validator.CheckCompatibility(app, []ir.TargetPlatform{ir.TargetWeb})
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}
	if violations[0].Severity != "ERROR" {
		t.Errorf("Expected ERROR severity for required unsupported capability, got %s", violations[0].Severity)
	}

	// Auditing for Android TV should produce NO violations
	tvViolations := validator.CheckCompatibility(app, []ir.TargetPlatform{ir.TargetAndroidTV})
	if len(tvViolations) != 0 {
		t.Errorf("Expected 0 violations on Android TV, got %d", len(tvViolations))
	}
}

func TestCapabilityValidator_UnsupportedOptionalFlagsWarning(t *testing.T) {
	validator := capability.NewValidator()

	app := &ir.Application{
		Name:    "store-app",
		Version: "1.0.0",
		Capabilities: []*ir.CapabilityIR{
			{
				Type:           ir.CapabilityBiometrics,
				Required:       false,
				FallbackAction: "request_pin_password",
			},
		},
	}

	// On Android TV, biometrics is unsupported, but it's optional with fallback
	violations := validator.CheckCompatibility(app, []ir.TargetPlatform{ir.TargetAndroidTV})
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}
	if violations[0].Severity != "WARNING" {
		t.Errorf("Expected WARNING severity for optional capability with fallback, got %s", violations[0].Severity)
	}
}
