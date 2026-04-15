package backtest

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

func WriteTextReport(w io.Writer, result *ReplayResult) error {
	if result == nil {
		return fmt.Errorf("replay result is required")
	}
	if _, err := fmt.Fprintf(w, "=== Gate Replay Report: %s ===\n", result.Symbol); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Rounds: total=%d replayable=%d skipped=%d allow=%d veto=%d wait=%d\n",
		result.Metrics.TotalRounds,
		result.Metrics.ReplayableRounds,
		result.Metrics.SkippedCount,
		result.Metrics.AllowCount,
		result.Metrics.VetoCount,
		result.Metrics.WaitCount,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Confusion: TP=%d FP=%d TN=%d FN=%d | Precision: %.1f%% | Win Rate: %.1f%%\n",
		result.Metrics.TruePositive,
		result.Metrics.FalsePositive,
		result.Metrics.TrueNegative,
		result.Metrics.FalseNegative,
		result.Metrics.Precision*100,
		result.Metrics.WinRate*100,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Profit Factor: %.2f | Max Drawdown: %.2f%% | Sharpe: %.2f | Calmar: %.2f\n",
		result.Metrics.ProfitFactor,
		result.Metrics.MaxDrawdown*100,
		result.Metrics.SharpeRatio,
		result.Metrics.CalmarRatio,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Changed Decisions: %d (%.1f%%)\n",
		result.Metrics.ChangedCount,
		result.Metrics.ChangeRate*100,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for _, round := range result.Rounds {
		state := strings.ToUpper(strings.TrimSpace(string(round.State)))
		if state == "" {
			state = "UNKNOWN"
		}
		if round.Skipped {
			if _, err := fmt.Fprintf(w, "#%d [%s] skipped: %s\n", round.SnapshotID, state, round.SkipReason); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "#%d [%s] %s -> %s | price %.4f -> %.4f\n",
			round.SnapshotID,
			state,
			upperOrDash(round.OriginalGate.DecisionAction),
			upperOrDash(round.ReplayedGate.DecisionAction),
			round.PriceAtDecision,
			round.PriceAfter,
		); err != nil {
			return err
		}
	}
	return nil
}

func WriteHTMLReport(w io.Writer, result *ReplayResult) error {
	if result == nil {
		return fmt.Errorf("replay result is required")
	}
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"pct": func(value float64) string { return fmt.Sprintf("%.2f%%", value*100) },
	}).Parse(htmlReplayReportTemplate))
	return tmpl.Execute(w, result)
}

func upperOrDash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return strings.ToUpper(trimmed)
}

const htmlReplayReportTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Gate Replay: {{.Symbol}}</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 24px; color: #111827; }
    table { border-collapse: collapse; width: 100%; margin-bottom: 24px; }
    th, td { border: 1px solid #d1d5db; padding: 8px 10px; text-align: left; }
    th { background: #f3f4f6; }
    .muted { color: #6b7280; }
  </style>
</head>
<body>
  <h1>Gate Replay Report: {{.Symbol}}</h1>
  <table>
    <tr><th>Total Rounds</th><td>{{.Metrics.TotalRounds}}</td><th>Replayable</th><td>{{.Metrics.ReplayableRounds}}</td></tr>
    <tr><th>Skipped</th><td>{{.Metrics.SkippedCount}}</td><th>Changed</th><td>{{.Metrics.ChangedCount}} ({{pct .Metrics.ChangeRate}})</td></tr>
    <tr><th>Allow</th><td>{{.Metrics.AllowCount}}</td><th>Veto</th><td>{{.Metrics.VetoCount}}</td></tr>
    <tr><th>Wait</th><td>{{.Metrics.WaitCount}}</td><th>Precision</th><td>{{pct .Metrics.Precision}}</td></tr>
    <tr><th>Win Rate</th><td>{{pct .Metrics.WinRate}}</td><th>Profit Factor</th><td>{{printf "%.2f" .Metrics.ProfitFactor}}</td></tr>
    <tr><th>Max Drawdown</th><td>{{pct .Metrics.MaxDrawdown}}</td><th>Sharpe / Calmar</th><td>{{printf "%.2f / %.2f" .Metrics.SharpeRatio .Metrics.CalmarRatio}}</td></tr>
  </table>

  <table>
    <thead>
      <tr>
        <th>Snapshot</th>
        <th>State</th>
        <th>Original</th>
        <th>Replay</th>
        <th>Price</th>
        <th>Price After</th>
        <th>Status</th>
      </tr>
    </thead>
    <tbody>
    {{range .Rounds}}
      <tr>
        <td>{{.SnapshotID}}</td>
        <td>{{.State}}</td>
        <td>{{.OriginalGate.DecisionAction}} / {{.OriginalGate.GateReason}}</td>
        <td>{{.ReplayedGate.DecisionAction}} / {{.ReplayedGate.GateReason}}</td>
        <td>{{printf "%.4f" .PriceAtDecision}}</td>
        <td>{{printf "%.4f" .PriceAfter}}</td>
        <td>{{if .Skipped}}SKIPPED: {{.SkipReason}}{{else if .Changed}}CHANGED{{else}}UNCHANGED{{end}}</td>
      </tr>
    {{end}}
    </tbody>
  </table>
</body>
</html>`
