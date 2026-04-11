package llmapp

import (
	"context"
	"encoding/json"
	"testing"

	"brale-core/internal/decision"
	"brale-core/internal/decision/features"
)

func TestPickInputsUsesDecisionIntervalForIndicator(t *testing.T) {
	service := LLMAgentService{DecisionInterval: "15m"}
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": {
				"5m":  {Symbol: "BTCUSDT", Interval: "5m"},
				"15m": {Symbol: "BTCUSDT", Interval: "15m"},
				"1h":  {Symbol: "BTCUSDT", Interval: "1h"},
			},
		},
	}

	inputs, errs := service.pickInputs(context.Background(), data, "BTCUSDT", decision.AgentEnabled{Indicator: true})
	if len(errs) != 0 {
		t.Fatalf("unexpected stage errors: %v", errs)
	}
	if inputs.indicator.Interval != "15m" {
		t.Fatalf("interval=%q want %q", inputs.indicator.Interval, "15m")
	}
}

func TestStripTrendGlobalContextPreservesEMAAndVolRatio(t *testing.T) {
	obj := map[string]any{
		"global_context": map[string]any{
			"trend_slope":      1.2,
			"normalized_slope": 0.8,
			"window":           200,
			"slope_state":      "up",
			"vol_ratio":        1.3,
			"ema20":            101.2,
			"ema50":            99.8,
			"ema200":           95.4,
		},
	}

	stripTrendGlobalContext(obj)

	globalContext, ok := obj["global_context"].(map[string]any)
	if !ok {
		t.Fatalf("expected global_context to remain")
	}
	for _, key := range []string{"trend_slope", "normalized_slope", "window"} {
		if _, exists := globalContext[key]; exists {
			t.Fatalf("field %q should be removed", key)
		}
	}
	for _, key := range []string{"slope_state", "vol_ratio", "ema20", "ema50", "ema200"} {
		if _, exists := globalContext[key]; !exists {
			t.Fatalf("field %q should remain", key)
		}
	}
}

func TestStripTrendGlobalContextRemovesEmptyMap(t *testing.T) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(`{
		"global_context": {
			"trend_slope": 1.2,
			"normalized_slope": 0.8,
			"window": 200
		}
	}`), &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	stripTrendGlobalContext(obj)

	if _, exists := obj["global_context"]; exists {
		t.Fatalf("expected empty global_context to be removed")
	}
}
