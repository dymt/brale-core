package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteReport writes the run report as JSON and a human-readable summary.
func WriteReport(dir string, report RunReport) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}

	// Write JSON report
	jsonPath := filepath.Join(dir, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return fmt.Errorf("write report.json: %w", err)
	}

	// Write summary.md
	mdPath := filepath.Join(dir, "summary.md")
	md := formatSummary(report)
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write summary.md: %w", err)
	}

	return nil
}

func formatSummary(r RunReport) string {
	result := "PASS"
	if !r.Passed {
		result = "FAIL"
	}
	s := fmt.Sprintf("# E2E Report: %s\n\n", r.RunID)
	s += fmt.Sprintf("- **Profile**: %s\n", r.Profile)
	s += fmt.Sprintf("- **Symbol**: %s\n", r.Symbol)
	s += fmt.Sprintf("- **Started**: %s\n", r.StartedAt.Format(time.RFC3339))
	s += fmt.Sprintf("- **Duration**: %s\n", r.Duration.Round(time.Second))
	s += fmt.Sprintf("- **Result**: %s\n\n", result)
	s += "## Suites\n\n"
	s += "| Suite | Status | Duration | Error |\n"
	s += "|-------|--------|----------|-------|\n"
	for _, sr := range r.Suites {
		errMsg := "-"
		if sr.Error != "" {
			errMsg = sr.Error
			if len(errMsg) > 80 {
				errMsg = errMsg[:80] + "..."
			}
		}
		s += fmt.Sprintf("| %s | %s | %s | %s |\n",
			sr.Name, sr.Status,
			sr.Duration.Round(time.Millisecond),
			errMsg)
	}
	s += "\n## Checks\n\n"
	for _, sr := range r.Suites {
		if len(sr.Checks) == 0 {
			continue
		}
		s += fmt.Sprintf("### %s\n\n", sr.Name)
		for _, c := range sr.Checks {
			mark := "✓"
			if !c.Passed {
				mark = "✗"
			}
			s += fmt.Sprintf("- %s %s", mark, c.Name)
			if c.Message != "" {
				s += fmt.Sprintf(": %s", c.Message)
			}
			s += "\n"
		}
		s += "\n"
	}
	return s
}
