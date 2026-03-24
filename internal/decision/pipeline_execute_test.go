package decision

import (
	"testing"
	"time"

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

func TestValidatePlanRejectsIncompleteValidLLMPlan(t *testing.T) {
	err := validatePlan(execution.ExecutionPlan{
		Symbol:           "ETHUSDT",
		Valid:            true,
		PlanSource:       execution.PlanSourceLLM,
		PositionID:       "ETHUSDT-1",
		CreatedAt:        time.Now().UTC(),
		Entry:            2163.77,
		StopLoss:         0,
		TakeProfits:      nil,
		TakeProfitRatios: nil,
	})
	if err == nil {
		t.Fatal("expected validatePlan to reject incomplete valid llm plan")
	}
}

func TestValidatePlanAcceptsCompleteValidPlan(t *testing.T) {
	err := validatePlan(execution.ExecutionPlan{
		Symbol:           "ETHUSDT",
		Valid:            true,
		PlanSource:       execution.PlanSourceLLM,
		PositionID:       "ETHUSDT-1",
		CreatedAt:        time.Now().UTC(),
		Entry:            2163.77,
		StopLoss:         2120.0,
		TakeProfits:      []float64{2200, 2240},
		TakeProfitRatios: []float64{0.6, 0.4},
	})
	if err != nil {
		t.Fatalf("validatePlan returned error: %v", err)
	}
}
