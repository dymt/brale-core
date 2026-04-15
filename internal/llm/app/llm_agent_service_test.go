package llmapp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"brale-core/internal/decision"
	"brale-core/internal/decision/features"
	"brale-core/internal/memory"
	"brale-core/internal/decision/agent"
)

func TestPickInputsUsesDecisionIntervalForIndicator(t *testing.T) {
	service := LLMAgentService{DecisionInterval: "15m"}
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": llmAgentIndicatorInputsForTest(t),
		},
	}

	inputs, errs := service.pickInputs(context.Background(), data, "BTCUSDT", decision.AgentEnabled{Indicator: true})
	if len(errs) != 0 {
		t.Fatalf("unexpected stage errors: %v", errs)
	}
	if inputs.indicator.Interval != "multi" {
		t.Fatalf("interval=%q want %q", inputs.indicator.Interval, "multi")
	}

	var payload struct {
		DecisionInterval string `json:"decision_interval"`
		MultiTF          []struct {
			Interval string `json:"interval"`
		} `json:"multi_tf"`
	}
	if err := json.Unmarshal(inputs.indicator.RawJSON, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.DecisionInterval != "15m" {
		t.Fatalf("decision_interval=%q want %q", payload.DecisionInterval, "15m")
	}
	if len(payload.MultiTF) == 0 || payload.MultiTF[0].Interval != "5m" {
		t.Fatalf("multi_tf=%v", payload.MultiTF)
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

func TestLLMAgentServiceAnalyzeBypassesCacheWhenWorkingMemoryContextChanges(t *testing.T) {
	indicatorStub := &stubRiskSessionProvider{callResp: `{"expansion":"expanding","alignment":"aligned","noise":"low"}`}
	service := LLMAgentService{
		Runner: &agent.Runner{Indicator: indicatorStub},
		Prompts: LLMPromptBuilder{
			AgentIndicatorSystem: "indicator-system",
			UserFormat:           UserPromptFormatBullet,
		},
		Cache:            NewLLMStageCache(),
		DecisionInterval: "15m",
	}
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": llmAgentIndicatorInputsForTest(t),
		},
	}

	if _, _, _, _, _, err := service.Analyze(context.Background(), "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	ctx := memory.WithPromptContext(context.Background(), "- [1h前] dir=long")
	if _, _, _, _, _, err := service.Analyze(ctx, "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() with memory error = %v", err)
	}

	if indicatorStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when memory context changes", indicatorStub.callCount)
	}
	if !strings.Contains(indicatorStub.lastUser, "近期决策记忆（仅供参考，禁止直接复用结论）") {
		t.Fatalf("user prompt missing memory block: %s", indicatorStub.lastUser)
	}
	if !strings.Contains(indicatorStub.lastUser, "dir=long") {
		t.Fatalf("user prompt missing memory payload: %s", indicatorStub.lastUser)
	}
}

func TestLLMAgentServiceAnalyzeBypassesCacheWhenEpisodicContextChanges(t *testing.T) {
	indicatorStub := &stubRiskSessionProvider{callResp: `{"expansion":"expanding","alignment":"aligned","noise":"low"}`}
	service := LLMAgentService{
		Runner: &agent.Runner{Indicator: indicatorStub},
		Prompts: LLMPromptBuilder{
			AgentIndicatorSystem: "indicator-system",
			UserFormat:           UserPromptFormatBullet,
		},
		Cache:            NewLLMStageCache(),
		DecisionInterval: "15m",
	}
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": llmAgentIndicatorInputsForTest(t),
		},
	}

	if _, _, _, _, _, err := service.Analyze(context.Background(), "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	ctx := memory.WithEpisodicContext(context.Background(), "[1] 上次追高后回撤")
	if _, _, _, _, _, err := service.Analyze(ctx, "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() with episodic memory error = %v", err)
	}

	if indicatorStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when episodic context changes", indicatorStub.callCount)
	}
	if !strings.Contains(indicatorStub.lastUser, "历史交易经验（仅供参考，禁止直接复用结论）") {
		t.Fatalf("user prompt missing episodic block: %s", indicatorStub.lastUser)
	}
}

func TestLLMAgentServiceAnalyzeAppendsSemanticRulesToSystem(t *testing.T) {
	indicatorStub := &stubRiskSessionProvider{callResp: `{"expansion":"expanding","alignment":"aligned","noise":"low"}`}
	service := LLMAgentService{
		Runner: &agent.Runner{Indicator: indicatorStub},
		Prompts: LLMPromptBuilder{
			AgentIndicatorSystem: "indicator-system",
			UserFormat:           UserPromptFormatBullet,
		},
		Cache:            NewLLMStageCache(),
		DecisionInterval: "15m",
	}
	data := features.CompressionResult{
		Indicators: map[string]map[string]features.IndicatorJSON{
			"BTCUSDT": llmAgentIndicatorInputsForTest(t),
		},
	}

	if _, _, _, _, _, err := service.Analyze(context.Background(), "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	ctx := memory.WithSemanticContext(context.Background(), "交易规则与经验:\n[1] 不追突破后的第三根阳线")
	if _, _, _, _, _, err := service.Analyze(ctx, "BTCUSDT", data, decision.AgentEnabled{Indicator: true}); err != nil {
		t.Fatalf("Analyze() with semantic rules error = %v", err)
	}

	if indicatorStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when semantic rules change", indicatorStub.callCount)
	}
	if !strings.Contains(indicatorStub.lastSystem, "不追突破后的第三根阳线") {
		t.Fatalf("system prompt missing semantic rules: %s", indicatorStub.lastSystem)
	}
}

func llmAgentIndicatorInputsForTest(t *testing.T) map[string]features.IndicatorJSON {
	t.Helper()

	build := func(interval string, current, previous float64, age int64, fast float64, fastLastN []float64, mid float64, midLastN []float64, slow float64, slowLastN []float64, rsi float64, rsiNorm float64, atr float64, atrChange float64, obvChange float64, stcState string) features.IndicatorJSON {
		raw, err := json.Marshal(map[string]any{
			"_meta": map[string]any{
				"series_order": "oldest_to_latest",
				"sampled_at":   "2026-04-11T00:00:00Z",
				"version":      "indicator_compress_v1",
				"data_age_sec": map[string]int64{"indicator": age},
			},
			"market": map[string]any{
				"symbol":         "BTCUSDT",
				"interval":       interval,
				"current_price":  current,
				"previous_price": previous,
				"price_timestamp": "2026-04-11T00:00:00Z",
			},
			"data": map[string]any{
				"ema_fast": map[string]any{"latest": fast, "last_n": fastLastN},
				"ema_mid":  map[string]any{"latest": mid, "last_n": midLastN},
				"ema_slow": map[string]any{"latest": slow, "last_n": slowLastN},
				"rsi":      map[string]any{"current": rsi, "normalized_slope": rsiNorm},
				"atr":      map[string]any{"latest": atr, "change_pct": atrChange},
				"obv":      map[string]any{"change_rate": obvChange},
				"stc":      map[string]any{"state": stcState},
			},
		})
		if err != nil {
			t.Fatalf("marshal indicator payload: %v", err)
		}
		return features.IndicatorJSON{Symbol: "BTCUSDT", Interval: interval, RawJSON: raw}
	}

	return map[string]features.IndicatorJSON{
		"5m":  build("5m", 105, 97, 12, 102, []float64{98, 102}, 101, []float64{101, 101}, 100, []float64{101, 100}, 62, 0.2, 4, 6, 0.03, "rising"),
		"15m": build("15m", 104, 102, 20, 101, []float64{100, 101}, 100, []float64{99, 100}, 99, []float64{98, 99}, 58, 0.18, 5, 4, 0.02, "rising"),
		"1h":  build("1h", 96, 98, 30, 98, []float64{99, 98}, 99, []float64{100, 99}, 100, []float64{101, 100}, 42, -0.2, 6, -3, -0.03, "falling"),
	}
}
