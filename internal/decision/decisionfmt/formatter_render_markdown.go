package decisionfmt

import (
	"fmt"
	"strings"
)

func (f DefaultFormatter) RenderDecisionMarkdown(report DecisionReport) string {
	var b strings.Builder
	gateLabel := "Gate"
	gateText := report.Gate.Overall.TradeableText
	if label, text, ok := resolveHoldStatusLine(report); ok {
		gateLabel = label
		gateText = text
	}
	fmt.Fprintf(&b, "[%s][snapshot:%d] %s: %s\n", report.Symbol, report.SnapshotID, gateLabel, gateText)
	fmt.Fprintf(&b, "时间: %s | 价格: %s\n\n", formatReportTime(), formatCurrentPrice(report))
	decisionText := report.Gate.Overall.DecisionText
	if strings.TrimSpace(decisionText) == "" {
		decisionText = report.Gate.Overall.DecisionAction
	}
	if strings.TrimSpace(decisionText) != "" {
		if strings.TrimSpace(report.Gate.Overall.DecisionAction) != "" && report.Gate.Overall.DecisionAction != decisionText {
			fmt.Fprintf(&b, "Gate 决策: %s (%s)\n", decisionText, report.Gate.Overall.DecisionAction)
		} else {
			fmt.Fprintf(&b, "Gate 决策: %s\n", decisionText)
		}
	}
	fmt.Fprintf(&b, "Gate Grade: %d\n", report.Gate.Overall.Grade)
	if report.Gate.Overall.Reason != "" {
		if report.Gate.Overall.ReasonCode != "" {
			fmt.Fprintf(&b, "Gate 原因: %s (原因码: %s)\n", report.Gate.Overall.Reason, report.Gate.Overall.ReasonCode)
		} else {
			fmt.Fprintf(&b, "Gate 原因: %s\n", report.Gate.Overall.Reason)
		}
	}
	if report.Gate.RuleHit != nil && strings.TrimSpace(report.Gate.RuleHit.Name) != "" {
		ruleText := fmt.Sprintf("Gate 命中规则: %s (priority %d)", displayGateReasonCode(report.Gate.RuleHit.Name), report.Gate.RuleHit.Priority)
		if strings.TrimSpace(report.Gate.RuleHit.Action) != "" {
			ruleText = fmt.Sprintf("%s, action=%s", ruleText, translateDecisionAction(report.Gate.RuleHit.Action))
		}
		if strings.TrimSpace(report.Gate.RuleHit.Reason) != "" {
			ruleText = fmt.Sprintf("%s, reason=%s", ruleText, displayGateReasonCode(report.Gate.RuleHit.Reason))
		}
		fmt.Fprintf(&b, "%s\n", ruleText)
	}
	if trace := formatGateTrace(report.Gate.Derived); trace != "" {
		fmt.Fprintf(&b, "Gate 过程: %s\n", trace)
	}
	if summary := formatDerivedSummary(report.Gate.Derived); summary != "" {
		fmt.Fprintf(&b, "Derived: %s\n", summary)
	}
	if report.Gate.Overall.Reason != "" || report.Gate.RuleHit != nil || len(report.Gate.Derived) > 0 {
		fmt.Fprint(&b, "\n")
	}
	if monitor := renderMonitorMarkdown(report); strings.TrimSpace(monitor) != "" {
		fmt.Fprintf(&b, "%s\n\n", monitor)
	}
	writeStageMarkdown(&b, "Provider", report.Providers)
	writeStageMarkdown(&b, "Agent", report.Agents)
	return strings.TrimSpace(b.String())
}

func writeStageMarkdown(b *strings.Builder, label string, stages []StageOutput) {
	if len(stages) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", label)
	for _, item := range stages {
		roleLabel := providerRoleLabel(item.Role)
		if roleLabel == "" {
			roleLabel = item.Role
		}
		fmt.Fprintf(b, "- %s\n", roleLabel)
		if item.Summary != "" {
			lines := strings.Split(item.Summary, "\n")
			for _, line := range lines {
				fmt.Fprintf(b, "  %s\n", line)
			}
		}
	}
	b.WriteString("\n")
}

func renderMonitorMarkdown(report DecisionReport) string {
	if !isHoldDecision(report.Gate.Overall.DecisionAction) {
		return ""
	}
	if report.Gate.Derived == nil {
		return ""
	}
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(report.Gate.Derived["gate_trace_mode"])))
	if mode != "monitor" {
		return ""
	}
	steps := collectMonitorTrace(report.Gate.Derived)
	if len(steps) == 0 {
		return ""
	}
	lines := []string{"持仓监控", "当前状态: 持仓中，未放行新开仓", "详细展开:"}
	blocked := make([]string, 0, len(steps))
	for _, step := range steps {
		label := strings.TrimSpace(step.label)
		if label == "" {
			label = step.step
		}
		detail := formatMonitorTag(step.tag, step.reason)
		lines = append(lines, fmt.Sprintf("- %s: %s", label, detail))
		if strings.EqualFold(step.tag, "keep") {
			blocked = append(blocked, fmt.Sprintf("%s=%s", label, detail))
		}
	}
	if len(blocked) > 0 {
		lines = append(lines, fmt.Sprintf("阻断步骤: %s", strings.Join(blocked, "；")))
	}
	return strings.Join(lines, "\n")
}

type monitorTraceStep struct {
	step   string
	label  string
	tag    string
	reason string
}

func collectMonitorTrace(derived map[string]any) []monitorTraceStep {
	stepsRaw, ok := derived["gate_trace"].([]any)
	if !ok || len(stepsRaw) == 0 {
		return nil
	}
	steps := make([]monitorTraceStep, 0, len(stepsRaw))
	for _, raw := range stepsRaw {
		stepMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		stepKey := strings.TrimSpace(fmt.Sprint(stepMap["step"]))
		if stepKey == "" || stepKey == "<nil>" {
			continue
		}
		label := translateGateStep(stepKey)
		if strings.TrimSpace(label) == "" {
			label = stepKey
		}
		tag := strings.TrimSpace(fmt.Sprint(stepMap["tag"]))
		if tag == "<nil>" {
			tag = ""
		}
		reason := strings.TrimSpace(fmt.Sprint(stepMap["reason"]))
		if reason == "<nil>" {
			reason = ""
		}
		steps = append(steps, monitorTraceStep{
			step:   stepKey,
			label:  label,
			tag:    tag,
			reason: reason,
		})
	}
	return steps
}

func formatMonitorTag(tag, reason string) string {
	label := strings.TrimSpace(translateDecisionAction(tag))
	if label == "" {
		label = tag
	}
	if strings.TrimSpace(reason) == "" {
		return label
	}
	return fmt.Sprintf("%s（原因：%s）", label, reason)
}
