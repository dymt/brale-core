package llmapp

import (
	"context"
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
