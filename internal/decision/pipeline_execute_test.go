package decision

import (
	"testing"

	"brale-core/internal/config"
	"brale-core/internal/decision/decisionmode"
	"brale-core/internal/execution"
	"brale-core/internal/strategy"
)

func TestEnrichRunOptionsInjectsRiskModeFromBinding(t *testing.T) {
	p := &Pipeline{
		Bindings: map[string]strategy.StrategyBinding{
			"ETHUSDT": {RiskManagement: config.RiskManagementConfig{RiskStrategy: config.RiskStrategyConfig{Mode: "llm"}}},
			"BTCUSDT": {RiskManagement: config.RiskManagementConfig{RiskStrategy: config.RiskStrategyConfig{Mode: "native"}}},
		},
	}

	runOpts := p.enrichRunOptions(
		RunOptions{BuildPlan: true},
		[]string{"ETHUSDT", "BTCUSDT"},
		map[string]decisionmode.Mode{"ETHUSDT": decisionmode.ModeFlat, "BTCUSDT": decisionmode.ModeFlat},
	)

	if got := runOpts.RiskStrategyModeBySymbol["ETHUSDT"]; got != execution.PlanSourceLLM {
		t.Fatalf("ETHUSDT risk mode=%q want=%q", got, execution.PlanSourceLLM)
	}
	if got := runOpts.RiskStrategyModeBySymbol["BTCUSDT"]; got != execution.PlanSourceGo {
		t.Fatalf("BTCUSDT risk mode=%q want=%q", got, execution.PlanSourceGo)
	}
}

func TestEnrichRunOptionsKeepsExplicitRiskMode(t *testing.T) {
	p := &Pipeline{
		Bindings: map[string]strategy.StrategyBinding{
			"ETHUSDT": {RiskManagement: config.RiskManagementConfig{RiskStrategy: config.RiskStrategyConfig{Mode: "native"}}},
		},
	}

	runOpts := p.enrichRunOptions(
		RunOptions{
			BuildPlan: true,
			RiskStrategyModeBySymbol: map[string]string{
				"ETHUSDT": execution.PlanSourceLLM,
			},
		},
		[]string{"ETHUSDT"},
		map[string]decisionmode.Mode{"ETHUSDT": decisionmode.ModeFlat},
	)

	if got := runOpts.RiskStrategyModeBySymbol["ETHUSDT"]; got != execution.PlanSourceLLM {
		t.Fatalf("ETHUSDT risk mode override=%q want=%q", got, execution.PlanSourceLLM)
	}
}
