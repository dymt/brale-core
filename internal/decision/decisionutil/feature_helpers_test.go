package decisionutil

import (
	"testing"

	"brale-core/internal/decision/features"
)

func TestPickTrendJSONPrefersLargestInterval(t *testing.T) {
	data := features.CompressionResult{
		Trends: map[string]map[string]features.TrendJSON{
			"ETHUSDT": {
				"15m": {Symbol: "ETHUSDT", Interval: "15m", RawJSON: []byte(`{"k":1}`)},
				"1h":  {Symbol: "ETHUSDT", Interval: "1h", RawJSON: []byte(`{"k":2}`)},
				"4h":  {Symbol: "ETHUSDT", Interval: "4h", RawJSON: []byte(`{"k":3}`)},
			},
		},
	}

	got, ok := PickTrendJSON(data, "ETHUSDT")
	if !ok {
		t.Fatalf("expected trend to be found")
	}
	if got.Interval != "4h" {
		t.Fatalf("interval=%q, want %q", got.Interval, "4h")
	}
}
