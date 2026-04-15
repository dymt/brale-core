package decisionfmt

import (
	"fmt"
	"html"
	"slices"
	"strings"
	"unicode/utf8"
)

func (f DefaultFormatter) RenderDecisionHTML(report DecisionReport) string {
	const telegramHTMLLimit = 4096
	baseSections := []string{
		renderHTMLHeader(report),
		renderHTMLNarrative(report),
	}
	metricsSection := renderHTMLMetrics(report)
	monitorSection := renderHTMLMonitorDetail(report)
	riskSection := renderHTMLRiskDetail(report)
	sections := slices.Clone(baseSections)
	sections = append(sections, metricsSection, monitorSection, riskSection)
	assembled := joinHTMLSections(sections)
	if utf8.RuneCountInString(assembled) > telegramHTMLLimit {
		if riskSection != "" {
			sections = slices.Clone(baseSections)
			sections = append(sections, metricsSection)
			assembled = joinHTMLSections(sections)
		}
	}
	if utf8.RuneCountInString(assembled) > telegramHTMLLimit {
		if metricsSection != "" {
			sections = slices.Clone(baseSections)
			assembled = joinHTMLSections(sections)
		}
	}
	if utf8.RuneCountInString(assembled) > telegramHTMLLimit {
		assembled = trimTelegramHTML(assembled, telegramHTMLLimit)
	}
	return assembled
}

func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(text)
}

func renderHTMLHeader(report DecisionReport) string {
	decisionText := strings.TrimSpace(report.Gate.Overall.DecisionText)
	if execTitle := resolveExecutionTitle(report); execTitle != "" {
		decisionText = execTitle
	}
	if decisionText == "" {
		decisionText = strings.TrimSpace(report.Gate.Overall.DecisionAction)
	}
	if decisionText == "" {
		decisionText = "—"
	}
	line1 := fmt.Sprintf("%s %s 决策报告 (Snapshot: %d)", decisionText, report.Symbol, report.SnapshotID)
	statusLabel := "可交易"
	gateText := strings.TrimSpace(report.Gate.Overall.TradeableText)
	if label, text, ok := resolveHoldStatusLine(report); ok {
		statusLabel = label
		gateText = text
	}
	line2Parts := []string{fmt.Sprintf("%s: %s", statusLabel, gateText)}
	if gateText == "" {
		line2Parts = line2Parts[:0]
	}
	action := strings.TrimSpace(report.Gate.Overall.DecisionAction)
	if action != "" && action != decisionText {
		line2Parts = append(line2Parts, fmt.Sprintf("动作: %s", action))
	}
	direction := strings.TrimSpace(report.Gate.Overall.Direction)
	if direction != "" && direction != "无方向" {
		line2Parts = append(line2Parts, fmt.Sprintf("方向: %s", direction))
	}
	line2 := strings.Join(line2Parts, " | ")
	line3 := fmt.Sprintf("时间: %s | 价格: %s", formatReportTime(), formatCurrentPrice(report))
	content := escapeHTML(line1)
	if strings.TrimSpace(line2) != "" {
		content = fmt.Sprintf("%s\n%s", content, escapeHTML(line2))
	}
	if strings.TrimSpace(line3) != "" {
		content = fmt.Sprintf("%s\n%s", content, escapeHTML(line3))
	}
	return fmt.Sprintf("<b>标题区</b>\n%s", wrapPre(content))
}

func renderHTMLNarrative(report DecisionReport) string {
	summary := pickNarrativeSummary(report)
	if strings.TrimSpace(summary) == "" {
		summary = "—"
	}
	sections := splitNarrativeSections(summary)
	labels := []struct {
		Key   string
		Label string
	}{
		{Key: "状态", Label: "🧭 状态"},
		{Key: "动作", Label: "⚙️ 动作"},
		{Key: "冲突", Label: "⚔️ 冲突"},
		{Key: "风险", Label: "🛡️ 风险"},
	}
	parts := make([]string, 0, 1+len(labels)*2)
	parts = append(parts, "<b>叙事分析</b>")
	for idx, label := range labels {
		value := strings.TrimSpace(sections[label.Key])
		if label.Key == "冲突" && isPlaceholderValue(value) {
			continue
		}
		if isPlaceholderValue(value) {
			value = "—"
		}
		if idx > 0 && len(parts) > 1 {
			parts = append(parts, "")
		}
		parts = append(parts, fmt.Sprintf("<b>%s</b>", label.Label))
		parts = append(parts, wrapPre(escapeHTML(value)))
	}
	return strings.Join(parts, "\n")
}

func renderHTMLRiskDetail(report DecisionReport) string {
	if !shouldRenderRiskDetail(report) {
		return ""
	}
	action := strings.TrimSpace(fmt.Sprint(report.Gate.Derived["sieve_action"]))
	reasonCode := strings.TrimSpace(fmt.Sprint(report.Gate.Derived["sieve_reason"]))
	if action == "" && reasonCode == "" {
		return ""
	}
	lines := make([]string, 0, 3)
	if action != "" {
		actionLabel := translateDecisionAction(action)
		if strings.TrimSpace(actionLabel) == "" {
			actionLabel = action
		}
		lines = append(lines, escapeHTML(fmt.Sprintf("拦截动作: %s", actionLabel)))
	}
	if reasonCode != "" {
		lines = append(lines, fmt.Sprintf("拦截代码: <code>%s</code>", escapeHTML(reasonCode)))
		reasonLabel := translateSieveReasonCode(reasonCode)
		if strings.TrimSpace(reasonLabel) != "" {
			lines = append(lines, escapeHTML(fmt.Sprintf("人话解释: %s", reasonLabel)))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("<b>风控拦截详情</b>\n%s", wrapPre(strings.Join(lines, "\n")))
}

func renderHTMLMonitorDetail(report DecisionReport) string {
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
	lines := []string{"当前状态: 持仓中，未放行新开仓", "详细展开:"}
	blocked := make([]string, 0, len(steps))
	for _, step := range steps {
		label := strings.TrimSpace(step.label)
		if label == "" {
			label = step.step
		}
		detail := formatMonitorTag(step.tag, step.reason)
		lines = append(lines, fmt.Sprintf("%s: %s", label, detail))
		if strings.EqualFold(step.tag, "keep") {
			blocked = append(blocked, fmt.Sprintf("%s=%s", label, detail))
		}
	}
	if len(blocked) > 0 {
		lines = append(lines, fmt.Sprintf("阻断步骤: %s", strings.Join(blocked, "；")))
	}
	return fmt.Sprintf("<b>持仓监控</b>\n%s", wrapPre(escapeHTML(strings.Join(lines, "\n"))))
}

func shouldRenderRiskDetail(report DecisionReport) bool {
	if len(report.Gate.Derived) == 0 {
		return false
	}
	action := strings.ToUpper(strings.TrimSpace(report.Gate.Overall.DecisionAction))
	if action != "WAIT" && action != "VETO" {
		return false
	}
	if strings.TrimSpace(fmt.Sprint(report.Gate.Derived["sieve_action"])) != "" {
		return true
	}
	if strings.TrimSpace(fmt.Sprint(report.Gate.Derived["sieve_reason"])) != "" {
		return true
	}
	return false
}

func wrapPre(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		trimmed = "—"
	}
	return fmt.Sprintf("<pre>%s</pre>", trimmed)
}

func joinHTMLSections(sections []string) string {
	parts := make([]string, 0, len(sections))
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n\n")
}

func trimHTMLRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit])
}

func trimTelegramHTML(text string, limit int) string {
	trimmed := trimHTMLRunes(text, limit)
	if isBalancedTelegramHTML(trimmed) {
		return trimmed
	}

	plain := telegramHTMLToPlainText(text)
	if limit <= len("<pre></pre>") {
		return trimHTMLRunes(plain, limit)
	}
	bodyLimit := limit - len("<pre></pre>")
	plain = trimHTMLRunes(plain, bodyLimit)
	plain = strings.TrimSpace(plain)
	if plain == "" {
		plain = "—"
	}
	return wrapPre(escapeHTML(plain))
}

func isBalancedTelegramHTML(text string) bool {
	tags := [][2]string{{"<b>", "</b>"}, {"<pre>", "</pre>"}, {"<code>", "</code>"}}
	for _, tag := range tags {
		if strings.Count(text, tag[0]) != strings.Count(text, tag[1]) {
			return false
		}
	}
	return true
}

func telegramHTMLToPlainText(text string) string {
	replacer := strings.NewReplacer(
		"<b>", "", "</b>", "",
		"<pre>", "", "</pre>", "",
		"<code>", "", "</code>", "",
	)
	plain := replacer.Replace(text)
	return html.UnescapeString(plain)
}
