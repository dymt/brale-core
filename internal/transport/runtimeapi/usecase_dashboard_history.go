package runtimeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"brale-core/internal/decision/decisionfmt"
	"brale-core/internal/pkg/parseutil"
	"brale-core/internal/runtime"
	"brale-core/internal/store"
)

type dashboardHistoryUsecase struct {
	store       store.Store
	allowSymbol func(string) bool
}

func newDashboardHistoryUsecase(s *Server) dashboardHistoryUsecase {
	if s == nil {
		return dashboardHistoryUsecase{}
	}
	return dashboardHistoryUsecase{store: s.Store, allowSymbol: s.AllowSymbol}
}

func (u dashboardHistoryUsecase) build(ctx context.Context, rawSymbol string, limit int, snapshotQuery string) (DashboardDecisionHistoryResponse, *usecaseError) {
	if u.store == nil {
		return DashboardDecisionHistoryResponse{}, &usecaseError{Status: 500, Code: "store_missing", Message: "Store 未配置"}
	}

	normalizedSymbol := runtime.NormalizeSymbol(strings.TrimSpace(rawSymbol))
	if normalizedSymbol == "" || !isValidDashboardSymbol(normalizedSymbol) {
		return DashboardDecisionHistoryResponse{}, &usecaseError{Status: 400, Code: "invalid_symbol", Message: "symbol 非法", Details: rawSymbol}
	}
	if u.allowSymbol != nil && !u.allowSymbol(normalizedSymbol) {
		return DashboardDecisionHistoryResponse{}, &usecaseError{Status: 400, Code: "symbol_not_allowed", Message: "symbol 不在允许列表", Details: normalizedSymbol}
	}

	gates, err := u.store.ListGateEvents(ctx, normalizedSymbol, limit)
	if err != nil {
		return DashboardDecisionHistoryResponse{}, &usecaseError{Status: 500, Code: "gate_events_failed", Message: "gate 事件读取失败", Details: err.Error()}
	}

	items := mapHistoryItems(gates)
	response := DashboardDecisionHistoryResponse{
		Status:  "ok",
		Symbol:  normalizedSymbol,
		Limit:   limit,
		Items:   items,
		Summary: dashboardContractSummary,
	}
	if len(items) == 0 {
		response.Message = "no_history_available"
	} else {
		response.Message = fmt.Sprintf("history_rows=%d", len(items))
	}

	detailSnapshotID, hasDetail, parseErr := parseDetailSnapshotQuery(snapshotQuery)
	if parseErr != nil {
		return DashboardDecisionHistoryResponse{}, &usecaseError{Status: 400, Code: "invalid_snapshot_id", Message: "snapshot_id 非法", Details: parseErr.Error()}
	}
	if hasDetail {
		detail, detailErr := buildDecisionDetail(ctx, u.store, normalizedSymbol, detailSnapshotID)
		if detailErr != nil {
			return DashboardDecisionHistoryResponse{}, detailErr
		}
		response.Detail = detail
	}

	return response, nil
}

func parseDetailSnapshotQuery(raw string) (uint, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, false, fmt.Errorf("snapshot_id must be positive integer")
	}
	return uint(parsed), true, nil
}

func mapHistoryItems(gates []store.GateEventRecord) []DashboardDecisionHistoryItem {
	if len(gates) == 0 {
		return []DashboardDecisionHistoryItem{}
	}
	out := make([]DashboardDecisionHistoryItem, 0, len(gates))
	for _, gate := range gates {
		at := ""
		if gate.Timestamp > 0 {
			at = time.Unix(gate.Timestamp, 0).UTC().Format(time.RFC3339)
		}
		consensus := extractConsensusMetrics(json.RawMessage(gate.DerivedJSON))
		out = append(out, DashboardDecisionHistoryItem{
			SnapshotID:          gate.SnapshotID,
			Action:              strings.ToUpper(strings.TrimSpace(gate.DecisionAction)),
			Reason:              strings.TrimSpace(gate.GateReason),
			At:                  at,
			ConsensusScore:      consensus.Score,
			ConsensusConfidence: consensus.Confidence,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].At > out[j].At
	})
	return out
}

func buildDecisionDetail(ctx context.Context, st store.Store, symbol string, snapshotID uint) (*DashboardDecisionDetail, *usecaseError) {
	if snapshotID == 0 {
		return nil, &usecaseError{Status: 400, Code: "invalid_snapshot_id", Message: "snapshot_id 非法"}
	}
	gates, err := st.ListGateEvents(ctx, symbol, dashboardDecisionFlowGateScanLimit)
	if err != nil {
		return nil, &usecaseError{Status: 500, Code: "gate_events_failed", Message: "gate 事件读取失败", Details: err.Error()}
	}
	var selected *store.GateEventRecord
	for idx := range gates {
		if gates[idx].SnapshotID == snapshotID {
			selected = &gates[idx]
			break
		}
	}
	if selected == nil {
		return nil, &usecaseError{Status: 404, Code: "snapshot_not_found", Message: "snapshot_id 对应决策不存在", Details: snapshotID}
	}

	providers, err := st.ListProviderEventsBySnapshot(ctx, symbol, snapshotID)
	if err != nil {
		return nil, &usecaseError{Status: 500, Code: "provider_events_failed", Message: "provider 事件读取失败", Details: err.Error()}
	}
	agents, err := st.ListAgentEventsBySnapshot(ctx, symbol, snapshotID)
	if err != nil {
		return nil, &usecaseError{Status: 500, Code: "agent_events_failed", Message: "agent 事件读取失败", Details: err.Error()}
	}

	formatter := decisionfmt.New()
	report, reportErr := formatter.BuildDecisionReport(decisionfmt.DecisionInput{
		Symbol:     symbol,
		SnapshotID: snapshotID,
		Gate: decisionfmt.GateEvent{
			ID:               selected.ID,
			SnapshotID:       selected.SnapshotID,
			GlobalTradeable:  selected.GlobalTradeable,
			DecisionAction:   selected.DecisionAction,
			Grade:            selected.Grade,
			GateReason:       selected.GateReason,
			Direction:        selected.Direction,
			ProviderRefsJSON: json.RawMessage(selected.ProviderRefsJSON),
			RuleHitJSON:      json.RawMessage(selected.RuleHitJSON),
			DerivedJSON:      json.RawMessage(selected.DerivedJSON),
		},
		Providers: mapDecisionProviders(providers),
		Agents:    mapDecisionAgents(agents),
	})
	if reportErr != nil {
		return nil, &usecaseError{Status: 500, Code: "decision_build_failed", Message: "决策详情解析失败", Details: reportErr.Error()}
	}

	providerSummaries := make([]string, 0, len(report.Providers))
	for _, stage := range report.Providers {
		providerSummaries = append(providerSummaries, strings.TrimSpace(stage.Role+": "+stage.Summary))
	}
	agentSummaries := make([]string, 0, len(report.Agents))
	for _, stage := range report.Agents {
		agentSummaries = append(agentSummaries, strings.TrimSpace(stage.Role+": "+stage.Summary))
	}
	consensus := extractConsensusMetrics(json.RawMessage(selected.DerivedJSON))

	detail := &DashboardDecisionDetail{
		SnapshotID:                   selected.SnapshotID,
		Action:                       strings.ToUpper(strings.TrimSpace(selected.DecisionAction)),
		Reason:                       strings.TrimSpace(selected.GateReason),
		Tradeable:                    selected.GlobalTradeable,
		ConsensusScore:               consensus.Score,
		ConsensusConfidence:          consensus.Confidence,
		ConsensusScoreThreshold:      consensus.ScoreThreshold,
		ConsensusConfidenceThreshold: consensus.ConfidenceThreshold,
		ConsensusScorePassed:         consensus.ScorePassed,
		ConsensusConfidencePassed:    consensus.ConfidencePassed,
		ConsensusPassed:              consensus.Passed,
		Providers:                    providerSummaries,
		Agents:                       agentSummaries,
		ReportMarkdown:               prependDecisionHeader("🚦 决策报告", formatter.RenderDecisionMarkdown(report)),
		DecisionViewURL:              fmt.Sprintf("/decision-view/?symbol=%s&snapshot_id=%d", symbol, snapshotID),
	}
	return detail, nil
}

type dashboardConsensusMetrics struct {
	Score               *float64
	Confidence          *float64
	ScoreThreshold      *float64
	ConfidenceThreshold *float64
	ScorePassed         *bool
	ConfidencePassed    *bool
	Passed              *bool
}

func extractConsensusMetrics(raw json.RawMessage) dashboardConsensusMetrics {
	if len(raw) == 0 {
		return dashboardConsensusMetrics{}
	}
	var derived map[string]any
	if err := json.Unmarshal(raw, &derived); err != nil {
		return dashboardConsensusMetrics{}
	}
	consensusRaw, ok := derived["direction_consensus"]
	if !ok {
		return dashboardConsensusMetrics{}
	}
	consensus, ok := consensusRaw.(map[string]any)
	if !ok || len(consensus) == 0 {
		return dashboardConsensusMetrics{}
	}
	out := dashboardConsensusMetrics{}
	if score, ok := parseutil.FloatOK(consensus["score"]); ok {
		out.Score = &score
	}
	if confidence, ok := parseutil.FloatOK(consensus["confidence"]); ok {
		out.Confidence = &confidence
	}
	if scoreThreshold, ok := parseutil.FloatOK(consensus["score_threshold"]); ok {
		out.ScoreThreshold = &scoreThreshold
	}
	if confidenceThreshold, ok := parseutil.FloatOK(consensus["confidence_threshold"]); ok {
		out.ConfidenceThreshold = &confidenceThreshold
	}
	if scorePassed, ok := parseConsensusBool(consensus["score_passed"]); ok {
		out.ScorePassed = boolPtr(scorePassed)
	} else if out.Score != nil && out.ScoreThreshold != nil {
		out.ScorePassed = boolPtr(absFloat(*out.Score) >= *out.ScoreThreshold)
	}
	if confidencePassed, ok := parseConsensusBool(consensus["confidence_passed"]); ok {
		out.ConfidencePassed = boolPtr(confidencePassed)
	} else if out.Confidence != nil && out.ConfidenceThreshold != nil {
		out.ConfidencePassed = boolPtr(*out.Confidence >= *out.ConfidenceThreshold)
	}
	if passed, ok := parseConsensusBool(consensus["passed"]); ok {
		out.Passed = boolPtr(passed)
	} else if out.ScorePassed != nil && out.ConfidencePassed != nil {
		out.Passed = boolPtr(*out.ScorePassed && *out.ConfidencePassed)
	}
	return out
}

func parseConsensusBool(value any) (bool, bool) {
	switch raw := value.(type) {
	case bool:
		return raw, true
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(raw))
		if trimmed == "true" {
			return true, true
		}
		if trimmed == "false" {
			return false, true
		}
		return false, false
	case float64:
		return raw != 0, true
	case float32:
		return raw != 0, true
	case int:
		return raw != 0, true
	case int64:
		return raw != 0, true
	case uint64:
		return raw != 0, true
	default:
		return false, false
	}
}

func boolPtr(value bool) *bool {
	out := value
	return &out
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func mapDecisionProviders(records []store.ProviderEventRecord) []decisionfmt.ProviderEvent {
	out := make([]decisionfmt.ProviderEvent, 0, len(records))
	for _, rec := range records {
		out = append(out, decisionfmt.ProviderEvent{
			SnapshotID: rec.SnapshotID,
			OutputJSON: json.RawMessage(rec.OutputJSON),
			Role:       rec.Role,
		})
	}
	return out
}

func mapDecisionAgents(records []store.AgentEventRecord) []decisionfmt.AgentEvent {
	out := make([]decisionfmt.AgentEvent, 0, len(records))
	for _, rec := range records {
		out = append(out, decisionfmt.AgentEvent{
			SnapshotID: rec.SnapshotID,
			OutputJSON: json.RawMessage(rec.OutputJSON),
			Stage:      rec.Stage,
		})
	}
	return out
}
