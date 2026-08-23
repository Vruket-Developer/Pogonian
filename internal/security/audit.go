package security

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding describes a security vulnerability or weakness.
type Finding struct {
	Type           string `json:"type"`
	Severity       string `json:"severity"` // "HIGH", "MEDIUM", "LOW"
	File           string `json:"file"`
	Line           int    `json:"line"`
	Description    string `json:"description"`
	Recommendation string `json:"recommendation"`
}

// AuditReport aggregates all findings and computes a security posture score.
type AuditReport struct {
	ProjectDir string     `json:"project_dir"`
	Score      int        `json:"score"` // 0 - 100
	Grade      string     `json:"grade"` // A+, A, B, C, F
	Findings   []*Finding `json:"findings"`
}

var (
	awsKeyRegex      = regexp.MustCompile(`(?i)(AKIA|ASIA)[0-9A-Z]{16}`)
	privateKeyRegex  = regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)
	secretAssignRe   = regexp.MustCompile(`(?i)(api_key|secret_key|auth_token|password)\s*(?::=|=)\s*["']([^"']{8,})["']`)
	sqlConcatRegex   = regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE).*\+.*`)
	sqlSprintfRegex  = regexp.MustCompile(`(?i)fmt\.Sprintf\(["'](SELECT|INSERT|UPDATE|DELETE)`)
)

// AuditProject performs static analysis on a Pogonian project for security vulnerabilities.
func AuditProject(projectDir string) (*AuditReport, error) {
	report := &AuditReport{
		ProjectDir: projectDir,
		Findings:   make([]*Finding, 0),
		Score:      100,
		Grade:      "A+",
	}

	_ = filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(projectDir, path)
		if strings.Contains(rel, ".git") || strings.Contains(rel, "node_modules") || strings.Contains(rel, "bin") ||
			strings.Contains(rel, "tests") || strings.Contains(rel, "testlab") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fileContent := string(contentBytes)
		lines := strings.Split(fileContent, "\n")

		for lineNum, lineText := range lines {
			lineNum = lineNum + 1

			// 1. Secret Exposure Checks
			if awsKeyRegex.MatchString(lineText) {
				report.Findings = append(report.Findings, &Finding{
					Type:           "SECRET_EXPOSURE",
					Severity:       "HIGH",
					File:           rel,
					Line:           lineNum,
					Description:    "Possible AWS Access Key exposed in source code",
					Recommendation: "Store credentials in environment variables or a secure secret vault.",
				})
				report.Score -= 25
			}

			if privateKeyRegex.MatchString(lineText) {
				report.Findings = append(report.Findings, &Finding{
					Type:           "SECRET_EXPOSURE",
					Severity:       "HIGH",
					File:           rel,
					Line:           lineNum,
					Description:    "Private Key header detected in source code",
					Recommendation: "Never commit private keys to repository files.",
				})
				report.Score -= 30
			}

			if secretAssignRe.MatchString(lineText) {
				report.Findings = append(report.Findings, &Finding{
					Type:           "SECRET_EXPOSURE",
					Severity:       "MEDIUM",
					File:           rel,
					Line:           lineNum,
					Description:    "Hardcoded credential or token assignment detected",
					Recommendation: "Load sensitive tokens from environment variables (os.Getenv).",
				})
				report.Score -= 15
			}

			// 2. SQL Injection Checks (in Go backend files)
			if strings.HasSuffix(path, ".go") {
				if sqlSprintfRegex.MatchString(lineText) || (strings.Contains(lineText, "Query(") && sqlConcatRegex.MatchString(lineText)) {
					report.Findings = append(report.Findings, &Finding{
						Type:           "SQL_INJECTION",
						Severity:       "HIGH",
						File:           rel,
						Line:           lineNum,
						Description:    "Dynamic string concatenation detected in SQL query",
						Recommendation: "Use parameterized queries with placeholder '?' to prevent SQL injection.",
					})
					report.Score -= 25
				}
			}

			// 3. CSRF Protection Check (in HTML forms)
			if strings.HasSuffix(path, ".html") && strings.Contains(lineText, "<form") && strings.Contains(lineText, "method=\"post\"") {
				if !strings.Contains(fileContent, "_csrf") && !strings.Contains(fileContent, "CSRFToken") {
					report.Findings = append(report.Findings, &Finding{
						Type:           "CSRF_VULNERABILITY",
						Severity:       "LOW",
						File:           rel,
						Line:           lineNum,
						Description:    "Form POST missing hidden CSRF input field",
						Recommendation: "Include `<input type=\"hidden\" name=\"_csrf\" value=\"{{ .CSRFToken }}\">`.",
					})
					report.Score -= 5
				}
			}
		}

		return nil
	})

	if report.Score < 0 {
		report.Score = 0
	}

	switch {
	case report.Score >= 95:
		report.Grade = "A+"
	case report.Score >= 85:
		report.Grade = "A"
	case report.Score >= 70:
		report.Grade = "B"
	case report.Score >= 50:
		report.Grade = "C"
	default:
		report.Grade = "F"
	}

	return report, nil
}

// Summary formats the audit report into a colorized terminal report.
func (r *AuditReport) Summary() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("🛡️  \x1b[1;36mPogonian Security Audit Report\x1b[0m\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("  Security Posture Score: \x1b[1;32m%d/100 (Grade %s)\x1b[0m\n", r.Score, r.Grade))
	sb.WriteString(fmt.Sprintf("  Total Findings:         %d\n", len(r.Findings)))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(r.Findings) == 0 {
		sb.WriteString("  ✨ \x1b[1;32m0 vulnerabilities detected! All security checks passed.\x1b[0m\n\n")
		return sb.String()
	}

	for i, f := range r.Findings {
		var sevColor string
		switch f.Severity {
		case "HIGH":
			sevColor = "\x1b[1;31m" // red
		case "MEDIUM":
			sevColor = "\x1b[1;33m" // yellow
		default:
			sevColor = "\x1b[1;34m" // blue
		}

		sb.WriteString(fmt.Sprintf("  %d. %s[%s]\x1b[0m \x1b[1m%s\x1b[0m (%s:%d)\n", i+1, sevColor, f.Severity, f.Description, f.File, f.Line))
		sb.WriteString(fmt.Sprintf("     \x1b[1;30m↳ Recommendation: %s\x1b[0m\n", f.Recommendation))
	}
	sb.WriteString("\n")

	return sb.String()
}
