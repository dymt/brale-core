package position

import (
	"context"
	"math"
	"strings"
	"testing"

	"brale-core/internal/store"
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
	assertContains(t, body, "交易对：BTCUSDT")
	assertContains(t, body, "方向：long")
	assertContains(t, body, "原止损：90 → 95")
	assertContains(t, body, "来源：monitor-tighten")
	assertContains(t, body, "标记价：102")
	assertContains(t, body, "评分：3.5 / 3 · 通过：true")
	assertContains(t, body, "收紧原因：monitor-tighten")
}

func formatNoticeLine(labelKey string, value string) string {
	return "- " + notify.Label(labelKey) + ": " + value
}

func assertContains(t *testing.T, text string, want string) {
	if !strings.Contains(text, want) {
		t.Fatalf("expected %q in %q", want, text)
	}
}

type trackedCloseNotifier struct {
	notice PositionCloseNotice
}

func (n *trackedCloseNotifier) SendError(context.Context, ErrorNotice) error {
	return nil
}

func (n *trackedCloseNotifier) SendPositionClose(_ context.Context, notice PositionCloseNotice) error {
	n.notice = notice
	return nil
}

func TestTrackedLogAndNotifyClosePrefersRequestPositionQty(t *testing.T) {
	t.Parallel()

	notifier := &trackedCloseNotifier{}
	svc := &PositionService{Notifier: notifier}
	pos := store.PositionRecord{
		Symbol:     "ETHUSDT",
		Side:       "long",
		Qty:        40,
		AvgEntry:   3200,
		PositionID: "pos-2",
	}
	request := CloseRequest{
		RequestedCloseQty: 20,
		EffectiveCloseQty: 20,
		PositionQty:       55,
	}

	svc.logAndNotifyClose(context.Background(), pos, "tp_1_hit", 3300, request, "REDUCE")

	if math.Abs(notifier.notice.Qty-55) > 1e-12 {
		t.Fatalf("qty=%v want 55", notifier.notice.Qty)
	}
}
