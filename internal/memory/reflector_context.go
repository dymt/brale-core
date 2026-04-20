package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"brale-core/internal/decision/fund"
	"brale-core/internal/execution"
	"brale-core/internal/store"
)

const (
	reflectionRiskTimelineLimit = 5
	reflectionGateHistoryLimit  = 3
	reflectionGateScanLimit     = 200
)

type reflectionContextStore interface {
	store.TimelineQueryStore
	store.RiskPlanQueryStore
}

type reflectionPromptContext struct {
	TradeResult       reflectionTradeResult        `json:"trade_result"`
	EntryContext      *reflectionEntryContext      `json:"entry_context,omitempty"`
	PositionEvolution *reflectionPositionEvolution `json:"position_evolution,omitempty"`
	ExitContext       *reflectionExitContext       `json:"exit_context,omitempty"`
}

type reflectionTradeResult struct {
	Symbol     string `json:"symbol"`
	Direction  string `json:"direction"`
	EntryPrice string `json:"entry_price"`
	ExitPrice  string `json:"exit_price"`
	PnLPercent string `json:"pnl_pct"`
	Duration   string `json:"duration"`
}

type reflectionEntryContext struct {
	SnapshotID          uint     `json:"entry_snapshot_id,omitempty"`
	Confidence          string   `json:"confidence,omitempty"`
	ResolveReason       string   `json:"resolve_reason,omitempty"`
	DecisionAction      string   `json:"decision_action,omitempty"`
	GateReason          string   `json:"gate_reason,omitempty"`
	Grade               int      `json:"grade,omitempty"`
	Direction           string   `json:"direction,omitempty"`
	RuleHit             string   `json:"rule_hit,omitempty"`
	EnterTag            string   `json:"enter_tag,omitempty"`
	ConsensusScore      *float64 `json:"consensus_score,omitempty"`
	ConsensusConfidence *float64 `json:"consensus_confidence,omitempty"`
	ProviderSummaries   []string `json:"provider_summaries,omitempty"`
	AgentSummaries      []string `json:"agent_summaries,omitempty"`
}

type reflectionPositionEvolution struct {
	RiskTimeline  []reflectionRiskTimelineItem `json:"risk_timeline,omitempty"`
	KeyGateEvents []reflectionGateEventSummary `json:"key_gate_events,omitempty"`
}

type reflectionRiskTimelineItem struct {
	Source              string    `json:"source,omitempty"`
	Label               string    `json:"label,omitempty"`
	CreatedAt           string    `json:"created_at,omitempty"`
	StopLoss            float64   `json:"stop_loss,omitempty"`
	TakeProfits         []float64 `json:"take_profits,omitempty"`
	PreviousStopLoss    float64   `json:"previous_stop_loss,omitempty"`
	PreviousTakeProfits []float64 `json:"previous_take_profits,omitempty"`
}

type reflectionExitContext struct {
	ClosedAt        string                      `json:"closed_at,omitempty"`
	ExitReason      string                      `json:"exit_reason,omitempty"`
	ExitOrderStatus string                      `json:"exit_order_status,omitempty"`
	StopReason      string                      `json:"stop_reason,omitempty"`
	AbortReason     string                      `json:"abort_reason,omitempty"`
	CloseOrders     []reflectionCloseOrder      `json:"close_orders,omitempty"`
	LatestDecision  *reflectionGateEventSummary `json:"latest_decision,omitempty"`
}

type reflectionCloseOrder struct {
	OrderID     string  `json:"order_id,omitempty"`
	Status      string  `json:"status,omitempty"`
	Side        string  `json:"side,omitempty"`
	OrderType   string  `json:"order_type,omitempty"`
	Price       float64 `json:"price,omitempty"`
	Average     float64 `json:"average,omitempty"`
	Filled      float64 `json:"filled,omitempty"`
	Amount      float64 `json:"amount,omitempty"`
	Remaining   float64 `json:"remaining,omitempty"`
	Cost        float64 `json:"cost,omitempty"`
	SubmittedAt string  `json:"submitted_at,omitempty"`
	FilledAt    string  `json:"filled_at,omitempty"`
	Tag         string  `json:"tag,omitempty"`
}

type reflectionGateEventSummary struct {
	SnapshotID uint   `json:"snapshot_id,omitempty"`
	At         string `json:"at,omitempty"`
	Action     string `json:"action,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Direction  string `json:"direction,omitempty"`
	Grade      int    `json:"grade,omitempty"`
}

type reflectionAnchor struct {
	SnapshotID    uint
	Confidence    string
	ResolveReason string
}

type reflectionRiskTimelineRow struct {
	Source              string
	Label               string
	CreatedAt           string
	StopLoss            float64
	TakeProfits         []float64
	PreviousStopLoss    float64
	PreviousTakeProfits []float64
}

func buildReflectionContext(ctx context.Context, st reflectionContextStore, pos store.PositionRecord, trade execution.Trade, input ReflectionInput) reflectionPromptContext {
	context := reflectionPromptContext{
		TradeResult: reflectionTradeResult{
			Symbol:     input.Symbol,
			Direction:  input.Direction,
			EntryPrice: input.EntryPrice,
			ExitPrice:  input.ExitPrice,
			PnLPercent: input.PnLPercent,
			Duration:   input.Duration,
		},
		ExitContext: &reflectionExitContext{
			ClosedAt:        formatMillisTimestamp(int64(trade.CloseTimestamp)),
			ExitReason:      strings.TrimSpace(string(trade.ExitReason)),
			ExitOrderStatus: strings.TrimSpace(string(trade.ExitOrderStatus)),
			StopReason:      strings.TrimSpace(pos.StopReason),
			AbortReason:     strings.TrimSpace(pos.AbortReason),
			CloseOrders:     buildReflectionCloseOrders(trade),
		},
	}
	if st == nil {
		return context
	}

	symbol := strings.TrimSpace(pos.Symbol)
	if symbol == "" {
		symbol = input.Symbol
	}
	if symbol == "" {
		return context
	}

	gates, err := st.ListGateEvents(ctx, symbol, reflectionGateScanLimit)
	if err == nil && len(gates) > 0 {
		if anchor, ok := resolveReflectionOpeningAnchor(pos, gates); ok {
			if entry := buildReflectionEntryContext(ctx, st, symbol, trade, anchor); entry != nil {
				context.EntryContext = entry
			}
		}
	}

	entrySnapshotID := uint(0)
	if context.EntryContext != nil {
		entrySnapshotID = context.EntryContext.SnapshotID
	}
	if evolution, latest := buildReflectionPositionEvolution(ctx, st, pos, trade, entrySnapshotID); latest != nil && context.ExitContext != nil {
		context.ExitContext.LatestDecision = latest
		if evolution != nil {
			context.PositionEvolution = evolution
		}
	} else if evolution != nil {
		context.PositionEvolution = evolution
	}

	if context.ExitContext != nil && isEmptyReflectionExitContext(*context.ExitContext) {
		context.ExitContext = nil
	}
	return context
}

func buildReflectionEntryContext(ctx context.Context, st reflectionContextStore, symbol string, trade execution.Trade, anchor reflectionAnchor) *reflectionEntryContext {
	if st == nil || anchor.SnapshotID == 0 {
		return nil
	}

	entry := &reflectionEntryContext{
		SnapshotID:    anchor.SnapshotID,
		Confidence:    strings.TrimSpace(anchor.Confidence),
		ResolveReason: strings.TrimSpace(anchor.ResolveReason),
		EnterTag:      strings.TrimSpace(string(trade.EnterTag)),
	}

	gate, ok, err := st.FindGateEventBySnapshot(ctx, symbol, anchor.SnapshotID)
	if err == nil && ok {
		action := strings.ToUpper(strings.TrimSpace(gate.DecisionAction))
		entry.DecisionAction = action
		entry.Grade = gate.Grade
		entry.Direction = strings.ToLower(strings.TrimSpace(gate.Direction))
		entry.GateReason = reflectionDisplayReason(action, strings.TrimSpace(gate.GateReason), gate.DerivedJSON)
		entry.RuleHit = parseReflectionRuleHit(gate.RuleHitJSON)
		entry.ConsensusScore, entry.ConsensusConfidence = parseReflectionConsensus(gate.DerivedJSON)
	}

	providers, err := st.ListProviderEventsBySnapshot(ctx, symbol, anchor.SnapshotID)
	if err == nil {
		entry.ProviderSummaries = buildReflectionProviderSummaries(providers)
	}
	agents, err := st.ListAgentEventsBySnapshot(ctx, symbol, anchor.SnapshotID)
	if err == nil {
		entry.AgentSummaries = buildReflectionAgentSummaries(agents)
	}

	if isEmptyReflectionEntryContext(*entry) {
		return nil
	}
	return entry
}

func buildReflectionPositionEvolution(ctx context.Context, st reflectionContextStore, pos store.PositionRecord, trade execution.Trade, entrySnapshotID uint) (*reflectionPositionEvolution, *reflectionGateEventSummary) {
	if st == nil {
		return nil, nil
	}

	evolution := &reflectionPositionEvolution{}
	timeline := buildReflectionRiskTimeline(ctx, st, pos)
	if len(timeline) > 0 {
		evolution.RiskTimeline = make([]reflectionRiskTimelineItem, 0, len(timeline))
		for _, item := range timeline {
			evolution.RiskTimeline = append(evolution.RiskTimeline, reflectionRiskTimelineItem{
				Source:              strings.TrimSpace(item.Source),
				Label:               strings.TrimSpace(item.Label),
				CreatedAt:           strings.TrimSpace(item.CreatedAt),
				StopLoss:            item.StopLoss,
				TakeProfits:         append([]float64(nil), item.TakeProfits...),
				PreviousStopLoss:    item.PreviousStopLoss,
				PreviousTakeProfits: append([]float64(nil), item.PreviousTakeProfits...),
			})
		}
	}

	gates := loadReflectionGateWindow(ctx, st, pos, trade)
	latestGate := latestReflectionGateSummary(gates, entrySnapshotID)
	keyEvents := selectReflectionKeyGateEvents(gates, entrySnapshotID)
	if len(keyEvents) > 0 {
		evolution.KeyGateEvents = keyEvents
	}

	if len(evolution.RiskTimeline) == 0 && len(evolution.KeyGateEvents) == 0 {
		return nil, latestGate
	}
	return evolution, latestGate
}

func buildReflectionCloseOrders(trade execution.Trade) []reflectionCloseOrder {
	if len(trade.Orders) == 0 {
		return nil
	}
	orders := make([]execution.TradeOrder, 0, len(trade.Orders))
	for _, order := range trade.Orders {
		if order.FTIsEntry {
			continue
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		return nil
	}
	sort.SliceStable(orders, func(i, j int) bool {
		return reflectionOrderTimestamp(orders[i]) < reflectionOrderTimestamp(orders[j])
	})
	out := make([]reflectionCloseOrder, 0, len(orders))
	for _, order := range orders {
		item := reflectionCloseOrder{
			OrderID:     strings.TrimSpace(order.OrderID),
			Status:      strings.TrimSpace(string(order.Status)),
			Side:        strings.TrimSpace(string(order.FTOrderSide)),
			OrderType:   strings.TrimSpace(string(order.OrderType)),
			Price:       float64(order.Price),
			Average:     float64(order.Average),
			Filled:      float64(order.Filled),
			Amount:      float64(order.Amount),
			Remaining:   float64(order.Remaining),
			Cost:        float64(order.OrderCost),
			SubmittedAt: formatMillisTimestamp(int64(order.OrderTimestamp)),
			FilledAt:    formatMillisTimestamp(int64(order.OrderFilledAt)),
			Tag:         strings.TrimSpace(string(order.FTOrderTag)),
		}
		if isEmptyReflectionCloseOrder(item) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func reflectionOrderTimestamp(order execution.TradeOrder) int64 {
	if order.OrderFilledAt > 0 {
		return int64(order.OrderFilledAt)
	}
	return int64(order.OrderTimestamp)
}

func loadReflectionGateWindow(ctx context.Context, st reflectionContextStore, pos store.PositionRecord, trade execution.Trade) []store.GateEventRecord {
	if st == nil {
		return nil
	}
	symbol := strings.TrimSpace(pos.Symbol)
	if symbol == "" {
		return nil
	}
	start, end, ok := reflectionGateRange(pos, trade)
	if !ok {
		return nil
	}
	rows, err := st.ListGateEventsByTimeRange(ctx, symbol, start, end)
	if err != nil {
		return nil
	}
	return rows
}

func reflectionGateRange(pos store.PositionRecord, trade execution.Trade) (int64, int64, bool) {
	start := int64(trade.OpenFillTimestamp) / 1000
	if start <= 0 && !pos.CreatedAt.IsZero() {
		start = pos.CreatedAt.Unix()
	}
	end := int64(trade.CloseTimestamp) / 1000
	if end <= 0 && !pos.UpdatedAt.IsZero() {
		end = pos.UpdatedAt.Unix()
	}
	if start <= 0 || end <= 0 || end < start {
		return 0, 0, false
	}
	return start, end, true
}

func latestReflectionGateSummary(gates []store.GateEventRecord, entrySnapshotID uint) *reflectionGateEventSummary {
	for idx := len(gates) - 1; idx >= 0; idx-- {
		if gates[idx].SnapshotID == 0 || gates[idx].SnapshotID == entrySnapshotID {
			continue
		}
		summary := summarizeReflectionGateEvent(gates[idx])
		if isEmptyReflectionGateSummary(summary) {
			continue
		}
		return &summary
	}
	return nil
}

func selectReflectionKeyGateEvents(gates []store.GateEventRecord, entrySnapshotID uint) []reflectionGateEventSummary {
	if len(gates) == 0 {
		return nil
	}
	selected := make([]reflectionGateEventSummary, 0, reflectionGateHistoryLimit)
	for _, gate := range gates {
		if gate.SnapshotID == 0 || gate.SnapshotID == entrySnapshotID {
			continue
		}
		summary := summarizeReflectionGateEvent(gate)
		if isEmptyReflectionGateSummary(summary) {
			continue
		}
		if strings.EqualFold(summary.Action, "ALLOW") {
			continue
		}
		selected = append(selected, summary)
	}
	if len(selected) == 0 {
		if latest := latestReflectionGateSummary(gates, entrySnapshotID); latest != nil {
			return []reflectionGateEventSummary{*latest}
		}
		return nil
	}
	if len(selected) > reflectionGateHistoryLimit {
		selected = selected[len(selected)-reflectionGateHistoryLimit:]
	}
	return selected
}

func summarizeReflectionGateEvent(gate store.GateEventRecord) reflectionGateEventSummary {
	action := strings.ToUpper(strings.TrimSpace(gate.DecisionAction))
	return reflectionGateEventSummary{
		SnapshotID: gate.SnapshotID,
		At:         formatGateTimestamp(gate),
		Action:     action,
		Reason:     reflectionDisplayReason(action, strings.TrimSpace(gate.GateReason), gate.DerivedJSON),
		Direction:  strings.ToLower(strings.TrimSpace(gate.Direction)),
		Grade:      gate.Grade,
	}
}

func resolveReflectionOpeningAnchor(pos store.PositionRecord, gates []store.GateEventRecord) (reflectionAnchor, bool) {
	snapshotID, ok := resolveReflectionOpeningSnapshotID(pos, gates)
	if !ok {
		return reflectionAnchor{}, false
	}
	confidence := "medium"
	reason := "matched_by_position_timeline"
	if reflectionSnapshotFromOpenIntentID(pos.OpenIntentID) == snapshotID {
		confidence = "high"
		reason = "matched_by_open_intent_id"
	}
	return reflectionAnchor{
		SnapshotID:    snapshotID,
		Confidence:    confidence,
		ResolveReason: reason,
	}, true
}

func resolveReflectionOpeningSnapshotID(pos store.PositionRecord, gates []store.GateEventRecord) (uint, bool) {
	if len(gates) == 0 {
		return 0, false
	}
	if openIntentSnapshot := reflectionSnapshotFromOpenIntentID(pos.OpenIntentID); openIntentSnapshot > 0 {
		for _, gate := range gates {
			if gate.SnapshotID == openIntentSnapshot {
				return gate.SnapshotID, true
			}
		}
	}
	anchorTimestamp := int64(0)
	if !pos.CreatedAt.IsZero() {
		anchorTimestamp = pos.CreatedAt.Unix()
	}
	if anchorTimestamp <= 0 && !pos.UpdatedAt.IsZero() {
		anchorTimestamp = pos.UpdatedAt.Unix()
	}
	bestSnapshot := uint(0)
	bestTimestamp := int64(0)
	for _, gate := range gates {
		if gate.SnapshotID == 0 || gate.Timestamp <= 0 {
			continue
		}
		if anchorTimestamp > 0 {
			if gate.Timestamp > anchorTimestamp {
				continue
			}
			if gate.Timestamp >= bestTimestamp {
				bestTimestamp = gate.Timestamp
				bestSnapshot = gate.SnapshotID
			}
			continue
		}
		if bestSnapshot == 0 || gate.Timestamp < bestTimestamp {
			bestTimestamp = gate.Timestamp
			bestSnapshot = gate.SnapshotID
		}
	}
	if bestSnapshot > 0 {
		return bestSnapshot, true
	}
	return 0, false
}

func reflectionSnapshotFromOpenIntentID(openIntentID string) uint {
	raw := strings.TrimSpace(openIntentID)
	if raw == "" {
		return 0
	}
	tokens := strings.FieldsFunc(raw, func(r rune) bool {
		return r < '0' || r > '9'
	})
	for _, token := range tokens {
		if len(token) < 9 {
			continue
		}
		parsed, err := strconv.ParseUint(token, 10, 64)
		if err == nil && parsed > 0 {
			return uint(parsed)
		}
	}
	return 0
}

func buildReflectionProviderSummaries(records []store.ProviderEventRecord) []string {
	if len(records) == 0 {
		return nil
	}
	sorted := append([]store.ProviderEventRecord(nil), records...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Role < sorted[j].Role
	})
	out := make([]string, 0, len(sorted))
	for _, item := range sorted {
		if summary := buildReflectionOutputSummary(strings.TrimSpace(item.Role), item.OutputJSON); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

func buildReflectionAgentSummaries(records []store.AgentEventRecord) []string {
	if len(records) == 0 {
		return nil
	}
	sorted := append([]store.AgentEventRecord(nil), records...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Stage < sorted[j].Stage
	})
	out := make([]string, 0, len(sorted))
	for _, item := range sorted {
		if summary := buildReflectionOutputSummary(strings.TrimSpace(item.Stage), item.OutputJSON); summary != "" {
			out = append(out, summary)
		}
	}
	return out
}

func buildReflectionOutputSummary(role string, raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload) == 0 {
		return ""
	}
	preferred := []string{
		"signal_tag",
		"monitor_tag",
		"regime",
		"last_break",
		"quality",
		"risk_level",
		"crowding",
		"alignment",
		"movement_score",
		"movement_confidence",
		"momentum_expansion",
		"clear_structure",
		"integrity",
		"threat_level",
		"momentum_sustaining",
		"divergence_detected",
		"mean_rev_noise",
		"reason",
	}
	parts := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, key := range preferred {
		if value, ok := payload[key]; ok {
			if formatted := formatReflectionValue(key, value); formatted != "" {
				parts = append(parts, formatted)
				seen[key] = struct{}{}
			}
		}
		if len(parts) == 3 {
			break
		}
	}
	if len(parts) < 3 {
		keys := make([]string, 0, len(payload))
		for key := range payload {
			if _, ok := seen[key]; ok {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if formatted := formatReflectionValue(key, payload[key]); formatted != "" {
				parts = append(parts, formatted)
			}
			if len(parts) == 3 {
				break
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return role + ": " + strings.Join(parts, ", ")
}

func formatReflectionValue(key string, value any) string {
	switch raw := value.(type) {
	case nil:
		return ""
	case string:
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return ""
		}
		return key + "=" + compactReflectionText(raw, 96)
	case bool:
		return fmt.Sprintf("%s=%t", key, raw)
	case float64:
		return fmt.Sprintf("%s=%.4g", key, raw)
	case map[string]any:
		parts := make([]string, 0, 2)
		for _, nested := range []string{"value", "confidence", "reason"} {
			if item, ok := raw[nested]; ok {
				if formatted := formatReflectionValue(nested, item); formatted != "" {
					parts = append(parts, formatted)
				}
			}
			if len(parts) == 2 {
				break
			}
		}
		if len(parts) == 0 {
			return ""
		}
		return key + ".{" + strings.Join(parts, ", ") + "}"
	default:
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text == "" || text == "<nil>" {
			return ""
		}
		return key + "=" + compactReflectionText(text, 96)
	}
}

func buildReflectionRiskTimeline(ctx context.Context, st reflectionContextStore, pos store.PositionRecord) []reflectionRiskTimelineRow {
	if st == nil || strings.TrimSpace(pos.PositionID) == "" {
		return nil
	}
	rows, err := st.ListRiskPlanHistory(ctx, pos.PositionID, reflectionRiskTimelineLimit)
	if err != nil || len(rows) == 0 {
		return nil
	}
	type decodedHistory struct {
		row         store.RiskPlanHistoryRecord
		stopLoss    float64
		takeProfits []float64
	}
	decoded := make([]decodedHistory, 0, len(rows))
	for _, row := range rows {
		stopLoss, takeProfits, ok := decodeReflectionRiskLevels(row.PayloadJSON)
		if !ok {
			continue
		}
		decoded = append(decoded, decodedHistory{
			row:         row,
			stopLoss:    stopLoss,
			takeProfits: takeProfits,
		})
	}
	out := make([]reflectionRiskTimelineRow, 0, len(decoded))
	for idx, item := range decoded {
		prevStop := item.stopLoss
		prevTPs := append([]float64(nil), item.takeProfits...)
		if idx+1 < len(decoded) {
			prevStop = decoded[idx+1].stopLoss
			prevTPs = append([]float64(nil), decoded[idx+1].takeProfits...)
		}
		createdAt := ""
		if !item.row.CreatedAt.IsZero() {
			createdAt = item.row.CreatedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, reflectionRiskTimelineRow{
			Source:              strings.TrimSpace(item.row.Source),
			Label:               reflectionRiskPlanLabel(strings.TrimSpace(item.row.Source), item.row.Version),
			CreatedAt:           createdAt,
			StopLoss:            item.stopLoss,
			TakeProfits:         append([]float64(nil), item.takeProfits...),
			PreviousStopLoss:    prevStop,
			PreviousTakeProfits: prevTPs,
		})
	}
	return out
}

func decodeReflectionRiskLevels(raw json.RawMessage) (float64, []float64, bool) {
	if len(raw) == 0 {
		return 0, nil, false
	}
	var payload struct {
		StopPrice float64 `json:"stop_price"`
		TPLevels  []struct {
			Price float64 `json:"price"`
		} `json:"tp_levels"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, nil, false
	}
	takeProfits := make([]float64, 0, len(payload.TPLevels))
	for _, level := range payload.TPLevels {
		takeProfits = append(takeProfits, level.Price)
	}
	return payload.StopPrice, takeProfits, payload.StopPrice > 0 || len(takeProfits) > 0
}

func reflectionRiskPlanLabel(source string, version int) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "monitor-tighten":
		return "收紧止损"
	case "monitor-breakeven":
		return "推保护本"
	case "entry-fill", "open-fill", "init", "init_from_plan":
		return "初始计划"
	default:
		if version > 0 {
			return fmt.Sprintf("计划 v%d", version)
		}
		return source
	}
}

func parseReflectionConsensus(raw json.RawMessage) (*float64, *float64) {
	if len(raw) == 0 {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil
	}
	consensusRaw, ok := payload["direction_consensus"].(map[string]any)
	if !ok {
		return nil, nil
	}
	return parseReflectionFloat(consensusRaw["score"]), parseReflectionFloat(consensusRaw["confidence"])
}

func reflectionDisplayReason(action string, fallback string, raw json.RawMessage) string {
	if !strings.EqualFold(strings.TrimSpace(action), "TIGHTEN") {
		return strings.TrimSpace(fallback)
	}
	if reason := reflectionTightenReason(raw); reason != "" {
		return reason
	}
	return strings.TrimSpace(fallback)
}

func reflectionTightenReason(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	executionRaw, ok := payload["execution"].(map[string]any)
	if !ok {
		return ""
	}
	action := strings.TrimSpace(fmt.Sprint(executionRaw["action"]))
	if !strings.EqualFold(action, "tighten") {
		return ""
	}
	executed := parseReflectionBool(executionRaw["executed"])
	tpTightened := parseReflectionBool(executionRaw["tp_tightened"])
	if executed {
		if tpTightened {
			return "已执行收紧，并同步收紧止盈"
		}
		return "已执行持仓收紧"
	}
	if blockedBy, ok := executionRaw["blocked_by"].([]any); ok && len(blockedBy) > 0 {
		first := strings.TrimSpace(fmt.Sprint(blockedBy[0]))
		if first != "" && first != "<nil>" {
			return "收紧未执行: " + first
		}
	}
	if parseReflectionBool(executionRaw["eligible"]) {
		return "满足收紧条件，等待执行"
	}
	if parseReflectionBool(executionRaw["evaluated"]) {
		return "已评估持仓收紧，但未触发"
	}
	return ""
}

func parseReflectionFloat(value any) *float64 {
	switch raw := value.(type) {
	case float64:
		return &raw
	case int:
		v := float64(raw)
		return &v
	case int64:
		v := float64(raw)
		return &v
	default:
		return nil
	}
}

func parseReflectionBool(value any) bool {
	switch raw := value.(type) {
	case bool:
		return raw
	case string:
		return strings.EqualFold(strings.TrimSpace(raw), "true")
	default:
		return false
	}
}

func parseReflectionRuleHit(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var hit fund.GateRuleHit
	if err := json.Unmarshal(raw, &hit); err != nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if name := strings.TrimSpace(hit.Name); name != "" {
		parts = append(parts, name)
	}
	if reason := strings.TrimSpace(hit.Reason); reason != "" && !strings.EqualFold(reason, hit.Name) {
		parts = append(parts, "reason="+reason)
	}
	if direction := strings.TrimSpace(hit.Direction); direction != "" {
		parts = append(parts, "direction="+strings.ToLower(direction))
	}
	return strings.Join(parts, ", ")
}

func formatGateTimestamp(gate store.GateEventRecord) string {
	if !gate.CreatedAt.IsZero() {
		return gate.CreatedAt.UTC().Format(time.RFC3339)
	}
	if gate.Timestamp > 0 {
		return time.Unix(gate.Timestamp, 0).UTC().Format(time.RFC3339)
	}
	return ""
}

func formatMillisTimestamp(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.UnixMilli(ts).UTC().Format(time.RFC3339)
}

func isEmptyReflectionEntryContext(entry reflectionEntryContext) bool {
	return entry.SnapshotID == 0 &&
		entry.Confidence == "" &&
		entry.ResolveReason == "" &&
		entry.DecisionAction == "" &&
		entry.GateReason == "" &&
		entry.Grade == 0 &&
		entry.Direction == "" &&
		entry.RuleHit == "" &&
		entry.EnterTag == "" &&
		entry.ConsensusScore == nil &&
		entry.ConsensusConfidence == nil &&
		len(entry.ProviderSummaries) == 0 &&
		len(entry.AgentSummaries) == 0
}

func isEmptyReflectionExitContext(exit reflectionExitContext) bool {
	return exit.ClosedAt == "" &&
		exit.ExitReason == "" &&
		exit.ExitOrderStatus == "" &&
		exit.StopReason == "" &&
		exit.AbortReason == "" &&
		len(exit.CloseOrders) == 0 &&
		exit.LatestDecision == nil
}

func isEmptyReflectionCloseOrder(order reflectionCloseOrder) bool {
	return order.OrderID == "" &&
		order.Status == "" &&
		order.Side == "" &&
		order.OrderType == "" &&
		order.Price == 0 &&
		order.Average == 0 &&
		order.Filled == 0 &&
		order.Amount == 0 &&
		order.Remaining == 0 &&
		order.Cost == 0 &&
		order.SubmittedAt == "" &&
		order.FilledAt == "" &&
		order.Tag == ""
}

func isEmptyReflectionGateSummary(summary reflectionGateEventSummary) bool {
	return summary.SnapshotID == 0 &&
		summary.At == "" &&
		summary.Action == "" &&
		summary.Reason == "" &&
		summary.Direction == "" &&
		summary.Grade == 0
}
