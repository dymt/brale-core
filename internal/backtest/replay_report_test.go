package backtest

import (
	"bytes"
	"strings"
	"testing"

	"brale-core/internal/decision/fsm"
)

func TestWriteTextReport(t *testing.T) {
	result := &ReplayResult{
		Symbol: "BTCUSDT",
		Rounds: []ReplayRound{
			{
				SnapshotID:      101,
				State:           fsm.StateFlat,
				Changed:         true,
				OriginalGate:    replayTestGate("WAIT", "QUALITY_TOO_LOW", "long", 0, 0.4),
				ReplayedGate:    replayTestGate("ALLOW", "ALLOW", "long", 3, 0.8),
				PriceAtDecision: 100,
				PriceAfter:      120,
			},
		},
	}
	result.Metrics = computeMetrics(result.Rounds)

	var buf bytes.Buffer
	if err := WriteTextReport(&buf, result); err != nil {
		t.Fatalf("WriteTextReport: %v", err)
	}
	out := buf.String()
	for _, needle := range []string{
		"Gate Replay Report: BTCUSDT",
		"Rounds:",
		"Win Rate:",
		"Changed Decisions:",
		"#101",
		"WAIT -> ALLOW",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in report:\n%s", needle, out)
		}
	}
}

func TestWriteHTMLReport(t *testing.T) {
	result := &ReplayResult{
		Symbol: "ETHUSDT",
		Rounds: []ReplayRound{
			{
				SnapshotID:      201,
				State:           fsm.StateFlat,
				OriginalGate:    replayTestGate("ALLOW", "ALLOW", "long", 3, 0.7),
				ReplayedGate:    replayTestGate("WAIT", "QUALITY_TOO_LOW", "long", 0, 0.3),
				PriceAtDecision: 2500,
				PriceAfter:      2450,
				Changed:         true,
			},
		},
	}
	result.Metrics = computeMetrics(result.Rounds)

	var buf bytes.Buffer
	if err := WriteHTMLReport(&buf, result); err != nil {
		t.Fatalf("WriteHTMLReport: %v", err)
	}
	out := buf.String()
	for _, needle := range []string{
		"<html",
		"Gate Replay Report: ETHUSDT",
		"<table",
		"QUALITY_TOO_LOW",
		"201",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in html report:\n%s", needle, out)
		}
	}
}
