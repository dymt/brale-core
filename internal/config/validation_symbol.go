package config

import "strings"

func ValidateSymbolIndexConfig(cfg SymbolIndexConfig) error {
	if len(cfg.Symbols) == 0 {
		return validationErrorf("symbols is required")
	}
	seen := make(map[string]struct{}, len(cfg.Symbols))
	for i, item := range cfg.Symbols {
		sym, err := validateCanonicalSymbol("symbols.symbol", item.Symbol)
		if err != nil {
			return validationErrorf("symbols[%d]: %v", i, err)
		}
		if _, ok := seen[sym]; ok {
			return validationErrorf("symbols contains duplicate symbol=%s", sym)
		}
		seen[sym] = struct{}{}
		if strings.TrimSpace(item.Config) == "" {
			return validationErrorf("symbols.%s config path is required", sym)
		}
		if strings.TrimSpace(item.Strategy) == "" {
			return validationErrorf("symbols.%s strategy path is required", sym)
		}
	}
	return nil
}

func ValidateSymbolConfig(cfg SymbolConfig) error {
	if _, err := validateCanonicalSymbol("symbol", cfg.Symbol); err != nil {
		return err
	}
	if len(cfg.Intervals) == 0 {
		return validationErrorf("intervals is required")
	}
	if cfg.KlineLimit <= 0 {
		return validationErrorf("kline_limit must be > 0")
	}
	enabled, err := ResolveAgentEnabled(cfg.Agent)
	if err != nil {
		return err
	}
	if err := validateIndicatorConfig(cfg.Indicators); err != nil {
		return err
	}
	if err := validateConsensusConfig(cfg.Consensus); err != nil {
		return err
	}
	if err := validateCooldownConfig(cfg.Cooldown); err != nil {
		return err
	}
	if err := validateLLMConfig(cfg.LLM, enabled); err != nil {
		return err
	}
	requiredLimit := requiredKlineLimit(cfg)
	if cfg.KlineLimit < requiredLimit {
		return validationErrorf("kline_limit must be >= %d", requiredLimit)
	}
	return nil
}

func validateIndicatorConfig(cfg IndicatorConfig) error {
	if cfg.EMAFast <= 0 || cfg.EMAMid <= 0 || cfg.EMASlow <= 0 {
		return validationErrorf("indicators.ema_fast/ema_mid/ema_slow must be > 0")
	}
	if cfg.RSIPeriod <= 0 {
		return validationErrorf("indicators.rsi_period must be > 0")
	}
	if cfg.ATRPeriod <= 0 {
		return validationErrorf("indicators.atr_period must be > 0")
	}
	if cfg.MACDFast <= 0 || cfg.MACDSlow <= 0 || cfg.MACDSignal <= 0 {
		return validationErrorf("indicators.macd_fast/macd_slow/macd_signal must be > 0")
	}
	if cfg.LastN <= 0 {
		return validationErrorf("indicators.last_n must be > 0")
	}
	return nil
}

func validateCooldownConfig(cfg CooldownConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.EntryCooldownSec <= 0 {
		return validationErrorf("cooldown.entry_cooldown_sec must be > 0")
	}
	return nil
}

func validateConsensusConfig(cfg ConsensusConfig) error {
	if cfg.ScoreThreshold < 0 || cfg.ScoreThreshold > 1 {
		return validationErrorf("consensus.score_threshold must be in [0,1]")
	}
	if cfg.ConfidenceThreshold < 0 || cfg.ConfidenceThreshold > 1 {
		return validationErrorf("consensus.confidence_threshold must be in [0,1]")
	}
	return nil
}

func validateLLMConfig(cfg SymbolLLMConfig, enabled AgentEnabled) error {
	if err := validateLLMRoleEnabled("llm.agent.indicator", cfg.Agent.Indicator, enabled.Indicator); err != nil {
		return err
	}
	if err := validateLLMRoleEnabled("llm.agent.structure", cfg.Agent.Structure, enabled.Structure); err != nil {
		return err
	}
	if err := validateLLMRoleEnabled("llm.agent.mechanics", cfg.Agent.Mechanics, enabled.Mechanics); err != nil {
		return err
	}
	if err := validateLLMRoleEnabled("llm.provider.indicator", cfg.Provider.Indicator, enabled.Indicator); err != nil {
		return err
	}
	if err := validateLLMRoleEnabled("llm.provider.structure", cfg.Provider.Structure, enabled.Structure); err != nil {
		return err
	}
	if err := validateLLMRoleEnabled("llm.provider.mechanics", cfg.Provider.Mechanics, enabled.Mechanics); err != nil {
		return err
	}
	return nil
}

func validateLLMRoleEnabled(prefix string, cfg LLMRoleConfig, enabled bool) error {
	if !enabled {
		return nil
	}
	return validateLLMRole(prefix, cfg)
}

func validateLLMRole(prefix string, cfg LLMRoleConfig) error {
	if strings.TrimSpace(cfg.Model) == "" {
		return validationErrorf("%s.model is required", prefix)
	}
	if cfg.Temperature == nil {
		return validationErrorf("%s.temperature is required", prefix)
	}
	if *cfg.Temperature < 0 {
		return validationErrorf("%s.temperature must be >= 0", prefix)
	}
	return nil
}

func ValidateSymbolLLMModels(sys SystemConfig, cfg SymbolConfig) error {
	enabled, err := ResolveAgentEnabled(cfg.Agent)
	if err != nil {
		return err
	}
	roles := []struct {
		path  string
		model string
		need  bool
	}{
		{"llm.agent.indicator", cfg.LLM.Agent.Indicator.Model, enabled.Indicator},
		{"llm.agent.structure", cfg.LLM.Agent.Structure.Model, enabled.Structure},
		{"llm.agent.mechanics", cfg.LLM.Agent.Mechanics.Model, enabled.Mechanics},
		{"llm.provider.indicator", cfg.LLM.Provider.Indicator.Model, enabled.Indicator},
		{"llm.provider.structure", cfg.LLM.Provider.Structure.Model, enabled.Structure},
		{"llm.provider.mechanics", cfg.LLM.Provider.Mechanics.Model, enabled.Mechanics},
	}
	for _, role := range roles {
		if !role.need {
			continue
		}
		model := strings.TrimSpace(role.model)
		if model == "" {
			continue
		}
		if _, ok := LookupLLMModelConfig(sys, model); !ok {
			return validationErrorf("%s.model=%s not found in system llm_models", role.path, model)
		}
	}
	return nil
}

func requiredKlineLimit(cfg SymbolConfig) int {
	trendRequired := TrendPresetRequiredBars(cfg.Intervals)
	required := maxInt(
		cfg.Indicators.EMAFast,
		cfg.Indicators.EMAMid,
		cfg.Indicators.EMASlow,
		cfg.Indicators.RSIPeriod,
		cfg.Indicators.ATRPeriod,
		cfg.Indicators.MACDFast,
		cfg.Indicators.MACDSlow,
		cfg.Indicators.MACDSignal,
		trendRequired,
	)
	required = max(1, required)
	return required + 1
}

func maxInt(values ...int) int {
	maxVal := 0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
