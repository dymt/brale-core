package config

import (
	"strings"

	"brale-core/internal/interval"
)

func ValidateStrategyConfig(cfg StrategyConfig) error {
	if _, err := validateCanonicalSymbol("symbol", cfg.Symbol); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.ID) == "" {
		return validationErrorf("id is required")
	}
	if strings.TrimSpace(cfg.RuleChainPath) == "" {
		return validationErrorf("rule_chain is required")
	}
	if err := validateRiskManagement(cfg.RiskManagement); err != nil {
		return err
	}
	return nil
}

func validateRiskManagement(cfg RiskManagementConfig) error {
	if err := validateRiskStrategyMode("risk_management.risk_strategy.mode", cfg.RiskStrategy.Mode); err != nil {
		return err
	}
	if err := validateRiskManagementValues(cfg); err != nil {
		return err
	}
	if err := validateRiskManagementEntry(cfg); err != nil {
		return err
	}
	if err := validateRiskManagementInitialExit(cfg); err != nil {
		return err
	}
	if err := validateRiskManagementTighten(cfg); err != nil {
		return err
	}
	return validateSieveConfig(cfg.Sieve)
}

func validateRiskStrategyMode(field, mode string) error {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return nil
	}
	switch normalized {
	case "llm", "native":
		return nil
	default:
		return validationErrorf("%s must be llm/native", field)
	}
}

func validateRiskManagementValues(cfg RiskManagementConfig) error {
	if cfg.RiskPerTradePct <= 0 {
		return validationErrorf("risk_management.risk_per_trade_pct must be > 0")
	}
	if cfg.MaxInvestPct <= 0 || cfg.MaxInvestPct > 1 {
		return validationErrorf("risk_management.max_invest_pct must be in (0,1]")
	}
	if cfg.MaxLeverage <= 0 {
		return validationErrorf("risk_management.max_leverage must be > 0")
	}
	if cfg.Grade1Factor <= 0 || cfg.Grade1Factor > 1 {
		return validationErrorf("risk_management.grade_1_factor must be in (0,1]")
	}
	if cfg.Grade2Factor <= 0 || cfg.Grade2Factor > 1 {
		return validationErrorf("risk_management.grade_2_factor must be in (0,1]")
	}
	if cfg.Grade3Factor <= 0 || cfg.Grade3Factor > 1 {
		return validationErrorf("risk_management.grade_3_factor must be in (0,1]")
	}
	if cfg.EntryOffsetATR < 0 {
		return validationErrorf("risk_management.entry_offset_atr must be >= 0")
	}
	if cfg.BreakevenFeePct < 0 {
		return validationErrorf("risk_management.breakeven_fee_pct must be >= 0")
	}
	if cfg.SlippageBufferPct < 0 {
		return validationErrorf("risk_management.slippage_buffer_pct must be >= 0")
	}
	return nil
}

func validateRiskManagementEntry(cfg RiskManagementConfig) error {
	entryMode := strings.ToLower(strings.TrimSpace(cfg.EntryMode))
	if entryMode != "" {
		switch entryMode {
		case "orderbook", "atr_offset", "market":
		default:
			return validationErrorf("risk_management.entry_mode must be orderbook/atr_offset/market")
		}
	}
	if cfg.OrderbookDepth != 0 {
		allowed := map[int]struct{}{5: {}, 10: {}, 20: {}, 50: {}, 100: {}, 500: {}, 1000: {}}
		if cfg.OrderbookDepth <= 0 {
			return validationErrorf("risk_management.orderbook_depth must be > 0")
		}
		if _, ok := allowed[cfg.OrderbookDepth]; !ok {
			return validationErrorf("risk_management.orderbook_depth must be one of 5/10/20/50/100/500/1000")
		}
	}
	return nil
}

func validateRiskManagementTighten(cfg RiskManagementConfig) error {
	if cfg.TightenATR.StructureThreatened <= 0 {
		return validationErrorf("risk_management.tighten_atr.structure_threatened must be > 0")
	}
	if cfg.TightenATR.TP1ATR < 0 {
		return validationErrorf("risk_management.tighten_atr.tp1_atr must be >= 0")
	}
	if cfg.TightenATR.TP2ATR < 0 {
		return validationErrorf("risk_management.tighten_atr.tp2_atr must be >= 0")
	}
	if cfg.TightenATR.MinTPDistancePct < 0 || cfg.TightenATR.MinTPDistancePct >= 1 {
		return validationErrorf("risk_management.tighten_atr.min_tp_distance_pct must be in [0,1)")
	}
	if cfg.TightenATR.MinTPGapPct < 0 || cfg.TightenATR.MinTPGapPct >= 1 {
		return validationErrorf("risk_management.tighten_atr.min_tp_gap_pct must be in [0,1)")
	}
	if cfg.TightenATR.MinUpdateIntervalSec < 0 {
		return validationErrorf("risk_management.tighten_atr.min_update_interval_sec must be >= 0")
	}
	return nil
}

func validateRiskManagementInitialExit(cfg RiskManagementConfig) error {
	policy := strings.TrimSpace(cfg.InitialExit.Policy)
	if policy == "" {
		return validationErrorf("risk_management.initial_exit.policy is required")
	}
	structureInterval := strings.ToLower(strings.TrimSpace(cfg.InitialExit.StructureInterval))
	if structureInterval != "" && structureInterval != "auto" {
		if _, err := interval.ParseInterval(structureInterval); err != nil {
			return validationErrorf("risk_management.initial_exit.structure_interval must be auto or a valid interval")
		}
	}
	if err := initialExitPolicyValidator(policy, cfg.InitialExit.Params); err != nil {
		return validationErrorf("risk_management.initial_exit invalid: %v", err)
	}
	return nil
}

func validateSieveConfig(cfg RiskManagementSieveConfig) error {
	if cfg.MinSizeFactor < 0 || cfg.MinSizeFactor > 1 {
		return validationErrorf("risk_management.sieve.min_size_factor must be in [0,1]")
	}
	if cfg.DefaultSizeFactor < 0 || cfg.DefaultSizeFactor > 1 {
		return validationErrorf("risk_management.sieve.default_size_factor must be in [0,1]")
	}
	defaultAction := strings.ToUpper(strings.TrimSpace(cfg.DefaultGateAction))
	if defaultAction != "" && defaultAction != "ALLOW" && defaultAction != "WAIT" && defaultAction != "VETO" {
		return validationErrorf("risk_management.sieve.default_gate_action must be ALLOW/WAIT/VETO")
	}
	allowedMechanics := map[string]struct{}{
		"fuel_ready":          {},
		"neutral":             {},
		"crowded_long":        {},
		"crowded_short":       {},
		"liquidation_cascade": {},
	}
	allowedConf := map[string]struct{}{
		"high": {},
		"low":  {},
	}
	for idx, row := range cfg.Rows {
		mech := strings.ToLower(strings.TrimSpace(row.MechanicsTag))
		if mech == "" {
			return validationErrorf("risk_management.sieve.rows[%d].mechanics_tag is required", idx)
		}
		if _, ok := allowedMechanics[mech]; !ok {
			return validationErrorf("risk_management.sieve.rows[%d].mechanics_tag must be one of fuel_ready/neutral/crowded_long/crowded_short/liquidation_cascade", idx)
		}
		conf := strings.ToLower(strings.TrimSpace(row.LiqConfidence))
		if conf == "" {
			return validationErrorf("risk_management.sieve.rows[%d].liq_confidence is required", idx)
		}
		if _, ok := allowedConf[conf]; !ok {
			return validationErrorf("risk_management.sieve.rows[%d].liq_confidence must be high/low", idx)
		}
		action := strings.ToUpper(strings.TrimSpace(row.GateAction))
		if action == "" {
			return validationErrorf("risk_management.sieve.rows[%d].gate_action is required", idx)
		}
		if action != "ALLOW" && action != "WAIT" && action != "VETO" {
			return validationErrorf("risk_management.sieve.rows[%d].gate_action must be ALLOW/WAIT/VETO", idx)
		}
		if row.SizeFactor < 0 || row.SizeFactor > 1 {
			return validationErrorf("risk_management.sieve.rows[%d].size_factor must be in [0,1]", idx)
		}
	}
	return nil
}
