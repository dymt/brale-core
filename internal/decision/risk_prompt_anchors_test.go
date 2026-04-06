package decision

import (
	"context"
	"encoding/json"
	"testing"

	"brale-core/internal/config"
	"brale-core/internal/decision/features"
	"brale-core/internal/decision/fund"
	"brale-core/internal/risk"
	"brale-core/internal/store"
	"brale-core/internal/strategy"
)

func TestBuildStructureAnchorSummaryExtractsExpectedAnchors(t *testing.T) {
	comp := features.CompressionResult{
		Trends: map[string]map[string]features.TrendJSON{
			"BTCUSDT": {
				"1h": mustTrendJSONForAnchorTests(t, "BTCUSDT", "1h", features.TrendCompressedInput{
					Meta: features.TrendCompressedMeta{Symbol: "BTCUSDT", Interval: "1h", Timestamp: "2026-04-06T00:00:00Z"},
					GlobalContext: features.TrendGlobalContext{
						EMA20: refFloat(99),
						EMA50: refFloat(97),
					},
					KeyLevels: &features.TrendKeyLevels{
						LastSwingHigh: &features.TrendKeyLevel{Price: 104, Idx: 90},
						LastSwingLow:  &features.TrendKeyLevel{Price: 96, Idx: 88},
					},
					StructureCandidates: []features.TrendStructureCandidate{
						{Price: 98.5, Type: "support", Source: "fractal_low", AgeCandles: 1},
						{Price: 103.2, Type: "resistance", Source: "fractal_high", AgeCandles: 2},
					},
					SMC: features.TrendSMC{
						OrderBlock: features.TrendOrderBlock{Type: "bullish", Lower: refFloat(95.5), Upper: refFloat(100.5)},
					},
					BreakSummary: &features.TrendBreakSummary{
						LatestEventType:       "break_up",
						LatestEventAge:        refInt(1),
						LatestEventLevelPrice: refFloat(104),
						LatestEventLevelIdx:   refInt(90),
						LatestEventBarIdx:     refInt(98),
					},
				}),
				"4h": mustTrendJSONForAnchorTests(t, "BTCUSDT", "4h", features.TrendCompressedInput{
					Meta: features.TrendCompressedMeta{Symbol: "BTCUSDT", Interval: "4h", Timestamp: "2026-04-05T20:00:00Z"},
					GlobalContext: features.TrendGlobalContext{
						EMA20:  refFloat(94),
						EMA50:  refFloat(92),
						EMA200: refFloat(88),
					},
					KeyLevels: &features.TrendKeyLevels{
						LastSwingHigh: &features.TrendKeyLevel{Price: 110, Idx: 70},
						LastSwingLow:  &features.TrendKeyLevel{Price: 90, Idx: 65},
					},
					StructureCandidates: []features.TrendStructureCandidate{
						{Price: 93, Type: "support", Source: "ema", AgeCandles: 0, Window: 50},
						{Price: 109, Type: "resistance", Source: "range_high", AgeCandles: 0, Window: 30},
					},
					BreakSummary: &features.TrendBreakSummary{
						LatestEventType:       "break_up",
						LatestEventAge:        refInt(0),
						LatestEventLevelPrice: refFloat(110),
						LatestEventLevelIdx:   refInt(70),
						LatestEventBarIdx:     refInt(99),
					},
				}),
			},
		},
	}

	got, err := buildStructureAnchorSummary(comp, "BTCUSDT", 100, 2)
	if err != nil {
		t.Fatalf("buildStructureAnchorSummary: %v", err)
	}
	below, ok := got["nearest_below_entry"].(map[string]any)
	if !ok {
		t.Fatalf("nearest_below_entry=%#v", got["nearest_below_entry"])
	}
	if below["price"] != 99.0 {
		t.Fatalf("nearest_below_entry.price=%v, want 99", below["price"])
	}
	above, ok := got["nearest_above_entry"].(map[string]any)
	if !ok {
		t.Fatalf("nearest_above_entry=%#v", got["nearest_above_entry"])
	}
	if above["price"] != 100.5 {
		t.Fatalf("nearest_above_entry.price=%v, want 100.5", above["price"])
	}
	latestBreak, ok := got["latest_break"].(map[string]any)
	if !ok {
		t.Fatalf("latest_break=%#v", got["latest_break"])
	}
	if latestBreak["interval"] != "4h" || latestBreak["type"] != "break_up" {
		t.Fatalf("latest_break=%v", latestBreak)
	}
	emaByInterval, ok := got["ema_by_interval"].(map[string]any)
	if !ok {
		t.Fatalf("ema_by_interval=%#v", got["ema_by_interval"])
	}
	if _, ok := emaByInterval["1h"]; !ok {
		t.Fatalf("ema_by_interval=%v", emaByInterval)
	}
}

func TestBuildTightenPlanPassesStructureAnchorsToLLM(t *testing.T) {
	called := false
	stop := 100.2
	p := &Pipeline{
		TightenRiskLLM: func(ctx context.Context, input TightenRiskUpdateInput) (*TightenRiskUpdatePatch, error) {
			called = true
			if _, ok := input.StructureAnchors["nearest_above_entry"]; !ok {
				t.Fatalf("structure_anchors=%v", input.StructureAnchors)
			}
			if input.DistanceToLiqPct != 0.1 {
				t.Fatalf("distance_to_liq_pct=%v, want 0.1", input.DistanceToLiqPct)
			}
			return &TightenRiskUpdatePatch{
				StopLoss:    &stop,
				TakeProfits: []float64{106.5, 109.5},
			}, nil
		},
	}

	plan := risk.RiskPlan{
		StopPrice: 95,
		TPLevels: []risk.TPLevel{
			{LevelID: "tp-1", Price: 110, QtyPct: 0.5},
			{LevelID: "tp-2", Price: 120, QtyPct: 0.5},
		},
	}
	pos := store.PositionRecord{Symbol: "BTCUSDT", Side: "long", AvgEntry: 100}
	updateCtx := tightenContext{
		Binding:          strategy.StrategyBinding{RiskManagement: config.RiskManagementConfig{RiskStrategy: config.RiskStrategyConfig{Mode: "llm"}, TightenATR: config.TightenATRConfig{TP1ATR: 0.5, TP2ATR: 1.0}}},
		Gate:             fund.GateDecision{Derived: map[string]any{"plan": map[string]any{"liquidation_price": 94.5}}},
		MarkPrice:        105,
		ATR:              2,
		StructureAnchors: map[string]any{"nearest_above_entry": map[string]any{"price": 106}},
	}

	_, _, _, err := p.buildTightenPlan(context.Background(), pos, plan, updateCtx, 99)
	if err != nil {
		t.Fatalf("buildTightenPlan: %v", err)
	}
	if !called {
		t.Fatalf("llm tighten callback should be called")
	}
}

func TestSelectNearestAnchorExcludesPriceEqualToEntry(t *testing.T) {
	order := map[string]int{"1h": 0}
	candidates := []structureAnchorCandidate{
		{Interval: "1h", Price: 100, Source: "equal", Type: "mid"},
	}

	if below, ok := selectNearestAnchor(candidates, 100, 2, true, order); ok {
		t.Fatalf("below=%v, want no selection for equal price", below)
	}
	if above, ok := selectNearestAnchor(candidates, 100, 2, false, order); ok {
		t.Fatalf("above=%v, want no selection for equal price", above)
	}
}

func mustTrendJSONForAnchorTests(t *testing.T, symbol, interval string, input features.TrendCompressedInput) features.TrendJSON {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal trend json: %v", err)
	}
	return features.TrendJSON{Symbol: symbol, Interval: interval, RawJSON: raw}
}

func refFloat(v float64) *float64 { return &v }

func refInt(v int) *int { return &v }
