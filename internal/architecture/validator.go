package architecture

import (
	"fmt"
	"strings"

	"pogonian/internal/ir"
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
	Strict bool
}

// NewValidator creates an architecture validator.
func NewValidator(strict bool) *Validator {
	return &Validator{Strict: strict}
}

// Validate executes all architectural rule checks against the IR and graph.
func (v *Validator) Validate(app *ir.Application, graph *DependencyGraph) []Violation {
	violations := make([]Violation, 0)

	// Rule 1: Handlers must NOT directly access Database (Layer Violation)
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

	// Rule 2: Check Circular Dependencies in graph
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

	// Rule 3: Ensure Handlers call Services
	for _, h := range app.Handlers {
		if len(h.CallsServices) == 0 && !h.DirectDBAccess {
			violations = append(violations, Violation{
				Severity:   "WARNING",
				Layer:      "Handler",
				Component:  fmt.Sprintf("%s (%s)", h.Name, h.File),
				Message:    "Handler has no Service dependencies attached",
				Suggestion: "Attach a domain Service to handle business logic.",
			})
		}
	}

	return violations
}
