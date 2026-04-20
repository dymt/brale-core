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

func TestGateMechanicsCascadeVeto(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceHigh, Reason: "cascade"},
			SignalTag:         "liquidation_cascade",
		},
	})
	if result.Gate.DecisionAction != "VETO" {
		t.Fatalf("gate action=%s want VETO", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "LIQUIDATION_CASCADE" {
		t.Fatalf("gate reason=%s want LIQUIDATION_CASCADE", result.Gate.GateReason)
	}
}

func TestGateMechanicsCascadeCanBeDisabledByStrategyConfig(t *testing.T) {
	disabled := false
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceHigh, Reason: "cascade"},
			SignalTag:         "liquidation_cascade",
		},
		riskManagement: &config.RiskManagementConfig{
			RiskPerTradePct: 0.01,
			MaxInvestPct:    1.0,
			MaxLeverage:     20,
			Grade1Factor:    1.0,
			Grade2Factor:    1.0,
			Grade3Factor:    1.0,
			EntryOffsetATR:  0,
			EntryMode:       "market",
			RiskStrategy:    config.RiskStrategyConfig{Mode: "native"},
			Gate: config.GateConfig{
				QualityThreshold: 0.35,
				EdgeThreshold:    0.10,
				HardStop: config.GateHardStopConfig{
					LiquidationCascade: &disabled,
				},
			},
			InitialExit: config.InitialExitConfig{
				Policy:            "atr_structure_v1",
				StructureInterval: "1h",
				Params:            map[string]any{},
			},
		},
	})
	if result.Gate.DecisionAction == "VETO" {
		t.Fatalf("gate action=%s want non-VETO", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "EDGE_TOO_LOW" {
		t.Fatalf("gate reason=%s want EDGE_TOO_LOW", result.Gate.GateReason)
	}
}

func TestGateMechanicsHighConfidenceStressCanStillAllow(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceHigh, Reason: "stress"},
			SignalTag:         "crowded_long",
		},
	})
	if result.Gate.DecisionAction != "ALLOW" {
		t.Fatalf("gate action=%s want ALLOW", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "ALLOW" {
		t.Fatalf("gate reason=%s want ALLOW", result.Gate.GateReason)
	}
}

func TestGateMechanicsMissingConfidenceCanStillAllow(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Reason: "stress"},
			SignalTag:         "crowded_long",
		},
	})
	if result.Gate.DecisionAction != "ALLOW" {
		t.Fatalf("gate action=%s want ALLOW", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "ALLOW" {
		t.Fatalf("gate reason=%s want ALLOW", result.Gate.GateReason)
	}
	if result.Plan == nil || !result.Plan.Valid {
		t.Fatalf("expected valid plan")
	}
}

func TestGateMechanicsLowConfidenceStressUsesSieve(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceLow, Reason: "stress"},
			SignalTag:         "crowded_long",
		},
		sieve: config.RiskManagementSieveConfig{
			MinSizeFactor:     0.1,
			DefaultGateAction: "ALLOW",
			DefaultSizeFactor: 1.0,
			Rows: []config.RiskManagementSieveRow{{
				MechanicsTag:  "crowded_long",
				LiqConfidence: "low",
				CrowdingAlign: boolPtr(true),
				GateAction:    "ALLOW",
				SizeFactor:    0.5,
				ReasonCode:    "CROWD_ALIGN_LOW",
			}},
		},
	})
	if result.Gate.DecisionAction != "ALLOW" {
		t.Fatalf("gate action=%s want ALLOW", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "ALLOW" {
		t.Fatalf("gate reason=%s want ALLOW", result.Gate.GateReason)
	}
	if got := result.Gate.Derived["gate_action_before_sieve"]; got != "ALLOW" {
		t.Fatalf("gate_action_before_sieve=%v want ALLOW", got)
	}
	if got := result.Gate.Derived["sieve_action"]; got != "ALLOW" {
		t.Fatalf("sieve_action=%v want ALLOW", got)
	}
	if got := result.Gate.Derived["sieve_reason"]; got != "CROWD_ALIGN_LOW" {
		t.Fatalf("sieve_reason=%v want CROWD_ALIGN_LOW", got)
	}
	if got := result.Gate.Derived["sieve_size_factor"]; got != 0.5 {
		t.Fatalf("sieve_size_factor=%v want 0.5", got)
	}
	if result.Plan == nil || !result.Plan.Valid {
		t.Fatalf("expected valid plan")
	}
	if result.Plan.RiskPct != 0.005 {
		t.Fatalf("risk_pct=%v want 0.005", result.Plan.RiskPct)
	}
}

func TestGateSieveWaitOverridesAllow(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceLow, Reason: "stress"},
			SignalTag:         "crowded_long",
		},
		sieve: config.RiskManagementSieveConfig{
			MinSizeFactor:     0.1,
			DefaultGateAction: "ALLOW",
			DefaultSizeFactor: 1.0,
			Rows: []config.RiskManagementSieveRow{{
				MechanicsTag:  "crowded_long",
				LiqConfidence: "low",
				CrowdingAlign: boolPtr(true),
				GateAction:    "WAIT",
				SizeFactor:    0.0,
				ReasonCode:    "CROWD_ALIGN_WAIT",
			}},
		},
	})
	if result.Gate.DecisionAction != "WAIT" {
		t.Fatalf("gate action=%s want WAIT", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "CROWD_ALIGN_WAIT" {
		t.Fatalf("gate reason=%s want CROWD_ALIGN_WAIT", result.Gate.GateReason)
	}
	if result.Gate.GlobalTradeable {
		t.Fatalf("global_tradeable=%v want false", result.Gate.GlobalTradeable)
	}
	if got := result.Gate.Derived["gate_action_before_sieve"]; got != "ALLOW" {
		t.Fatalf("gate_action_before_sieve=%v want ALLOW", got)
	}
	if got := result.Gate.Derived["sieve_action"]; got != "WAIT" {
		t.Fatalf("sieve_action=%v want WAIT", got)
	}
	if got := result.Gate.Derived["sieve_reason"]; got != "CROWD_ALIGN_WAIT" {
		t.Fatalf("sieve_reason=%v want CROWD_ALIGN_WAIT", got)
	}
	if result.Plan != nil {
		t.Fatalf("expected no plan when sieve waits")
	}
}

func TestDeriveTradeableIgnoresNonCascadeStress(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		mechanics: provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: true, Confidence: provider.ConfidenceHigh, Reason: "stress"},
			SignalTag:         "crowded_long",
		},
	})
	mechanics, ok := result.Gate.Derived["mechanics"].(map[string]any)
	if !ok {
		t.Fatalf("missing mechanics derived block")
	}
	if tradeable, ok := mechanics["tradeable"].(bool); !ok || !tradeable {
		t.Fatalf("mechanics.tradeable=%v want true", mechanics["tradeable"])
	}
}

func TestGateStructureHardStopCanBeDisabledByStrategyConfig(t *testing.T) {
	disabled := false
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		structure: provider.StructureProviderOut{
			ClearStructure: false,
			Integrity:      false,
			Reason:         "broken",
			SignalTag:      "structure_broken",
		},
		riskManagement: &config.RiskManagementConfig{
			RiskPerTradePct: 0.01,
			MaxInvestPct:    1.0,
			MaxLeverage:     20,
			Grade1Factor:    1.0,
			Grade2Factor:    1.0,
			Grade3Factor:    1.0,
			EntryOffsetATR:  0,
			EntryMode:       "market",
			RiskStrategy:    config.RiskStrategyConfig{Mode: "native"},
			Gate: config.GateConfig{
				QualityThreshold: 0.35,
				EdgeThreshold:    0.10,
				HardStop: config.GateHardStopConfig{
					StructureInvalidation: &disabled,
				},
			},
			InitialExit: config.InitialExitConfig{
				Policy:            "atr_structure_v1",
				StructureInterval: "1h",
				Params:            map[string]any{},
			},
		},
	})
	if result.Gate.DecisionAction == "VETO" {
		t.Fatalf("gate action=%s want non-VETO", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "QUALITY_TOO_LOW" {
		t.Fatalf("gate reason=%s want QUALITY_TOO_LOW", result.Gate.GateReason)
	}
}

func TestGateIndicatorNoiseAndTagConsistencyFlowIntoQuality(t *testing.T) {
	tests := []struct {
		name       string
		indicator  string
		meanNoise  bool
		alignment  bool
		momentum   bool
		wantAction string
		wantReason string
	}{
		{
			name:       "noise can still allow when quality remains above threshold",
			indicator:  "noise",
			meanNoise:  true,
			alignment:  true,
			momentum:   true,
			wantAction: "ALLOW",
			wantReason: "ALLOW",
		},
		{
			name:       "tag inconsistency now degrades quality to wait",
			indicator:  "trend_surge",
			meanNoise:  false,
			alignment:  false,
			momentum:   false,
			wantAction: "WAIT",
			wantReason: "QUALITY_TOO_LOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := evaluateGateDecision(gateInputs{
				State:              "FLAT",
				StructureDirection: "long",
				IndicatorTag:       tt.indicator,
				StructureTag:       "unknown_structure",
				MechanicsTag:       "neutral",
				MomentumExpansion:  tt.momentum,
				Alignment:          tt.alignment,
				MeanRevNoise:       tt.meanNoise,
				StructureClear:     true,
				StructureIntegrity: true,
				LiquidationStress:  false,
				LiqConfidence:      "low",
				ConsensusScore:     0.7,
				QualityThreshold:   0.35,
				EdgeThreshold:      0.10,
			}, false)

			if decision.Action != tt.wantAction {
				t.Fatalf("action=%s want %s", decision.Action, tt.wantAction)
			}
			if decision.Reason != tt.wantReason {
				t.Fatalf("reason=%s want %s", decision.Reason, tt.wantReason)
			}
		})
	}
}

func TestGateLowAgentConfidenceCanReduceQualityViaRuleflow(t *testing.T) {
	result := evaluateFlatRuleflow(t, flatRuleflowOptions{
		indicator: provider.IndicatorProviderOut{
			MomentumExpansion: false,
			Alignment:         true,
			MeanRevNoise:      false,
			SignalTag:         "pullback_entry",
		},
		structure: provider.StructureProviderOut{
			ClearStructure: true,
			Integrity:      true,
			Reason:         "ok",
			SignalTag:      "support_retest",
		},
		indicatorConf: 0.1,
		structureConf: 0.1,
	})
	if result.Gate.DecisionAction != "WAIT" {
		t.Fatalf("gate action=%s want WAIT", result.Gate.DecisionAction)
	}
	if result.Gate.GateReason != "QUALITY_TOO_LOW" {
		t.Fatalf("gate reason=%s want QUALITY_TOO_LOW", result.Gate.GateReason)
	}
}

func evaluateFlatRuleflow(t *testing.T, opts flatRuleflowOptions) Result {
	t.Helper()
	engine := NewEngine()
	indicator := opts.indicator
	if indicator.SignalTag == "" {
		indicator = provider.IndicatorProviderOut{
			MomentumExpansion: true,
			Alignment:         true,
			MeanRevNoise:      false,
			SignalTag:         "trend_surge",
		}
	}
	structure := opts.structure
	if structure.SignalTag == "" {
		structure = provider.StructureProviderOut{
			ClearStructure: true,
			Integrity:      true,
			Reason:         "ok",
			SignalTag:      "breakout_confirmed",
		}
	}
	mechanics := opts.mechanics
	if mechanics.SignalTag == "" {
		mechanics = provider.MechanicsProviderOut{
			LiquidationStress: provider.SemanticSignal{Value: false, Confidence: provider.ConfidenceLow, Reason: "ok"},
			SignalTag:         "neutral",
		}
	}
	sieve := opts.sieve
	riskManagement := config.RiskManagementConfig{
		RiskPerTradePct: 0.01,
		MaxInvestPct:    1.0,
		MaxLeverage:     20,
		Grade1Factor:    1.0,
		Grade2Factor:    1.0,
		Grade3Factor:    1.0,
		EntryOffsetATR:  0,
		EntryMode:       "market",
		RiskStrategy:    config.RiskStrategyConfig{Mode: "native"},
		Gate: config.GateConfig{
			QualityThreshold: 0.35,
			EdgeThreshold:    0.10,
		},
		InitialExit: config.InitialExitConfig{
			Policy:            "atr_structure_v1",
			StructureInterval: "1h",
			Params:            map[string]any{},
		},
		Sieve: sieve,
	}
	if opts.riskManagement != nil {
		riskManagement = *opts.riskManagement
		if len(riskManagement.Sieve.Rows) == 0 &&
			riskManagement.Sieve.DefaultGateAction == "" &&
			riskManagement.Sieve.DefaultSizeFactor == 0 &&
			riskManagement.Sieve.MinSizeFactor == 0 {
			riskManagement.Sieve = sieve
		}
	}
	result, err := engine.Evaluate(context.Background(), defaultRuleChainPath(t), Input{
		Symbol: "BTCUSDT",
		Providers: fund.ProviderBundle{
			Indicator: indicator,
			Structure: structure,
			Mechanics: mechanics,
			Enabled:   fund.ProviderEnabled{Indicator: true, Structure: true, Mechanics: true},
		},
		AgentIndicator: agent.IndicatorSummary{
			MovementConfidence: resolveTestConfidence(opts.indicatorConf),
		},
		AgentStructure: agent.StructureSummary{
			MovementConfidence: resolveTestConfidence(opts.structureConf),
		},
		StructureDirection:  "long",
		ConsensusScore:      resolveTestConsensusScore(opts.consensusScore),
		ConsensusConfidence: 0.8,
		ScoreThreshold:      0.35,
		ConfidenceThreshold: 0.52,
		State:               fsm.StateFlat,
		BuildPlan:           true,
		Account:             execution.AccountState{Equity: 10000, Available: 10000},
		Risk:                execution.RiskParams{RiskPerTradePct: 0.01},
		Binding: strategy.StrategyBinding{
			Symbol:         "BTCUSDT",
			RiskManagement: riskManagement,
		},
		Compression: features.CompressionResult{
			Indicators: map[string]map[string]features.IndicatorJSON{
				"BTCUSDT": {
					"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{"close":100,"atr":2}`)},
				},
			},
			Trends: map[string]map[string]features.TrendJSON{
				"BTCUSDT": {
					"1h": {Symbol: "BTCUSDT", Interval: "1h", RawJSON: []byte(`{"recent_candles":[{"h":101,"l":99},{"h":103,"l":98}],"structure_candidates":[{"price":98,"type":"support"}]}`)},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("evaluate ruleflow: %v", err)
	}
	return result
}

type flatRuleflowOptions struct {
	indicator      provider.IndicatorProviderOut
	structure      provider.StructureProviderOut
	mechanics      provider.MechanicsProviderOut
	sieve          config.RiskManagementSieveConfig
	riskManagement *config.RiskManagementConfig
	consensusScore float64
	indicatorConf  float64
	structureConf  float64
}

func resolveTestConsensusScore(v float64) float64 {
	if v == 0 {
		return 0.7
	}
	return v
}

func resolveTestConfidence(v float64) float64 {
	if v == 0 {
		return 0.9
	}
	return v
}

func TestSelectBestSieveRowIgnoresPriorityField(t *testing.T) {
	rows := []any{
		map[string]any{
			"mechanics_tag":  "crowded_long",
			"liq_confidence": "low",
			"gate_action":    "ALLOW",
			"size_factor":    0.5,
			"reason_code":    "FIRST_MATCH",
			"priority":       1,
		},
		map[string]any{
			"mechanics_tag":  "crowded_long",
			"liq_confidence": "low",
			"gate_action":    "WAIT",
			"size_factor":    0.0,
			"reason_code":    "SECOND_MATCH",
			"priority":       99,
		},
	}

	best := selectBestSieveRow(rows, sieveMatchInputs{MechanicsTag: "crowded_long", LiqConfidence: "low"})
	if got := toString(best["reason_code"]); got != "FIRST_MATCH" {
		t.Fatalf("reason_code=%q want FIRST_MATCH", got)
	}
}

func TestMatchSieveRowIgnoresUnsupportedFields(t *testing.T) {
	row := map[string]any{
		"mechanics_tag":  "crowded_long",
		"liq_confidence": "low",
		"gate_action":    "ALLOW",
		"size_factor":    0.5,
		"reason_code":    "SUPPORTED_ONLY",
		"indicator_tag":  "mismatch_indicator",
		"structure_tag":  "mismatch_structure",
		"direction":      "short",
	}

	matchCount, ok := matchSieveRow(row, sieveMatchInputs{
		MechanicsTag:  "crowded_long",
		LiqConfidence: "low",
		CrowdingAlign: true,
	})
	if !ok {
		t.Fatalf("expected row to match on supported fields only")
	}
	if matchCount != 2 {
		t.Fatalf("matchCount=%d want 2", matchCount)
	}
}
