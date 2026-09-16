package capability

import (
	"fmt"
	"strings"

	"pogonian/pkg/ir"
)

// SupportLevel defines whether a capability is supported on a target platform.
type SupportLevel string

const (
	Supported   SupportLevel = "SUPPORTED"
	Conditional SupportLevel = "CONDITIONAL" // Requires runtime permissions or hardware check
	Unsupported SupportLevel = "UNSUPPORTED"
)

// CapabilityViolation describes an unsupported or conditionally supported capability constraint.
type CapabilityViolation struct {
	Platform       ir.TargetPlatform `json:"platform"`
	Capability     ir.CapabilityType `json:"capability"`
	Level          SupportLevel      `json:"level"`
	Severity       string            `json:"severity"` // "ERROR" or "WARNING"
	Message        string            `json:"message"`
	Recommendation string            `json:"recommendation"`
}

// Matrix defines hardware and OS capabilities across all Pogonian target platforms.
type Matrix map[ir.TargetPlatform]map[ir.CapabilityType]SupportLevel

// DefaultCapabilityMatrix returns the canonical Pogonian capability availability map.
func DefaultCapabilityMatrix() Matrix {
	m := make(Matrix)

	// Target: Web
	m[ir.TargetWeb] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityTouch:          Supported,
		ir.CapabilityClipboard:      Conditional,
		ir.CapabilityNotifications:  Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilityFileSystem:     Conditional,
		ir.CapabilitySensors:        Conditional,
		ir.CapabilityBiometrics:     Conditional, // WebAuthn
		ir.CapabilityBluetooth:      Conditional, // Web Bluetooth
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Unsupported,
	}

	// Target: Android
	m[ir.TargetAndroid] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityTouch:          Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityFileSystem:     Supported,
		ir.CapabilityClipboard:      Supported,
		ir.CapabilityNotifications:  Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilitySensors:        Supported,
		ir.CapabilityBiometrics:     Supported,
		ir.CapabilityBluetooth:      Conditional,
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Supported, // WorkManager
	}

	// Target: iOS
	m[ir.TargetIOS] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityTouch:          Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityFileSystem:     Supported, // Sandboxed app container
		ir.CapabilityClipboard:      Supported,
		ir.CapabilityNotifications:  Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilitySensors:        Supported,
		ir.CapabilityBiometrics:     Supported, // FaceID / TouchID
		ir.CapabilityBluetooth:      Conditional,
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Supported, // BGTaskScheduler
	}

	// Target: Linux Desktop
	m[ir.TargetLinux] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityFileSystem:     Supported,
		ir.CapabilityClipboard:      Supported,
		ir.CapabilityNotifications:  Supported, // Libnotify / Freedesktop
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilitySensors:        Conditional,
		ir.CapabilityBiometrics:     Conditional, // PAM / fprintd
		ir.CapabilityBluetooth:      Supported,   // BlueZ
		ir.CapabilityTouch:          Conditional,
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Supported,
	}

	// Target: Windows Desktop
	m[ir.TargetWindows] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityFileSystem:     Supported,
		ir.CapabilityClipboard:      Supported,
		ir.CapabilityNotifications:  Supported, // Toast notifications
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilitySensors:        Conditional,
		ir.CapabilityBiometrics:     Supported, // Windows Hello
		ir.CapabilityBluetooth:      Supported,
		ir.CapabilityTouch:          Conditional,
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Supported,
	}

	// Target: macOS Desktop
	m[ir.TargetMacOS] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMouse:          Supported,
		ir.CapabilityFileSystem:     Supported,
		ir.CapabilityClipboard:      Supported,
		ir.CapabilityNotifications:  Supported,
		ir.CapabilityCamera:         Conditional,
		ir.CapabilityMicrophone:     Conditional,
		ir.CapabilityLocation:       Conditional,
		ir.CapabilitySensors:        Conditional,
		ir.CapabilityBiometrics:     Supported, // TouchID
		ir.CapabilityBluetooth:      Supported,
		ir.CapabilityTouch:          Unsupported,
		ir.CapabilityGameController: Supported,
		ir.CapabilityTVRemote:       Unsupported,
		ir.CapabilityBackgroundExec: Supported,
	}

	// Target: Android TV / Google TV
	m[ir.TargetAndroidTV] = map[ir.CapabilityType]SupportLevel{
		ir.CapabilityNetwork:        Supported,
		ir.CapabilityTVRemote:       Supported, // D-Pad / Leanback Remote
		ir.CapabilityGameController: Supported,
		ir.CapabilityNotifications:  Supported,
		ir.CapabilityFileSystem:     Supported,
		ir.CapabilityKeyboard:       Supported,
		ir.CapabilityMicrophone:     Conditional, // Voice Remote input
		ir.CapabilityBluetooth:      Supported,
		ir.CapabilityTouch:          Unsupported, // 10-foot UI: no touch screen
		ir.CapabilityMouse:          Unsupported, // No desktop mouse cursor
		ir.CapabilityCamera:         Unsupported,
		ir.CapabilityBiometrics:     Unsupported,
		ir.CapabilityLocation:       Unsupported,
		ir.CapabilitySensors:        Unsupported,
		ir.CapabilityClipboard:      Unsupported,
		ir.CapabilityBackgroundExec: Supported,
	}

	return m
}

// Validator evaluates an application IR against target platform capability constraints.
type Validator struct {
	matrix Matrix
}

// NewValidator creates a capability validator using the canonical matrix.
func NewValidator() *Validator {
	return &Validator{
		matrix: DefaultCapabilityMatrix(),
	}
}

// CheckCompatibility audits all declared application capabilities against requested target platforms.
func (v *Validator) CheckCompatibility(app *ir.Application, targets []ir.TargetPlatform) []CapabilityViolation {
	var violations []CapabilityViolation

	if len(targets) == 0 {
		if len(app.SupportedTargets) > 0 {
			targets = app.SupportedTargets
		} else {
			targets = []ir.TargetPlatform{ir.TargetWeb}
		}
	}

	for _, target := range targets {
		platformMap, ok := v.matrix[target]
		if !ok {
			violations = append(violations, CapabilityViolation{
				Platform:       target,
				Severity:       "ERROR",
				Message:        fmt.Sprintf("Unknown or unsupported target platform '%s'", target),
				Recommendation: "Ensure target is one of: web, android, ios, linux, windows, macos, android_tv",
			})
			continue
		}

		for _, capReq := range app.Capabilities {
			level, recognized := platformMap[capReq.Type]
			if !recognized {
				level = Unsupported
			}

			switch level {
			case Unsupported:
				sev := "ERROR"
				if !capReq.Required || capReq.FallbackAction != "" {
					sev = "WARNING"
				}
				violations = append(violations, CapabilityViolation{
					Platform:   target,
					Capability: capReq.Type,
					Level:      Unsupported,
					Severity:   sev,
					Message: fmt.Sprintf("Platform '%s' does not support capability '%s'%s",
						target, capReq.Type, formatRequired(capReq.Required)),
					Recommendation: suggestFallback(target, capReq),
				})

			case Conditional:
				if len(capReq.Permissions) == 0 && requiresPermission(capReq.Type) {
					violations = append(violations, CapabilityViolation{
						Platform:   target,
						Capability: capReq.Type,
						Level:      Conditional,
						Severity:   "WARNING",
						Message: fmt.Sprintf("Capability '%s' on '%s' requires user permissions, but no manifest permissions are specified",
							capReq.Type, target),
						Recommendation: fmt.Sprintf("Declare required OS permissions in .capabilities for '%s'", capReq.Type),
					})
				}
			}
		}
	}

	return violations
}

func formatRequired(req bool) string {
	if req {
		return " (marked as Required)"
	}
	return " (marked as Optional)"
}

func requiresPermission(capType ir.CapabilityType) bool {
	switch capType {
	case ir.CapabilityCamera, ir.CapabilityMicrophone, ir.CapabilityLocation,
		ir.CapabilityNotifications, ir.CapabilityBluetooth:
		return true
	default:
		return false
	}
}

func suggestFallback(target ir.TargetPlatform, capReq *ir.CapabilityIR) string {
	if capReq.FallbackAction != "" {
		return fmt.Sprintf("Fallback action '%s' configured", capReq.FallbackAction)
	}

	if target == ir.TargetAndroidTV && (capReq.Type == ir.CapabilityTouch || capReq.Type == ir.CapabilityMouse) {
		return "Adapt UI with D-pad navigation and focus management for TV remote interaction."
	}
	if target == ir.TargetWeb && capReq.Type == ir.CapabilityTVRemote {
		return "Bind arrow keys or gamepad API as web keyboard navigation fallback."
	}
	if capReq.Type == ir.CapabilityBiometrics {
		return "Provide password/PIN fallback authentication for platforms without biometric sensors."
	}

	return fmt.Sprintf("Mark capability '%s' as optional (required: false) or define a platform fallback.", capReq.Type)
}

// FormatViolations converts violations into human-readable text.
func FormatViolations(violations []CapabilityViolation) string {
	if len(violations) == 0 {
		return "✓ All requested capabilities are compatible with selected targets."
	}

	var sb strings.Builder
	for _, v := range violations {
		color := "\x1b[1;33m" // Yellow for warning
		if v.Severity == "ERROR" {
			color = "\x1b[1;31m" // Red for error
		}
		sb.WriteString(fmt.Sprintf("  %s[%s]\x1b[0m [%s] %s: %s\n    ➜ Recommendation: %s\n",
			color, v.Severity, v.Platform, v.Capability, v.Message, v.Recommendation))
	}
	return sb.String()
}

// AndroidPermissions returns all Android manifest permission strings corresponding to the given capabilities.
func AndroidPermissions(caps []*ir.CapabilityIR) []string {
	permSet := make(map[string]bool)
	// Base Internet permission always granted for networked apps
	permSet["android.permission.INTERNET"] = true
	permSet["android.permission.ACCESS_NETWORK_STATE"] = true

	for _, c := range caps {
		switch c.Type {
		case ir.CapabilityCamera:
			permSet["android.permission.CAMERA"] = true
		case ir.CapabilityMicrophone:
			permSet["android.permission.RECORD_AUDIO"] = true
		case ir.CapabilityLocation:
			permSet["android.permission.ACCESS_FINE_LOCATION"] = true
			permSet["android.permission.ACCESS_COARSE_LOCATION"] = true
		case ir.CapabilityNotifications:
			permSet["android.permission.POST_NOTIFICATIONS"] = true
		case ir.CapabilityBluetooth:
			permSet["android.permission.BLUETOOTH_CONNECT"] = true
			permSet["android.permission.BLUETOOTH_SCAN"] = true
		case ir.CapabilityBiometrics:
			permSet["android.permission.USE_BIOMETRIC"] = true
		}
		for _, custom := range c.Permissions {
			if strings.HasPrefix(custom, "android.permission.") {
				permSet[custom] = true
			}
		}
	}

	var result []string
	for p := range permSet {
		result = append(result, p)
	}
	return result
}

// IOSUsageDescriptions returns key-value pairs for Info.plist privacy descriptions.
func IOSUsageDescriptions(caps []*ir.CapabilityIR) map[string]string {
	desc := make(map[string]string)
	for _, c := range caps {
		switch c.Type {
		case ir.CapabilityCamera:
			desc["NSCameraUsageDescription"] = "This application requires camera access for capturing photos and video."
		case ir.CapabilityMicrophone:
			desc["NSMicrophoneUsageDescription"] = "This application requires microphone access for audio recording."
		case ir.CapabilityLocation:
			desc["NSLocationWhenInUseUsageDescription"] = "This application requires location access to provide location-based features."
		case ir.CapabilityBiometrics:
			desc["NSFaceIDUsageDescription"] = "This application requires Face ID / Touch ID for biometric authentication."
		case ir.CapabilityBluetooth:
			desc["NSBluetoothAlwaysUsageDescription"] = "This application requires Bluetooth for peripheral connectivity."
		}
	}
	return desc
}

