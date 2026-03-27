package decisionfmt

import (
	"fmt"
	"math"
	"strings"
)

func renderHTMLMetrics(report DecisionReport) string {
	consensus := parseDirectionConsensusMetrics(report.Gate.Derived)
	metrics := make([]string, 0, 10)
	metrics = appendBaseHTMLMetrics(metrics, report, consensus)
	metrics = appendTightenHTMLMetrics(metrics, report)
	metrics = append(metrics, "可交易判定: 指标=动量扩张&趋势一致&无噪音; 结构=结构清晰&完整; 市场机制=无清算压力")
	metrics = append(metrics, "方向权重: 结构1.0 / 指标0.7 / 市场机制0.5")
	metrics = appendRuleAndStopHTMLMetrics(metrics, report)
	metrics = appendConsensusHTMLMetrics(metrics, report, consensus)
	return renderHTMLMetricsBlock(metrics)
}

func appendBaseHTMLMetrics(metrics []string, report DecisionReport, consensus *directionConsensusMetrics) []string {
	if consensus != nil && consensus.ConfidenceOK {
		metrics = append(metrics, fmt.Sprintf("信心指数: %s / 100", formatExecutionFloat(consensus.Confidence*100)))
	} else {
		metrics = append(metrics, fmt.Sprintf("信心指数: %d", report.Gate.Overall.Grade))
	}
	metrics = append(metrics, fmt.Sprintf("Gate等级: %d", report.Gate.Overall.Grade))
	statusLabel, gateText := resolveStatusHTMLMetric(report)
	if gateText != "" {
		metrics = append(metrics, fmt.Sprintf("%s: %s", statusLabel, gateText))
	}
	reason := strings.TrimSpace(report.Gate.Overall.Reason)
	if reason != "" {
		metrics = append(metrics, fmt.Sprintf("主要原因: %s", reason))
	}
	direction := strings.TrimSpace(report.Gate.Overall.Direction)
	if direction != "" {
		metrics = append(metrics, fmt.Sprintf("趋势方向: %s", direction))
	}
	return metrics
}

func resolveStatusHTMLMetric(report DecisionReport) (statusLabel, gateText string) {
	statusLabel = "可交易"
	gateText = strings.TrimSpace(report.Gate.Overall.TradeableText)
	if label, text, ok := resolveHoldStatusLine(report); ok {
		statusLabel = label
		gateText = text
	}
	return statusLabel, gateText
}

func appendTightenHTMLMetrics(metrics []string, report DecisionReport) []string {
	exec := parseExecutionSummary(report.Gate.Derived)
	if exec == nil || !strings.EqualFold(exec.Action, "tighten") {
		return metrics
	}
	metrics = append(metrics, fmt.Sprintf("执行结论: %s", formatExecutionSummary(*exec)))
	if exec.ScoreThreshold > 0 || exec.ScoreParseOK {
		metrics = append(metrics, fmt.Sprintf("收紧评分: %s / %s", formatExecutionFloat(exec.ScoreTotal), formatExecutionFloat(exec.ScoreThreshold)))
	}
	if len(exec.BlockedBy) > 0 {
		metrics = append(metrics, fmt.Sprintf("阻断原因: %s", formatExecutionBlockedReasons(exec.BlockedBy)))
		if blockedStage := formatExecutionBlockedStages(exec.BlockedBy); blockedStage != "" {
			metrics = append(metrics, fmt.Sprintf("收紧阻隔环节: %s", blockedStage))
		}
	}
	if exec.ATRChangePctOK || exec.MonitorGateHit || exec.ATRThreshold > 0 {
		atrChangeText := "—"
		if exec.ATRChangePctOK {
			atrChangeText = formatExecutionFloat(math.Abs(exec.ATRChangePct))
		}
		metrics = append(metrics, fmt.Sprintf("收紧门槛: 监控收紧=%s; |ATR变化|=%s (>=%s)", translateBoolStatus(exec.MonitorGateHit), atrChangeText, formatExecutionFloat(exec.ATRThreshold)))
	}
	return metrics
}

func appendRuleAndStopHTMLMetrics(metrics []string, report DecisionReport) []string {
	if report.Gate.RuleHit != nil {
		ruleName := strings.TrimSpace(report.Gate.RuleHit.Name)
		if ruleName != "" {
			metrics = append(metrics, fmt.Sprintf("命中规则: %s", displayGateReasonCode(ruleName)))
		}
	}
	stopStep := strings.TrimSpace(fmt.Sprint(report.Gate.Derived["gate_stop_step"]))
	if stopStep != "" {
		metrics = append(metrics, fmt.Sprintf("停在步骤: %s", displayGateStep(stopStep)))
	}
	return metrics
}

func appendConsensusHTMLMetrics(metrics []string, report DecisionReport, consensus *directionConsensusMetrics) []string {
	if !strings.EqualFold(strings.TrimSpace(report.Gate.Overall.ReasonCode), "CONSENSUS_NOT_PASSED") || consensus == nil {
		return metrics
	}
	sourceScores := make([]string, 0, 3)
	if consensus.StructureScoreOK {
		sourceScores = append(sourceScores, fmt.Sprintf("结构=%s", formatExecutionFloat(consensus.StructureScore)))
	}
	if consensus.IndicatorScoreOK {
		sourceScores = append(sourceScores, fmt.Sprintf("指标=%s", formatExecutionFloat(consensus.IndicatorScore)))
	}
	if consensus.MechanicsScoreOK {
		sourceScores = append(sourceScores, fmt.Sprintf("市场机制=%s", formatExecutionFloat(consensus.MechanicsScore)))
	}
	if len(sourceScores) > 0 {
		metrics = append(metrics, fmt.Sprintf("三路得分: %s", strings.Join(sourceScores, " / ")))
	}
	if consensus.ScoreOK {
		if consensus.ScoreThresholdOK {
			metrics = append(metrics, fmt.Sprintf("共识总分: %s (需达到 |score| >= %s)", formatExecutionFloat(consensus.Score), formatExecutionFloat(consensus.ScoreThreshold)))
		} else {
			metrics = append(metrics, fmt.Sprintf("共识总分: %s", formatExecutionFloat(consensus.Score)))
		}
	}
	if consensus.ConfidenceOK && consensus.ConfidenceThresholdOK {
		metrics = append(metrics, fmt.Sprintf("共识置信: %s (需达到 >= %s)", formatExecutionFloat(consensus.Confidence), formatExecutionFloat(consensus.ConfidenceThreshold)))
	}
	return metrics
}

func renderHTMLMetricsBlock(metrics []string) string {
	lines := make([]string, 0, len(metrics))
	for _, item := range metrics {
		lines = append(lines, escapeHTML(item))
	}
	return fmt.Sprintf("<b>关键仪表</b>\n%s", wrapPre(strings.Join(lines, "\n")))
}
