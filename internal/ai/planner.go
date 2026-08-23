package ai

import (
	"fmt"
	"strings"
)

// PlanItem represents a step in an architectural AI feature implementation plan.
type PlanItem struct {
	Step        int    `json:"step"`
	Layer       string `json:"layer"`
	File        string `json:"file"`
	Action      string `json:"action"` // "CREATE", "MODIFY", "MIGRATE"
	Description string `json:"description"`
}

// AIPlan contains the complete architectural change plan.
type AIPlan struct {
	Prompt      string      `json:"prompt"`
	FeatureName string      `json:"feature_name"`
	Steps       []*PlanItem `json:"steps"`
	Explanation string      `json:"explanation"`
}

// GeneratePlan generates a structured multi-layer architecture implementation plan.
func GeneratePlan(prompt string, ctx *AIContext) *AIPlan {
	featureName := deriveFeatureName(prompt)
	featureLower := strings.ToLower(featureName)
	featurePascal := strings.ToUpper(featureName[:1]) + featureName[1:]

	steps := []*PlanItem{
		{
			Step:        1,
			Layer:       "Database Migration",
			File:        fmt.Sprintf("migrations/002_create_%s.sql", featureLower),
			Action:      "MIGRATE",
			Description: fmt.Sprintf("Create SQLite table `%s` with foreign keys and timestamp columns.", featureLower),
		},
		{
			Step:        2,
			Layer:       "Domain Entity",
			File:        fmt.Sprintf("server/%s/entity.go", featureLower),
			Action:      "CREATE",
			Description: fmt.Sprintf("Define `%s` struct with ID, attributes, and JSON tags.", featurePascal),
		},
		{
			Step:        3,
			Layer:       "Repository (Data Access)",
			File:        fmt.Sprintf("server/%s/repository.go", featureLower),
			Action:      "CREATE",
			Description: fmt.Sprintf("Implement SQL queries (`FindAll`, `FindByID`, `Create`, `Delete`) for `%s`.", featurePascal),
		},
		{
			Step:        4,
			Layer:       "Service (Business Logic)",
			File:        fmt.Sprintf("server/%s/service.go", featureLower),
			Action:      "CREATE",
			Description: fmt.Sprintf("Implement business validation and domain logic for `%s`.", featurePascal),
		},
		{
			Step:        5,
			Layer:       "Handler (HTTP Actions)",
			File:        fmt.Sprintf("server/%s/handler.go", featureLower),
			Action:      "CREATE",
			Description: fmt.Sprintf("Implement HTTP action handlers and data loaders for `/%s` routes.", featureLower),
		},
		{
			Step:        6,
			Layer:       "Frontend Route",
			File:        fmt.Sprintf("app/%s/page.html", featureLower),
			Action:      "CREATE",
			Description: fmt.Sprintf("Create %s filesystem page with responsive list and <form method=\"post\"> creation.", featurePascal),
		},
		{
			Step:        7,
			Layer:       "Architecture Spec",
			File:        ".pogonian/architecture/app.ai",
			Action:      "MODIFY",
			Description: fmt.Sprintf("Register module `%s` in architecture metadata.", featureLower),
		},
	}

	return &AIPlan{
		Prompt:      prompt,
		FeatureName: featureName,
		Steps:       steps,
		Explanation: fmt.Sprintf("Architecture-driven plan enforcing Pogonian flow: Page → Handler → Service → Repository → SQLite Database for '%s'.", prompt),
	}
}

func deriveFeatureName(prompt string) string {
	cleaned := strings.ToLower(prompt)
	for _, prefix := range []string{"add ", "create ", "implement ", "build ", "new "} {
		cleaned = strings.TrimPrefix(cleaned, prefix)
	}
	for _, suffix := range []string{" functionality", " feature", " support", " module"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	words := strings.Fields(cleaned)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return "feature"
}
