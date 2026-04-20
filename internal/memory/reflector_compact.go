package memory

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	reflectionCompactRiskEventLimit = 5
	reflectionCompactTextLimit      = 140
)

type reflectionCompactPrompt struct {
	TradeResult   reflectionCompactTradeResult    `json:"trade_result"`
	EntryDecision *reflectionCompactEntryDecision `json:"entry_decision,omitempty"`
	EntrySignals  *reflectionCompactEntrySignals  `json:"entry_signals,omitempty"`
	Management    *reflectionCompactManagement    `json:"management,omitempty"`
	Exits         []reflectionCompactExit         `json:"exits,omitempty"`
}

type reflectionCompactTradeResult struct {
	Symbol          string `json:"symbol,omitempty"`
	Direction       string `json:"direction,omitempty"`
	Entry           string `json:"entry,omitempty"`
	FinalExit       string `json:"final_exit,omitempty"`
	PnLPercent      string `json:"pnl_pct,omitempty"`
	Duration        string `json:"duration,omitempty"`
	ClosedAt        string `json:"closed_at,omitempty"`
	ExitReason      string `json:"exit_reason,omitempty"`
	ExitOrderStatus string `json:"exit_order_status,omitempty"`
}

type reflectionCompactEntryDecision struct {
	SnapshotID          uint     `json:"snapshot_id,omitempty"`
	Confidence          string   `json:"confidence,omitempty"`
	Action              string   `json:"action,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	RuleHit             string   `json:"rule_hit,omitempty"`
	Direction           string   `json:"direction,omitempty"`
	ConsensusScore      *float64 `json:"consensus_score,omitempty"`
	ConsensusConfidence *float64 `json:"consensus_confidence,omitempty"`
}

type reflectionCompactEntrySignals struct {
	Agent    map[string]string `json:"agent,omitempty"`
	Provider map[string]string `json:"provider,omitempty"`
}

type reflectionCompactManagement struct {
	StopBasis  string   `json:"stop_basis,omitempty"`
	RiskEvents []string `json:"risk_events,omitempty"`
	LatestGate string   `json:"latest_gate,omitempty"`
}

type reflectionCompactExit struct {
	Role string  `json:"role,omitempty"`
	At   string  `json:"at,omitempty"`
	Side string  `json:"side,omitempty"`
	Avg  float64 `json:"avg,omitempty"`
	Qty  float64 `json:"qty,omitempty"`
	Tag  string  `json:"tag,omitempty"`
}

func buildReflectionCompactPrompt(input ReflectionInput) reflectionCompactPrompt {
	context := input.Context
	out := reflectionCompactPrompt{
		TradeResult: reflectionCompactTradeResult{
			Symbol:     context.TradeResult.Symbol,
			Direction:  context.TradeResult.Direction,
			Entry:      context.TradeResult.EntryPrice,
			FinalExit:  context.TradeResult.ExitPrice,
			PnLPercent: context.TradeResult.PnLPercent,
			Duration:   context.TradeResult.Duration,
		},
	}
	if context.ExitContext != nil {
		out.TradeResult.ClosedAt = context.ExitContext.ClosedAt
		out.TradeResult.ExitReason = context.ExitContext.ExitReason
		out.TradeResult.ExitOrderStatus = context.ExitContext.ExitOrderStatus
		out.Exits = buildReflectionCompactExits(context.ExitContext.CloseOrders)
	}
	if context.EntryContext != nil {
		out.EntryDecision = buildReflectionCompactEntryDecision(*context.EntryContext)
		out.EntrySignals = buildReflectionCompactEntrySignals(*context.EntryContext)
	}
	out.Management = buildReflectionCompactManagement(context.PositionEvolution, context.ExitContext)
	return out
}

func buildReflectionCompactEntryDecision(entry reflectionEntryContext) *reflectionCompactEntryDecision {
	out := &reflectionCompactEntryDecision{
		SnapshotID:          entry.SnapshotID,
		Confidence:          entry.Confidence,
		Action:              entry.DecisionAction,
		Reason:              compactReflectionText(entry.GateReason, reflectionCompactTextLimit),
		RuleHit:             compactReflectionText(entry.RuleHit, reflectionCompactTextLimit),
		Direction:           entry.Direction,
		ConsensusScore:      entry.ConsensusScore,
		ConsensusConfidence: entry.ConsensusConfidence,
	}
	if out.SnapshotID == 0 &&
		out.Confidence == "" &&
		out.Action == "" &&
		out.Reason == "" &&
		out.RuleHit == "" &&
		out.Direction == "" &&
		out.ConsensusScore == nil &&
		out.ConsensusConfidence == nil {
		return nil
	}
	return out
}

func buildReflectionCompactEntrySignals(entry reflectionEntryContext) *reflectionCompactEntrySignals {
	out := &reflectionCompactEntrySignals{
		Agent:    compactReflectionSummaries(entry.AgentSummaries),
		Provider: compactReflectionSummaries(entry.ProviderSummaries),
	}
	if len(out.Agent) == 0 && len(out.Provider) == 0 {
		return nil
	}
	return out
}

func compactReflectionSummaries(items []string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		role, summary, ok := strings.Cut(strings.TrimSpace(item), ": ")
		if !ok {
			role = fmt.Sprintf("signal_%d", len(out)+1)
			summary = item
		}
		role = compactReflectionRole(role)
		summary = compactReflectionText(summary, reflectionCompactTextLimit)
		if role == "" || summary == "" {
			continue
		}
		out[role] = summary
	}
	return out
}

func compactReflectionRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	role = strings.ReplaceAll(role, " ", "_")
	role = strings.ReplaceAll(role, "-", "_")
	return role
}

func buildReflectionCompactManagement(evolution *reflectionPositionEvolution, exit *reflectionExitContext) *reflectionCompactManagement {
	out := &reflectionCompactManagement{}
	if exit != nil {
		out.StopBasis = compactReflectionText(exit.StopReason, reflectionCompactTextLimit)
		if exit.LatestDecision != nil {
			out.LatestGate = compactReflectionGate(*exit.LatestDecision)
		}
	}
	if evolution != nil {
		out.RiskEvents = compactReflectionRiskEvents(evolution.RiskTimeline)
	}
	if out.StopBasis == "" && out.LatestGate == "" && len(out.RiskEvents) == 0 {
		return nil
	}
	return out
}

func compactReflectionRiskEvents(items []reflectionRiskTimelineItem) []string {
	if len(items) == 0 {
		return nil
	}
	events := make([]string, 0, len(items))
	for idx := len(items) - 1; idx >= 0; idx-- {
		item := items[idx]
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = strings.TrimSpace(item.Source)
		}
		if label == "" {
			label = "risk_update"
		}
		event := ""
		switch {
		case item.PreviousStopLoss > 0 && item.StopLoss > 0 && item.PreviousStopLoss != item.StopLoss:
			event = fmt.Sprintf("%s: stop %s -> %s", label, compactFloat(item.PreviousStopLoss), compactFloat(item.StopLoss))
		case item.StopLoss > 0:
			event = fmt.Sprintf("%s: stop %s", label, compactFloat(item.StopLoss))
		case len(item.TakeProfits) > 0:
			event = fmt.Sprintf("%s: take_profits %s", label, compactFloatSlice(item.TakeProfits))
		}
		if event == "" {
			continue
		}
		if item.CreatedAt != "" {
			event = item.CreatedAt + " " + event
		}
		events = append(events, event)
	}
	return selectCompactReflectionEvents(events, reflectionCompactRiskEventLimit)
}

func selectCompactReflectionEvents(events []string, limit int) []string {
	if limit <= 0 || len(events) <= limit {
		return events
	}
	out := make([]string, 0, limit)
	out = append(out, events[0])
	out = append(out, events[len(events)-(limit-1):]...)
	return out
}

func compactReflectionGate(gate reflectionGateEventSummary) string {
	parts := make([]string, 0, 4)
	if gate.Action != "" {
		parts = append(parts, gate.Action)
	}
	if gate.Reason != "" {
		parts = append(parts, compactReflectionText(gate.Reason, reflectionCompactTextLimit))
	}
	if gate.Direction != "" {
		parts = append(parts, "direction="+gate.Direction)
	}
	if gate.At != "" {
		parts = append(parts, "at="+gate.At)
	}
	return strings.Join(parts, ", ")
}

func buildReflectionCompactExits(orders []reflectionCloseOrder) []reflectionCompactExit {
	if len(orders) == 0 {
		return nil
	}
	out := make([]reflectionCompactExit, 0, len(orders))
	for idx, order := range orders {
		role := "partial_exit"
		if idx == len(orders)-1 {
			role = "final_exit"
		}
		at := order.FilledAt
		if at == "" {
			at = order.SubmittedAt
		}
		avg := order.Average
		if avg == 0 {
			avg = order.Price
		}
		qty := order.Filled
		if qty == 0 {
			qty = order.Amount
		}
		exit := reflectionCompactExit{
			Role: role,
			At:   at,
			Side: order.Side,
			Avg:  avg,
			Qty:  qty,
			Tag:  compactReflectionText(order.Tag, 64),
		}
		if exit.At == "" && exit.Side == "" && exit.Avg == 0 && exit.Qty == 0 && exit.Tag == "" {
			continue
		}
		out = append(out, exit)
	}
	return out
}

func compactReflectionText(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" || limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}

func compactFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func compactFloatSlice(items []float64) string {
	if len(items) == 0 {
		return "[]"
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, compactFloat(item))
	}
	return "[" + strings.Join(out, ",") + "]"
}
