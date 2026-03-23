package runtimeapi

import (
	"fmt"
	"strings"

	"brale-core/internal/pkg/parseutil"
)

func buildMonitorRiskPlan(bundle ConfigBundle) MonitorRiskPlan {
	riskMgmt := bundle.Strategy.RiskManagement
	mode := normalizeMonitorRiskPlanMode(riskMgmt.RiskStrategy.Mode)
	entryMode := strings.TrimSpace(riskMgmt.EntryMode)
	if entryMode == "" {
		entryMode = "signal"
	}
	plan := MonitorRiskPlan{
		Mode:  mode,
		Label: monitorRiskPlanModeLabel(mode),
		EntryPricing: MonitorEntryPricing{
			Mode:  entryMode,
			Label: entryMode,
		},
	}
	if mode == "llm" {
		plan.Initial = MonitorRiskPlanSection{Source: "llm", Label: "LLM生成"}
		plan.Tighten = MonitorRiskPlanSection{Source: "llm", Label: "LLM生成"}
		return plan
	}
	plan.Initial = MonitorRiskPlanSection{Source: "go", Label: "Go规则", Params: buildNativeInitialRiskPlanParams(bundle)}
	plan.Tighten = MonitorRiskPlanSection{Source: "go", Label: "Go规则", Params: buildNativeTightenRiskPlanParams(bundle)}
	return plan
}

func normalizeMonitorRiskPlanMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "llm") {
		return "llm"
	}
	return "native"
}

func monitorRiskPlanModeLabel(mode string) string {
	if mode == "llm" {
		return "LLM生成"
	}
	return "Go规则"
}

func buildNativeInitialRiskPlanParams(bundle ConfigBundle) map[string]any {
	riskMgmt := bundle.Strategy.RiskManagement
	params := map[string]any{
		"policy":           strings.TrimSpace(riskMgmt.InitialExit.Policy),
		"risk_pct":         riskMgmt.RiskPerTradePct,
		"max_leverage":     riskMgmt.MaxLeverage,
		"entry_offset_atr": riskMgmt.EntryOffsetATR,
	}
	if structureInterval := strings.TrimSpace(riskMgmt.InitialExit.StructureInterval); structureInterval != "" {
		params["structure_interval"] = structureInterval
	}
	if value, ok := parseutil.FloatOK(riskMgmt.InitialExit.Params["stop_atr_multiplier"]); ok && value > 0 {
		params["stop_atr_multiplier"] = value
	}
	if value, ok := parseutil.FloatOK(riskMgmt.InitialExit.Params["stop_min_distance_pct"]); ok && value > 0 {
		params["stop_min_distance_pct"] = value
	}
	if values := monitorRiskPlanFloatSlice(riskMgmt.InitialExit.Params["take_profit_rr"]); len(values) > 0 {
		params["take_profit_rr"] = values
	}
	if values := monitorRiskPlanFloatSlice(riskMgmt.InitialExit.Params["take_profit_ratios"]); len(values) > 0 {
		params["take_profit_ratios"] = values
	}
	return params
}

func buildNativeTightenRiskPlanParams(bundle ConfigBundle) map[string]any {
	riskMgmt := bundle.Strategy.RiskManagement
	return map[string]any{
		"breakeven_fee_pct":             riskMgmt.BreakevenFeePct,
		"min_update_interval_sec":       riskMgmt.TightenATR.MinUpdateIntervalSec,
		"structure_threatened_atr_mult": riskMgmt.TightenATR.StructureThreatened,
		"tp1_atr":                       riskMgmt.TightenATR.TP1ATR,
		"tp2_atr":                       riskMgmt.TightenATR.TP2ATR,
		"min_tp_distance_pct":           riskMgmt.TightenATR.MinTPDistancePct,
		"min_tp_gap_pct":                riskMgmt.TightenATR.MinTPGapPct,
	}
}

func monitorRiskPlanFloatSlice(raw any) []float64 {
	switch values := raw.(type) {
	case []float64:
		return append([]float64(nil), values...)
	case []any:
		out := make([]float64, 0, len(values))
		for _, item := range values {
			if value, ok := parseutil.FloatOK(item); ok {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

func formatMonitorRiskPlanInitialLines(plan MonitorRiskPlan) []string {
	if plan.Initial.Source == "llm" {
		return []string{"初始止盈止损: LLM生成"}
	}
	lines := []string{"初始止盈止损: Go规则"}
	if stopATR, ok := parseutil.FloatOK(plan.Initial.Params["stop_atr_multiplier"]); ok && stopATR > 0 {
		lines = append(lines, fmt.Sprintf("初始止损: ATR x %.2f", stopATR))
	} else if stopMin, ok := parseutil.FloatOK(plan.Initial.Params["stop_min_distance_pct"]); ok && stopMin > 0 {
		lines = append(lines, fmt.Sprintf("初始止损最小距离: %.4f", stopMin))
	}
	if rr := monitorRiskPlanFloatSlice(plan.Initial.Params["take_profit_rr"]); len(rr) > 0 {
		lines = append(lines, "初始止盈: RR x "+formatRRList(rr))
	}
	return lines
}

func formatMonitorRiskPlanTightenLines(plan MonitorRiskPlan) []string {
	if plan.Tighten.Source == "llm" {
		return []string{"持仓收紧: LLM生成"}
	}
	lines := []string{"持仓收紧: Go规则"}
	if intervalSec, ok := parseutil.FloatOK(plan.Tighten.Params["min_update_interval_sec"]); ok && intervalSec > 0 {
		lines = append(lines, fmt.Sprintf("收紧间隔: %ds", int64(intervalSec)))
	}
	if feePct, ok := parseutil.FloatOK(plan.Tighten.Params["breakeven_fee_pct"]); ok {
		lines = append(lines, fmt.Sprintf("保本偏移: %.4f", feePct))
	}
	return lines
}

func formatRRList(values []float64) string {
	if len(values) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.2f", value))
	}
	return strings.Join(parts, " / ")
}
