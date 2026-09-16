package state

import (
	"fmt"

	"pogonian/pkg/capability"
	"pogonian/pkg/ir"
)

// CompatibilityStatus defines the tripartite classification of destination hardware suitability.
type CompatibilityStatus string

const (
	StatusCompatible          CompatibilityStatus = "COMPATIBLE"
	StatusPartiallyCompatible CompatibilityStatus = "PARTIALLY_COMPATIBLE"
	StatusIncompatible        CompatibilityStatus = "INCOMPATIBLE"
)

// CapabilityAdaptationPlan details how application input and features adapt when migrating across device classes.
type CapabilityAdaptationPlan struct {
	OriginPlatform  ir.TargetPlatform            `json:"origin_platform"`
	TargetPlatform  ir.TargetPlatform            `json:"target_platform"`
	InputMode       string                       `json:"input_mode"` // "keyboard_mouse", "direct_touch", "dpad_spatial"
	Status          CompatibilityStatus          `json:"status"`     // COMPATIBLE, PARTIALLY_COMPATIBLE, INCOMPATIBLE
	CompatibleCaps  []ir.CapabilityType          `json:"compatible_caps"`
	DegradedCaps    []ir.CapabilityType          `json:"degraded_caps,omitempty"`
	UnsupportedCaps []ir.CapabilityType          `json:"unsupported_caps,omitempty"`
	FallbackActions map[ir.CapabilityType]string `json:"fallback_actions,omitempty"`
	IsCompatible    bool                         `json:"is_compatible"`
}

// AssessCapabilityAdaptation determines how the application adapts to the destination device's hardware profile.
func AssessCapabilityAdaptation(snapshot *ir.StateSnapshot, targetApp *ir.Application, targetPlatform ir.TargetPlatform) (*CapabilityAdaptationPlan, error) {
	origin := snapshot.OriginPlatform
	plan := &CapabilityAdaptationPlan{
		OriginPlatform:  origin,
		TargetPlatform:  targetPlatform,
		Status:          StatusCompatible,
		CompatibleCaps:  make([]ir.CapabilityType, 0),
		DegradedCaps:    make([]ir.CapabilityType, 0),
		UnsupportedCaps: make([]ir.CapabilityType, 0),
		FallbackActions: make(map[ir.CapabilityType]string),
		IsCompatible:    true,
	}

	// 1. Determine Input Mode Adaptation
	switch {
	case targetPlatform.IsTVTarget():
		plan.InputMode = "dpad_spatial"
	case targetPlatform.IsMobileTarget():
		plan.InputMode = "direct_touch"
	default:
		plan.InputMode = "keyboard_mouse"
	}

	// 2. Hardware profile capabilities mapping from canonical capability matrix
	matrix := capability.DefaultCapabilityMatrix()
	targetProfile, exists := matrix[targetPlatform]
	if !exists {
		targetProfile = make(map[ir.CapabilityType]capability.SupportLevel)
	}

	// Build fallback lookup from application declared capabilities
	appFallbackMap := make(map[ir.CapabilityType]string)
	if targetApp != nil {
		for _, c := range targetApp.Capabilities {
			if c.FallbackAction != "" {
				appFallbackMap[c.Type] = c.FallbackAction
			}
		}
	}

	// 3. Compare required capabilities from origin snapshot against destination hardware profile
	for _, reqCap := range snapshot.RequiredCapabilities {
		level, hasCap := targetProfile[reqCap]
		if hasCap && (level == capability.Supported || level == capability.Conditional) {
			plan.CompatibleCaps = append(plan.CompatibleCaps, reqCap)
		} else {
			// Capability is Unsupported on target hardware profile
			if fallbackAction, hasFallback := appFallbackMap[reqCap]; hasFallback {
				plan.DegradedCaps = append(plan.DegradedCaps, reqCap)
				plan.FallbackActions[reqCap] = fallbackAction
				plan.Status = StatusPartiallyCompatible
			} else {
				// Non-negotiable missing capability without fallback
				plan.UnsupportedCaps = append(plan.UnsupportedCaps, reqCap)
				plan.Status = StatusIncompatible
				plan.IsCompatible = false
				levelStr := string(level)
				if !hasCap {
					levelStr = "UNSUPPORTED"
				}
				return plan, fmt.Errorf("capability mismatch: target platform '%s' lacks capability '%s' required by application (support level: %s)",
					targetPlatform, reqCap, levelStr)
			}
		}
	}

	return plan, nil
}

// AdaptUIForTargetPlatform adapts an abstract UI element tree for the target device class.
func AdaptUIForTargetPlatform(root *ir.UIElementIR, targetPlatform ir.TargetPlatform) *ir.UIElementIR {
	if root == nil {
		return nil
	}

	adapted := cloneUIElement(root)

	// Adapt nodes recursively
	var walkAndAdapt func(elem *ir.UIElementIR)
	walkAndAdapt = func(elem *ir.UIElementIR) {
		if elem == nil {
			return
		}
		if targetPlatform.IsTVTarget() {
			// TV: Ensure buttons and cards have focusable markers and 10-foot sizing
			if elem.Kind == ir.UIElementButton || elem.Kind == ir.UIElementInput || elem.Kind == ir.UIElementCard {
				if elem.Properties == nil {
					elem.Properties = make(map[string]string)
				}
				elem.Properties["focusable"] = "true"
				elem.Properties["min_height_dp"] = "48"
			}
		} else if targetPlatform.IsMobileTarget() {
			// Mobile: Ensure minimum 48dp touch target sizing
			if elem.Kind == ir.UIElementButton || elem.Kind == ir.UIElementInput {
				if elem.Properties == nil {
					elem.Properties = make(map[string]string)
				}
				elem.Properties["touch_target_dp"] = "48"
			}
		}
		for _, child := range elem.Children {
			walkAndAdapt(child)
		}
	}
	walkAndAdapt(adapted)

	return adapted
}

func cloneUIElement(elem *ir.UIElementIR) *ir.UIElementIR {
	if elem == nil {
		return nil
	}
	c := &ir.UIElementIR{
		ID:         elem.ID,
		Kind:       elem.Kind,
		Text:       elem.Text,
		Binding:    elem.Binding,
		Events:     elem.Events,
		Properties: make(map[string]string),
		Children:   make([]*ir.UIElementIR, 0, len(elem.Children)),
	}
	for k, v := range elem.Properties {
		c.Properties[k] = v
	}
	for _, child := range elem.Children {
		c.Children = append(c.Children, cloneUIElement(child))
	}
	return c
}
