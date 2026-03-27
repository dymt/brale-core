package decisionfmt

import (
	"fmt"
	"strings"
)

func (f DefaultFormatter) RenderGateText(report GateReport) string {
	lines := []string{
		fmt.Sprintf("全局可交易: %s", report.Overall.TradeableText),
	}
	decisionText := report.Overall.DecisionText
	if strings.TrimSpace(decisionText) == "" {
		decisionText = report.Overall.DecisionAction
	}
	if strings.TrimSpace(decisionText) != "" {
		if strings.TrimSpace(report.Overall.DecisionAction) != "" && report.Overall.DecisionAction != decisionText {
			lines = append(lines, fmt.Sprintf("决策: %s (%s)", decisionText, report.Overall.DecisionAction))
		} else {
			lines = append(lines, fmt.Sprintf("决策: %s", decisionText))
		}
	}
	lines = append(lines, fmt.Sprintf("Grade: %d", report.Overall.Grade))
	reason := report.Overall.Reason
	if strings.TrimSpace(reason) == "" {
		reason = "—"
	}
	if strings.TrimSpace(report.Overall.ReasonCode) != "" {
		lines = append(lines, fmt.Sprintf("逻辑解释: %s (原因码: %s)", reason, report.Overall.ReasonCode))
	} else {
		lines = append(lines, fmt.Sprintf("逻辑解释: %s", reason))
	}
	if strings.TrimSpace(report.Overall.Direction) != "" && report.Overall.Direction != "—" {
		lines = append(lines, fmt.Sprintf("方向: %s", report.Overall.Direction))
	}
	if report.RuleHit != nil && strings.TrimSpace(report.RuleHit.Name) != "" {
		ruleText := fmt.Sprintf("命中规则: %s (priority %d)", displayGateReasonCode(report.RuleHit.Name), report.RuleHit.Priority)
		if strings.TrimSpace(report.RuleHit.Action) != "" {
			ruleText = fmt.Sprintf("%s, action=%s", ruleText, translateDecisionAction(report.RuleHit.Action))
		}
		if strings.TrimSpace(report.RuleHit.Reason) != "" {
			ruleText = fmt.Sprintf("%s, reason=%s", ruleText, displayGateReasonCode(report.RuleHit.Reason))
		}
		lines = append(lines, ruleText)
	}
	if trace := formatGateTrace(report.Derived); trace != "" {
		lines = append(lines, fmt.Sprintf("Gate 过程: %s", trace))
	}
	if summary := formatDerivedSummary(report.Derived); summary != "" {
		lines = append(lines, fmt.Sprintf("Derived: %s", summary))
	}
	for _, p := range report.Providers {
		label := providerRoleLabel(p.Role)
		line := fmt.Sprintf("%s Provider: %s", label, p.TradeableText)
		if len(p.Factors) > 0 {
			parts := make([]string, 0, len(p.Factors))
			for _, f := range p.Factors {
				parts = append(parts, fmt.Sprintf("%s=%s", f.Label, f.Status))
			}
			line = fmt.Sprintf("%s | %s", line, strings.Join(parts, "；"))
		}
		lines = append(lines, line)
	}
	if report.Overall.ExpectedSnapID > 0 {
		lines = append(lines, fmt.Sprintf("说明: Gate 事件缺失 (snapshot %d)", report.Overall.ExpectedSnapID))
	}
	return strings.Join(lines, "\n")
}
