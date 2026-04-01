package runtimeapi

import (
	"context"
	"strings"

	"brale-core/internal/position"
	"brale-core/internal/runtime"
	"brale-core/internal/store"
)

const dashboardDecisionFlowGateScanLimit = 200

type dashboardFlowUsecase struct {
	resolver    SymbolResolver
	store       dashboardFlowStore
	allowSymbol func(string) bool
	symbolCfgs  map[string]ConfigBundle
}

type dashboardFlowStore interface {
	store.TimelineQueryStore
	store.PositionQueryStore
	store.RiskPlanQueryStore
}

func newDashboardFlowUsecase(s *Server) dashboardFlowUsecase {
	if s == nil {
		return dashboardFlowUsecase{}
	}
	return dashboardFlowUsecase{
		resolver:    s.Resolver,
		store:       s.Store,
		allowSymbol: s.AllowSymbol,
		symbolCfgs:  s.SymbolConfigs,
	}
}

func (u dashboardFlowUsecase) build(ctx context.Context, rawSymbol string, snapshotQuery string) (DashboardDecisionFlowResponse, *usecaseError) {
	if u.store == nil {
		return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "store_missing", Message: "Store 未配置"}
	}

	normalizedSymbol := runtime.NormalizeSymbol(strings.TrimSpace(rawSymbol))
	if normalizedSymbol == "" || !isValidDashboardSymbol(normalizedSymbol) {
		return DashboardDecisionFlowResponse{}, &usecaseError{Status: 400, Code: "invalid_symbol", Message: "symbol 非法", Details: rawSymbol}
	}
	if u.allowSymbol != nil && !u.allowSymbol(normalizedSymbol) {
		return DashboardDecisionFlowResponse{}, &usecaseError{Status: 400, Code: "symbol_not_allowed", Message: "symbol 不在允许列表", Details: normalizedSymbol}
	}

	selectedSnapshotID, hasSelectedSnapshot, parseErr := parseDetailSnapshotQuery(snapshotQuery)
	if parseErr != nil {
		return DashboardDecisionFlowResponse{}, &usecaseError{Status: 400, Code: "invalid_snapshot_id", Message: "snapshot_id 非法", Details: parseErr.Error()}
	}

	var err error
	gates := []store.GateEventRecord{}
	if !hasSelectedSnapshot {
		gates, err = u.store.ListGateEvents(ctx, normalizedSymbol, dashboardDecisionFlowGateScanLimit)
		if err != nil {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "gate_events_failed", Message: "gate 事件读取失败", Details: err.Error()}
		}
	}

	pos, isOpen, err := u.store.FindPositionBySymbol(ctx, normalizedSymbol, position.OpenPositionStatuses)
	if err != nil {
		return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "position_lookup_failed", Message: "持仓查询失败", Details: err.Error()}
	}

	selection := dashboardFlowSelection{}
	if hasSelectedSnapshot {
		selectedGate, found, gateErr := u.store.FindGateEventBySnapshot(ctx, normalizedSymbol, selectedSnapshotID)
		if gateErr != nil {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "gate_events_failed", Message: "gate 事件读取失败", Details: gateErr.Error()}
		}
		if !found {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 404, Code: "snapshot_not_found", Message: "snapshot_id 对应决策不存在", Details: selectedSnapshotID}
		}
		selection = dashboardFlowSelection{
			Anchor: DashboardFlowAnchor{Type: "selected_round", SnapshotID: selectedSnapshotID, Confidence: "high", Reason: "selected_by_snapshot_id"},
			Gate:   &selectedGate,
		}
	} else {
		var ok bool
		selection, ok = selectDashboardFlowSelection(selectedSnapshotID, false, pos, isOpen, gates)
		if !ok {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 404, Code: "snapshot_not_found", Message: "snapshot_id 对应决策不存在", Details: selectedSnapshotID}
		}
	}
	anchor := selection.Anchor
	gateway := selection.Gate

	providers := []store.ProviderEventRecord{}
	agents := []store.AgentEventRecord{}
	if gateway != nil && gateway.SnapshotID > 0 {
		providers, err = u.store.ListProviderEventsBySnapshot(ctx, normalizedSymbol, gateway.SnapshotID)
		if err != nil {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "provider_events_failed", Message: "provider 事件读取失败", Details: err.Error()}
		}
		agents, err = u.store.ListAgentEventsBySnapshot(ctx, normalizedSymbol, gateway.SnapshotID)
		if err != nil {
			return DashboardDecisionFlowResponse{}, &usecaseError{Status: 500, Code: "agent_events_failed", Message: "agent 事件读取失败", Details: err.Error()}
		}
	}

	tighten := resolveDashboardTightenInfo(gateway)
	if tighten == nil && isOpen && !hasSelectedSnapshot {
		tighten = resolveDashboardTightenFromRiskHistory(ctx, u.store, pos)
	}

	preferInPositionProvider := shouldPreferInPositionProvider(isOpen, gateway)
	agentModels := u.resolveAgentStageModels(normalizedSymbol)
	stages := assembleDashboardFlowStageSet(providers, agents, preferInPositionProvider, agentModels)
	nodes := buildDashboardFlowNodes(stages, gateway, tighten)
	intervals := u.resolveSymbolIntervals(normalizedSymbol)
	trace := buildDashboardFlowTrace(stages, gateway, pos, isOpen)

	return DashboardDecisionFlowResponse{
		Status: "ok",
		Symbol: normalizedSymbol,
		Flow: DashboardDecisionFlow{
			Anchor:    anchor,
			Nodes:     nodes,
			Intervals: intervals,
			Trace:     trace,
			Tighten:   tighten,
		},
		Summary: dashboardContractSummary,
	}, nil
}

func (u dashboardFlowUsecase) resolveAgentStageModels(symbol string) map[string]string {
	if len(u.symbolCfgs) == 0 {
		return nil
	}
	bundle, ok := u.symbolCfgs[symbol]
	if !ok {
		return nil
	}
	models := map[string]string{
		"indicator": strings.TrimSpace(bundle.Symbol.LLM.Agent.Indicator.Model),
		"structure": strings.TrimSpace(bundle.Symbol.LLM.Agent.Structure.Model),
		"mechanics": strings.TrimSpace(bundle.Symbol.LLM.Agent.Mechanics.Model),
	}
	for stage, model := range models {
		if model == "" {
			delete(models, stage)
		}
	}
	if len(models) == 0 {
		return nil
	}
	return models
}

func (u dashboardFlowUsecase) resolveSymbolIntervals(symbol string) []string {
	if u.resolver == nil {
		return nil
	}
	resolved, err := u.resolver.Resolve(symbol)
	if err != nil {
		return nil
	}
	return normalizedIntervals(resolved.Intervals)
}
