package decision

import (
	"context"
	"strings"
	"testing"

	"brale-core/internal/decision/features"
	"brale-core/internal/decision/fund"
	"brale-core/internal/pkg/logging"
	"brale-core/internal/strategy"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestBuildTightenContextWarnsWhenStructureAnchorsUnavailable(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	ctx := logging.WithLogger(context.Background(), logger)

	p := &Pipeline{
		Bindings: map[string]strategy.StrategyBinding{
			"BTCUSDT": {Symbol: "BTCUSDT"},
		},
	}
	res := SymbolResult{
		Symbol: "BTCUSDT",
		Gate:   fund.GateDecision{Derived: map[string]any{"current_price": 100.0}},
	}
	comp := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": {
				"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{"close":100,"atr":2}`)},
			},
		},
		Trends: map[string]map[string]features.TrendJSON{
			"BTCUSDT": {
				"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{not-json}`)},
			},
		},
	}

	got, reason, err := p.buildTightenContext(ctx, res, comp, tightenExecution{}, 100)
	if err != nil {
		t.Fatalf("buildTightenContext err=%v", err)
	}
	if reason != "" {
		t.Fatalf("reason=%q want empty", reason)
	}
	if got.StructureAnchors != nil {
		t.Fatalf("structure_anchors=%v want nil fallback", got.StructureAnchors)
	}
	if observed.Len() != 1 {
		t.Fatalf("warn logs=%d want 1", observed.Len())
	}
	if !strings.Contains(observed.All()[0].Message, "structure anchors unavailable") {
		t.Fatalf("warn message=%q", observed.All()[0].Message)
	}
}
