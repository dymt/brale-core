package ruleflow

import (
	"context"
	"testing"

	"brale-core/internal/config"
	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/features"
	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/fund"
	"brale-core/internal/decision/provider"
	"brale-core/internal/execution"
	"brale-core/internal/strategy"
)

func TestHardGuardStopLossExitsByDefault(t *testing.T) {
	result := evaluateInPositionRuleflow(t, inPositionRuleflowOptions{})
	if result.Gate.DecisionAction != "EXIT" {
		t.Fatalf("gate action=%s want EXIT", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "STOP_LOSS" {
		t.Fatalf("gate reason=%s want STOP_LOSS", result.Gate.GateReason)
	}
}

func TestHardGuardCanBeDisabledByStrategyConfig(t *testing.T) {
	disabled := false
	result := evaluateInPositionRuleflow(t, inPositionRuleflowOptions{
		hardGuard: config.HardGuardToggleConfig{
			Enabled: &disabled,
		},
	})
	if result.Gate.DecisionAction == "EXIT" {
		t.Fatalf("gate action=%s want non-EXIT", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "KEEP" {
		t.Fatalf("gate reason=%s want KEEP", result.Gate.GateReason)
	}
}

func TestHardGuardCanDisableIndividualRule(t *testing.T) {
	disabled := false
	result := evaluateInPositionRuleflow(t, inPositionRuleflowOptions{
		hardGuard: config.HardGuardToggleConfig{
			Enabled:  boolPtr(true),
			StopLoss: &disabled,
		},
	})
	if result.Gate.DecisionAction == "EXIT" {
		t.Fatalf("gate action=%s want non-EXIT", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "KEEP" {
		t.Fatalf("gate reason=%s want KEEP", result.Gate.GateReason)
	}
}

type inPositionRuleflowOptions struct {
	hardGuard config.HardGuardToggleConfig
}

func evaluateInPositionRuleflow(t *testing.T, opts inPositionRuleflowOptions) Result {
	t.Helper()
	engine := NewEngine()
	result, err := engine.Evaluate(context.Background(), defaultRuleChainPath(t), Input{
		Symbol: "BTCUSDT",
		Providers: fund.ProviderBundle{
			Indicator: provider.IndicatorProviderOut{SignalTag: "trend_surge"},
			Structure: provider.StructureProviderOut{ClearStructure: true, Integrity: true, SignalTag: "breakout_confirmed"},
			Mechanics: provider.MechanicsProviderOut{
				LiquidationStress: provider.SemanticSignal{Value: false, Confidence: provider.ConfidenceLow, Reason: "ok"},
				SignalTag:         "neutral",
			},
			Enabled: fund.ProviderEnabled{Indicator: true, Structure: true, Mechanics: true},
		},
		AgentIndicator:      agent.IndicatorSummary{MovementConfidence: 0.9},
		AgentStructure:      agent.StructureSummary{MovementConfidence: 0.9},
		AgentMechanics:      agent.MechanicsSummary{MovementConfidence: 0.9},
		InPosition:          InPositionOutputs{Indicator: provider.InPositionIndicatorOut{MonitorTag: "keep", Reason: "ok"}, Structure: provider.InPositionStructureOut{MonitorTag: "keep", Reason: "ok"}, Mechanics: provider.InPositionMechanicsOut{MonitorTag: "keep", Reason: "ok"}, Ready: true},
		Position:            HardGuardPosition{Side: "long", MarkPrice: 95, MarkPriceOK: true, StopLoss: 100, StopLossOK: true},
		State:               fsm.StateInPosition,
		StructureDirection:  "long",
		ConsensusScore:      0.7,
		ConsensusConfidence: 0.8,
		ConsensusAgreement:  1.0,
		ConsensusCoverage:   1.0,
		ConsensusResonance:  0.0,
		ConsensusResonant:   false,
		ScoreThreshold:      0.35,
		ConfidenceThreshold: 0.52,
		Account:             execution.AccountState{Equity: 10000, Available: 10000},
		Risk:                execution.RiskParams{RiskPerTradePct: 0.01},
		Binding: strategy.StrategyBinding{
			Symbol: "BTCUSDT",
			RiskManagement: config.RiskManagementConfig{
				RiskPerTradePct: 0.01,
				MaxInvestPct:    1.0,
				MaxLeverage:     20,
				Grade1Factor:    1.0,
				Grade2Factor:    1.0,
				Grade3Factor:    1.0,
				EntryMode:       "market",
				RiskStrategy:    config.RiskStrategyConfig{Mode: "native"},
				InitialExit: config.InitialExitConfig{
					Policy:            "atr_structure_v1",
					StructureInterval: "1h",
					Params:            map[string]any{},
				},
				Gate: config.GateConfig{
					QualityThreshold: 0.35,
					EdgeThreshold:    0.10,
				},
				HardGuard: opts.hardGuard,
			},
		},
		Compression: features.CompressionResult{
			Indicators: map[string]map[string]features.IndicatorJSON{
				"BTCUSDT": {
					"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{"close":95,"rsi":50,"pct_change_5m":-0.1}`)},
				},
			},
			Trends: map[string]map[string]features.TrendJSON{
				"BTCUSDT": {
					"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{"recent_candles":[{"h":101,"l":99},{"h":103,"l":98}]}`)},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("evaluate ruleflow: %v", err)
	}
	return result
}
