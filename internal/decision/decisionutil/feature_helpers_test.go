package decisionutil

import (
	"testing"

	"brale-core/internal/decision/features"
)

func TestPickIndicatorJSONForIntervalPrefersDecisionInterval(t *testing.T) {
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": {
				"5m":  {Symbol: "BTCUSDT", Interval: "5m"},
				"15m": {Symbol: "BTCUSDT", Interval: "15m"},
				"1h":  {Symbol: "BTCUSDT", Interval: "1h"},
			},
		},
	}

	got, ok := PickIndicatorJSONForInterval(data, "BTCUSDT", "15m")
	if !ok {
		t.Fatalf("expected indicator selection to succeed")
	}
	if got.Interval != "15m" {
		t.Fatalf("interval=%q want %q", got.Interval, "15m")
	}
}

func TestPickIndicatorJSONForIntervalFallsBackToNearestAvailable(t *testing.T) {
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": {
				"5m": {Symbol: "BTCUSDT", Interval: "5m"},
				"1h": {Symbol: "BTCUSDT", Interval: "1h"},
			},
		},
	}

	got, ok := PickIndicatorJSONForInterval(data, "BTCUSDT", "15m")
	if !ok {
		t.Fatalf("expected indicator selection to succeed")
	}
	if got.Interval != "5m" {
		t.Fatalf("interval=%q want %q", got.Interval, "5m")
	}
}
