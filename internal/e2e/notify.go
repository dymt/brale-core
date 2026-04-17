package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"brale-core/internal/transport/notify"
)

const telegramReportRuneLimit = 3500

func sendTelegramReport(ctx context.Context, token string, chatID int64, report RunReport, reportDir string) error {
	sender, err := notify.NewTelegramSender(notify.TelegramConfig{
		Enabled: true,
		Token:   strings.TrimSpace(token),
		ChatID:  chatID,
	})
	if err != nil {
		return err
	}
	return sender.Send(ctx, notify.Message{Plain: formatTelegramReport(report, reportDir)})
}

func formatTelegramReport(report RunReport, reportDir string) string {
	lines := []string{
		fmt.Sprintf("E2E %s", reportStatusText(report.Passed)),
		fmt.Sprintf("Run: %s", strings.TrimSpace(report.RunID)),
		fmt.Sprintf("Profile: %s", strings.TrimSpace(report.Profile)),
	}
	if symbol := strings.TrimSpace(report.Symbol); symbol != "" {
		lines = append(lines, fmt.Sprintf("Symbol: %s", symbol))
	}
	lines = append(lines, fmt.Sprintf("Duration: %s", report.Duration.Round(time.Millisecond)))

	if len(report.Suites) > 0 {
		lines = append(lines, "", "Suites:")
		for _, suite := range report.Suites {
			lines = append(lines, fmt.Sprintf("- %s %s (%s)", suiteStatusText(suite.Status), suite.Name, suite.Duration.Round(time.Millisecond)))
			if errText := strings.TrimSpace(suite.Error); errText != "" {
				lines = append(lines, fmt.Sprintf("  error: %s", errText))
			}
			failed := failedChecksText(suite.Checks)
			if failed != "" {
				lines = append(lines, fmt.Sprintf("  failed: %s", failed))
			}
		}
	}

	if path := strings.TrimSpace(reportDir); path != "" {
		lines = append(lines, "", fmt.Sprintf("Report: %s", path))
	}

	return truncateTelegramText(strings.Join(lines, "\n"))
}

func reportStatusText(passed bool) string {
	if passed {
		return "PASSED"
	}
	return "FAILED"
}

func suiteStatusText(status SuiteStatus) string {
	switch status {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusSkip:
		return "SKIP"
	case StatusBlocked:
		return "BLOCKED"
	default:
		return string(status)
	}
}

func failedChecksText(checks []CheckResult) string {
	failed := make([]string, 0, len(checks))
	for _, check := range checks {
		if check.Passed {
			continue
		}
		text := strings.TrimSpace(check.Name)
		if msg := strings.TrimSpace(check.Message); msg != "" {
			text = fmt.Sprintf("%s (%s)", text, msg)
		}
		failed = append(failed, text)
	}
	return strings.Join(failed, "; ")
}

func truncateTelegramText(text string) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= telegramReportRuneLimit {
		return string(runes)
	}
	limit := telegramReportRuneLimit - len([]rune("\n...(truncated)"))
	if limit < 0 {
		limit = 0
	}
	return string(runes[:limit]) + "\n...(truncated)"
}
