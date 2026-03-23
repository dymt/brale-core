package botruntime

import (
	"fmt"
	"strings"

	"brale-core/internal/pkg/parseutil"
)

func DescribeMonitorRiskPlan(plan MonitorRiskPlan) []string {
	lines := []string{fmt.Sprintf("风险计划: %s", plan.Label)}
	lines = append(lines, describeMonitorInitial(plan.Initial)...)
	lines = append(lines, describeMonitorTighten(plan.Tighten)...)
	lines = append(lines, fmt.Sprintf("入场定价: %s", plan.EntryPricing.Label))
	return lines
}

func describeMonitorInitial(section MonitorRiskPlanSection) []string {
	if section.Source == "llm" {
		return []string{"初始止盈止损: LLM生成"}
	}
	lines := []string{"初始止盈止损: Go规则"}
	if stopATR, ok := parseutil.FloatOK(section.Params["stop_atr_multiplier"]); ok && stopATR > 0 {
		lines = append(lines, fmt.Sprintf("初始止损: ATR x %.2f", stopATR))
	} else if stopMin, ok := parseutil.FloatOK(section.Params["stop_min_distance_pct"]); ok && stopMin > 0 {
		lines = append(lines, fmt.Sprintf("初始止损最小距离: %.4f", stopMin))
	}
	if rr := monitorFloatSlice(section.Params["take_profit_rr"]); len(rr) > 0 {
		lines = append(lines, "初始止盈: RR x "+formatMonitorValues(rr))
	}
	return lines
}

func describeMonitorTighten(section MonitorRiskPlanSection) []string {
	if section.Source == "llm" {
		return []string{"持仓收紧: LLM生成"}
	}
	lines := []string{"持仓收紧: Go规则"}
	if intervalSec, ok := parseutil.FloatOK(section.Params["min_update_interval_sec"]); ok && intervalSec > 0 {
		lines = append(lines, fmt.Sprintf("收紧间隔: %ds", int64(intervalSec)))
	}
	if feePct, ok := parseutil.FloatOK(section.Params["breakeven_fee_pct"]); ok {
		lines = append(lines, fmt.Sprintf("保本偏移: %.4f", feePct))
	}
	return lines
}

func monitorFloatSlice(raw any) []float64 {
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

func formatMonitorValues(values []float64) string {
	if len(values) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.2f", value))
	}
	return strings.Join(parts, " / ")
}
