package config

import "testing"

func TestValidateSymbolConfig_DoesNotRequireMACDFields(t *testing.T) {
	indicatorEnabled := true
	structureEnabled := false
	mechanicsEnabled := false
	temp := 0.1

	cfg := SymbolConfig{
		Symbol:     "BTCUSDT",
		Intervals:  []string{"1h"},
		KlineLimit: 250,
		Agent: AgentConfig{
			Indicator: &indicatorEnabled,
			Structure: &structureEnabled,
			Mechanics: &mechanicsEnabled,
		},
		Indicators: IndicatorConfig{
			EMAFast:   21,
			EMAMid:    50,
			EMASlow:   200,
			RSIPeriod: 14,
			ATRPeriod: 14,
			LastN:     5,
			SkipSTC:   true,
		},
		LLM: SymbolLLMConfig{
			Agent: LLMRoleSet{
				Indicator: LLMRoleConfig{Model: "test-model", Temperature: &temp},
			},
			Provider: LLMRoleSet{
				Indicator: LLMRoleConfig{Model: "test-model", Temperature: &temp},
			},
		},
	}

	if err := ValidateSymbolConfig(cfg); err != nil {
		t.Fatalf("ValidateSymbolConfig() error = %v", err)
	}
}
