package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Runner orchestrates E2E suite execution.
type Runner struct {
	Config RunConfig
}

// NewRunner creates a runner from the given config.
func NewRunner(cfg RunConfig) *Runner {
	return &Runner{Config: cfg}
}

// Run executes all requested suites and returns a report.
func (r *Runner) Run() (RunReport, error) {
	start := time.Now()
	runID := fmt.Sprintf("e2e-%s-%s", r.Config.Profile, start.Format("20060102-150405"))

	ectx, err := NewContext(r.Config)
	if err != nil {
		return RunReport{}, fmt.Errorf("initialize context: %w", err)
	}
	defer ectx.Cancel()

	suites := r.Config.Suites
	if len(suites) == 0 {
		suites = []string{"ctl"}
	}

	report := RunReport{
		RunID:     runID,
		Profile:   r.Config.Profile,
		Symbol:    r.Config.Symbol,
		StartedAt: start,
		Passed:    true,
	}

	for _, name := range suites {
		name = strings.TrimSpace(name)
		suite, err := GetSuite(name)
		if err != nil {
			report.Suites = append(report.Suites, SuiteResult{
				Name:      name,
				Status:    StatusFail,
				Error:     err.Error(),
				StartedAt: time.Now(),
			})
			report.Passed = false
			continue
		}

		fmt.Printf("\n━━━ Suite: %s ━━━\n", name)
		result := suite.Run(ectx)
		report.Suites = append(report.Suites, result)

		if result.Status == StatusFail {
			report.Passed = false
		}

		statusIcon := "✓"
		switch result.Status {
		case StatusFail:
			statusIcon = "✗"
		case StatusSkip:
			statusIcon = "⊘"
		case StatusBlocked:
			statusIcon = "⊘"
		}
		fmt.Printf("  %s %s [%s] %s\n", statusIcon, name, result.Status, result.Duration.Round(time.Millisecond))
		if result.Error != "" {
			fmt.Printf("    Error: %s\n", result.Error)
		}
		for _, c := range result.Checks {
			mark := "  ✓"
			if !c.Passed {
				mark = "  ✗"
			}
			fmt.Printf("  %s %s", mark, c.Name)
			if c.Message != "" {
				fmt.Printf(": %s", c.Message)
			}
			fmt.Println()
		}
	}

	report.Duration = time.Since(start)

	reportDir := ""
	// Write report if dir specified
	if r.Config.ReportDir != "" {
		reportDir = fmt.Sprintf("%s/%s", r.Config.ReportDir, runID)
		if err := WriteReport(reportDir, report); err != nil {
			fmt.Printf("[WARN] write report: %v\n", err)
		} else {
			fmt.Printf("\n[OK] Report written to %s\n", reportDir)
		}
	}

	if r.Config.TelegramToken != "" || r.Config.TelegramChatID != 0 {
		notifyCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := sendTelegramReport(notifyCtx, r.Config.TelegramToken, r.Config.TelegramChatID, report, reportDir); err != nil {
			fmt.Printf("[WARN] telegram report: %v\n", err)
		} else {
			fmt.Println("[OK] Telegram report sent")
		}
	}

	return report, nil
}
