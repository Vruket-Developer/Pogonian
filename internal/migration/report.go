package migration

import (
	"fmt"
	"strings"
)

// Report provides detailed migration confidence metrics, component inventories, and behavioral analysis.
type Report struct {
	AppName          string              `json:"app_name"`
	SourceFramework  string              `json:"source_framework"`
	AutomaticPercent float64             `json:"automatic_percent"`
	AIAssisted       float64             `json:"ai_assisted_percent"`
	ManualPercent    float64             `json:"manual_percent"`
	RoutesCount      int                 `json:"routes_count"`
	ComponentsCount  int                 `json:"components_count"`
	APIsCount        int                 `json:"apis_count"`
	ConvertedRoutes  []string            `json:"converted_routes"`
	Warnings         []string            `json:"warnings"`
	Behavior         *BehavioralAnalysis `json:"behavior,omitempty"`
}

// Summary formats the migration report into a human-readable summary.
func (r *Report) Summary() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("  \x1b[1;36mPogonian Migration Report\x1b[0m — %s (%s ➔ Pogonian)\n", r.AppName, r.SourceFramework))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString("  \x1b[1mConfidence Scores:\x1b[0m\n")
	sb.WriteString(fmt.Sprintf("    \x1b[1;32m✓ Automatic:\x1b[0m    %.0f%%\n", r.AutomaticPercent))
	sb.WriteString(fmt.Sprintf("    \x1b[1;33m⚡ AI-assisted:\x1b[0m  %.0f%%\n", r.AIAssisted))
	sb.WriteString(fmt.Sprintf("    \x1b[1;31m⚠ Manual Review:\x1b[0m %.0f%%\n\n", r.ManualPercent))

	sb.WriteString("  \x1b[1mConverted Assets:\x1b[0m\n")
	sb.WriteString(fmt.Sprintf("    • %d Routes / Pages\n", r.RoutesCount))
	sb.WriteString(fmt.Sprintf("    • %d Components\n", r.ComponentsCount))
	sb.WriteString(fmt.Sprintf("    • %d Backend API Endpoints\n\n", r.APIsCount))

	if r.Behavior != nil {
		sb.WriteString("  \x1b[1mBehavioral Interaction Mapping:\x1b[0m\n")
		sb.WriteString(fmt.Sprintf("    • %d Client UI Interactions (Mapped to globals.js progressive handlers)\n", r.Behavior.ClientInteractions))
		sb.WriteString(fmt.Sprintf("    • %d Data Fetching Operations (Mapped to Go Server Handlers)\n", r.Behavior.ServerDataLoaders))
		sb.WriteString(fmt.Sprintf("    • %d State Mutations (Mapped to Pogonian Form Actions)\n\n", r.Behavior.FormMutations))
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("  \x1b[1;33mAction Items & Warnings:\x1b[0m\n")
		for _, w := range r.Warnings {
			sb.WriteString("    ⚠ " + w + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
