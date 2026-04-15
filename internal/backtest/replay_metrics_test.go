package backtest

import (
	"math"
	"testing"

	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/fund"
)

func TestComputeMetrics(t *testing.T) {
	rounds := []ReplayRound{
		{
			State:           fsm.StateFlat,
			ReplayedGate:    replayTestGate("ALLOW", "ALLOW", "long", 3, 0.8),
			PriceAtDecision: 100,
			PriceAfter:      120,
		},
		{
			State:           fsm.StateFlat,
			Changed:         true,
			ReplayedGate:    replayTestGate("ALLOW", "ALLOW", "short", 2, -0.7),
			PriceAtDecision: 100,
			PriceAfter:      110,
		},
		{
			State:           fsm.StateFlat,
			ReplayedGate:    replayTestGate("WAIT", "QUALITY_TOO_LOW", "long", 0, 0.6),
			PriceAtDecision: 100,
			PriceAfter:      90,
		},
		{
			State:           fsm.StateFlat,
			Changed:         true,
			ReplayedGate:    replayTestGate("VETO", "STRUCT_HARD_INVALIDATION", "short", 0, -0.8),
			PriceAtDecision: 100,
			PriceAfter:      90,
		},
		{
			State:      fsm.StateInPosition,
			Skipped:    true,
			SkipReason: "in_position_round",
		},
	}

	metrics := computeMetrics(rounds)
	if metrics.TotalRounds != 5 {
		t.Fatalf("TotalRounds=%d want=5", metrics.TotalRounds)
	}
	if metrics.ReplayableRounds != 4 {
		t.Fatalf("ReplayableRounds=%d want=4", metrics.ReplayableRounds)
	}
	if metrics.SkippedCount != 1 {
		t.Fatalf("SkippedCount=%d want=1", metrics.SkippedCount)
	}
	if metrics.AllowCount != 2 || metrics.WaitCount != 1 || metrics.VetoCount != 1 {
		t.Fatalf("unexpected action counts: %+v", metrics)
	}
	if metrics.TruePositive != 1 || metrics.FalsePositive != 1 || metrics.TrueNegative != 1 || metrics.FalseNegative != 1 {
		t.Fatalf("unexpected confusion matrix: %+v", metrics)
	}
	if math.Abs(metrics.Precision-0.5) > 1e-9 {
		t.Fatalf("Precision=%v want=0.5", metrics.Precision)
	}
	if math.Abs(metrics.WinRate-0.5) > 1e-9 {
		t.Fatalf("WinRate=%v want=0.5", metrics.WinRate)
	}
	if math.Abs(metrics.ProfitFactor-2.0) > 1e-9 {
		t.Fatalf("ProfitFactor=%v want=2.0", metrics.ProfitFactor)
	}
	if math.Abs(metrics.MaxDrawdown-0.10) > 1e-9 {
		t.Fatalf("MaxDrawdown=%v want=0.10", metrics.MaxDrawdown)
	}
	if math.Abs(metrics.CalmarRatio-0.8) > 1e-9 {
		t.Fatalf("CalmarRatio=%v want=0.8", metrics.CalmarRatio)
	}
	if math.Abs(metrics.SharpeRatio-0.2294157339) > 1e-9 {
		t.Fatalf("SharpeRatio=%v want≈0.2294", metrics.SharpeRatio)
	}
	if metrics.ChangedCount != 2 {
		t.Fatalf("ChangedCount=%d want=2", metrics.ChangedCount)
	}
	if math.Abs(metrics.ChangeRate-0.4) > 1e-9 {
		t.Fatalf("ChangeRate=%v want=0.4", metrics.ChangeRate)
	}
}

func replayTestGate(action, reason, direction string, grade int, score float64) fund.GateDecision {
	return fund.GateDecision{
		GlobalTradeable: action == "ALLOW",
		DecisionAction:  action,
		GateReason:      reason,
		Direction:       direction,
		Grade:           grade,
		Derived: map[string]any{
			"direction_consensus": map[string]any{
				"score": score,
			},
		},
	}
}
