package position

import (
	"context"
	"strings"
	"testing"

	"brale-core/internal/transport/notify"
)

type captureSender struct {
	msg notify.Message
}

func (c *captureSender) Send(ctx context.Context, msg notify.Message) error {
	c.msg = msg
	return nil
}

func TestRiskPlanUpdateNotificationPayload(t *testing.T) {
	sender := &captureSender{}
	manager := notify.NewTestManager(sender)
	notice := notify.RiskPlanUpdateNotice{
		Symbol:         "BTCUSDT",
		Direction:      "long",
		EntryPrice:     100,
		OldStop:        90,
		NewStop:        95,
		TakeProfits:    []float64{110, 120},
		Source:         "monitor-tighten",
		MarkPrice:      102,
		ATR:            2,
		Volatility:     0.08,
		GateSatisfied:  true,
		ScoreTotal:     3.5,
		ScoreThreshold: 3,
		ScoreBreakdown: []notify.RiskPlanUpdateScoreItem{
			{Signal: "monitor_tag", Weight: 2, Value: "true", Contribution: 2},
		},
		ParseOK:       true,
		TightenReason: "monitor-tighten",
		TPTightened:   true,
	}

	if err := manager.SendRiskPlanUpdate(context.Background(), notice); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := sender.msg.Markdown
	assertContains(t, body, formatNoticeLine("volatility", "0.08"))
	assertContains(t, body, formatNoticeLine("gate_satisfied", "true"))
	assertContains(t, body, formatNoticeLine("score_total", "3.5"))
	assertContains(t, body, formatNoticeLine("score_threshold", "3"))
	assertContains(t, body, formatNoticeLine("score_breakdown", "monitor_tag=true (w=2, c=2)"))
	assertContains(t, body, formatNoticeLine("parse_ok", "true"))
	assertContains(t, body, formatNoticeLine("tighten_reason", "monitor-tighten"))
	assertContains(t, body, formatNoticeLine("tp_tightened", "true"))
}

func formatNoticeLine(labelKey string, value string) string {
	return "- " + notify.Label(labelKey) + ": " + value
}

func assertContains(t *testing.T, text string, want string) {
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in %q", want, text)
	}
}
