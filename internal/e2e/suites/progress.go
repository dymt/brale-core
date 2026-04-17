package suites

import (
	"strings"
	"time"

	"brale-core/internal/e2e"
)

const (
	decisionProgressHeartbeatInterval = time.Minute
	decisionProgressSummaryRuneLimit  = 120
)

type decisionProgressTracker struct {
	symbol         string
	lastSnapshotID uint
	lastHeartbeat  time.Time
}

func newDecisionProgressTracker(ctx *e2e.Context, symbol string) *decisionProgressTracker {
	tracker := &decisionProgressTracker{
		symbol:        symbol,
		lastHeartbeat: time.Now(),
	}
	if ctx == nil || ctx.Client == nil {
		return tracker
	}
	latest, err := ctx.Client.FetchDecisionLatest(ctx.Ctx, symbol)
	if err == nil {
		tracker.lastSnapshotID = latest.SnapshotID
	}
	return tracker
}

func (t *decisionProgressTracker) Poll(ctx *e2e.Context, phase string) {
	if t == nil || ctx == nil || ctx.Client == nil {
		return
	}
	latest, err := ctx.Client.FetchDecisionLatest(ctx.Ctx, t.symbol)
	if err != nil {
		t.logHeartbeat(ctx, phase)
		return
	}
	if latest.SnapshotID > t.lastSnapshotID {
		t.lastSnapshotID = latest.SnapshotID
		summary := formatDecisionProgressSummary(latest.Summary, latest.Report)
		if summary == "" {
			ctx.Log("%s: new decision snapshot=%d", phase, latest.SnapshotID)
		} else {
			ctx.Log("%s: new decision snapshot=%d summary=%q", phase, latest.SnapshotID, summary)
		}
		t.lastHeartbeat = time.Now()
		return
	}
	t.logHeartbeat(ctx, phase)
}

func (t *decisionProgressTracker) logHeartbeat(ctx *e2e.Context, phase string) {
	if time.Since(t.lastHeartbeat) < decisionProgressHeartbeatInterval {
		return
	}
	if t.lastSnapshotID > 0 {
		ctx.Log("%s: still waiting; last_snapshot=%d", phase, t.lastSnapshotID)
	} else {
		ctx.Log("%s: still waiting; no decision snapshot yet", phase)
	}
	t.lastHeartbeat = time.Now()
}

func formatDecisionProgressSummary(summary, report string) string {
	text := strings.TrimSpace(summary)
	if text == "" {
		text = strings.TrimSpace(report)
	}
	if text == "" {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) <= decisionProgressSummaryRuneLimit {
		return text
	}
	return string(runes[:decisionProgressSummaryRuneLimit-3]) + "..."
}
