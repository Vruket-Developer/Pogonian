package architecture

import (
	"fmt"
	"strings"

	"pogonian/pkg/ir"
)

// Violation describes an architectural drift or rule break.
type Violation struct {
	Severity   string `json:"severity"` // "ERROR" or "WARNING"
	Layer      string `json:"layer"`
	Component  string `json:"component"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// Validator enforces Pogonian architectural boundaries and clean layer separation.
type Validator struct {
	Strict      bool
	Enforcement string // "warn", "strict", "off"
}

// NewValidator creates an architecture validator from a boolean strict flag.
func NewValidator(strict bool) *Validator {
	enforcement := "warn"
	if strict {
		enforcement = "strict"
	}
	return &Validator{Strict: strict, Enforcement: enforcement}
}

// NewValidatorWithEnforcement creates an architecture validator with an explicit enforcement mode.
func NewValidatorWithEnforcement(enforcement string) *Validator {
	normalized := strings.ToLower(strings.TrimSpace(enforcement))
	if normalized == "" {
		normalized = "warn"
	}
	return &Validator{
		Strict:      normalized == "strict",
		Enforcement: normalized,
	}
}

// Validate executes all architectural rule checks against the IR and graph.
func (v *Validator) Validate(app *ir.Application, graph *DependencyGraph) []Violation {
	violations := make([]Violation, 0)

	// Contract 1: Handlers must NOT directly access Database (Layer Isolation Violation)
	for _, h := range app.Handlers {
		if h.DirectDBAccess {
			violations = append(violations, Violation{
				Severity:   "ERROR",
				Layer:      "Handler",
				Component:  fmt.Sprintf("%s (%s)", h.Name, h.File),
				Message:    "Handler directly references database (*sql.DB)",
				Suggestion: "Remove direct DB access. Route request through a Service and Repository.",
			})
		}
	}

	// Contract 2: Check Circular Dependencies in graph (Cycle Detection)
	cycles := graph.DetectCycles()
	for _, cycle := range cycles {
		cycleStr := strings.Join(cycle, " ➔ ")
		violations = append(violations, Violation{
			Severity:   "ERROR",
			Layer:      "Cross-Module",
			Component:  cycle[0],
			Message:    fmt.Sprintf("Circular dependency detected: %s", cycleStr),
			Suggestion: "Refactor shared logic into a common domain service to break the cycle.",
		})
	}

	// Contract 3: Ensure Handlers call Services (Encapsulation)
	for _, h := range app.Handlers {
		if len(h.CallsServices) == 0 && !h.DirectDBAccess {
			sev := "WARNING"
			if v.Strict {
				sev = "ERROR"
			}
			violations = append(violations, Violation{
				Severity:   sev,
				Layer:      "Handler",
				Component:  fmt.Sprintf("%s (%s)", h.Name, h.File),
				Message:    "Handler has no Service dependencies attached",
				Suggestion: "Attach a domain Service to encapsulate business logic.",
			})
		}
	}

	// Contract 4: Ensure Services call Repositories when performing data operations
	for _, s := range app.Services {
		if len(s.CallsRepos) == 0 {
			sev := "WARNING"
			if v.Strict {
				sev = "ERROR"
			}
			violations = append(violations, Violation{
				Severity:   sev,
				Layer:      "Service",
				Component:  fmt.Sprintf("%s (%s)", s.Name, s.File),
				Message:    "Service has no Repository dependencies attached",
				Suggestion: "Attach a Repository to persist domain entities.",
			})
		}
	}

	return violations
}
