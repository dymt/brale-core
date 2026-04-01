package runtimeapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"brale-core/internal/execution"
	"brale-core/internal/position"
	readmodel "brale-core/internal/readmodel/dashboard"
	"brale-core/internal/runtime"
	"brale-core/internal/store"
)

type portfolioUsecase struct {
	execClient  runtimeExecClient
	store       portfolioStore
	allowSymbol func(symbol string) bool
}

type portfolioStore interface {
	store.PositionQueryStore
	store.RiskPlanQueryStore
}

const (
	dashboardRiskPlanTimelineLimit = 6
)

type dashboardPnLProvenance struct {
	RealizedSource   string
	UnrealizedSource string
	TotalSource      string
}

func newPortfolioUsecase(s *Server) portfolioUsecase {
	if s == nil {
		return portfolioUsecase{}
	}
	return portfolioUsecase{execClient: s.ExecClient, store: s.Store, allowSymbol: s.AllowSymbol}
}

func (u portfolioUsecase) balanceUSDT(ctx context.Context) float64 {
	if u.execClient == nil {
		return 0
	}
	quote, err := u.execClient.Balance(ctx)
	if err != nil {
		return 0
	}
	return execution.BalanceEquity(quote)
}

func (u portfolioUsecase) buildObserveAccountState(ctx context.Context) (execution.AccountState, error) {
	if u.execClient == nil {
		return execution.AccountState{}, fmt.Errorf("exec client missing")
	}
	quote, err := u.execClient.Balance(ctx)
	if err != nil {
		return execution.AccountState{}, err
	}
	return execution.AccountStateFromBalance(quote)
}

func (u portfolioUsecase) buildPositionStatus(ctx context.Context) ([]PositionStatusItem, error) {
	if u.execClient == nil {
		return nil, fmt.Errorf("exec client missing")
	}
	trades, err := u.execClient.ListOpenTrades(ctx)
	if err != nil {
		return nil, err
	}
	positions := make([]PositionStatusItem, 0, len(trades))
	for _, tr := range trades {
		symbol := normalizeFreqtradePair(tr.Pair)
		if symbol == "" {
			continue
		}
		if u.allowSymbol != nil && !u.allowSymbol(symbol) {
			continue
		}
		amount := float64(tr.Amount)
		amountRequested := float64(tr.AmountRequested)
		margin := float64(tr.StakeAmount)
		if margin <= 0 {
			margin = float64(tr.OpenTradeValue)
		}
		entryPrice := float64(tr.OpenRate)
		currentPrice := float64(tr.CurrentRate)
		pnl, _ := resolveDashboardPnLFromTrade(tr)
		openedAt, durationMin, durationSec := positionStatusTiming(int64(tr.OpenFillTimestamp))
		riskState := u.lookupRiskState(ctx, symbol)
		side := "long"
		if tr.IsShort {
			side = "short"
		}
		positions = append(positions, PositionStatusItem{
			Symbol:           symbol,
			Amount:           amount,
			AmountRequested:  amountRequested,
			MarginAmount:     margin,
			EntryPrice:       entryPrice,
			CurrentPrice:     currentPrice,
			Side:             side,
			ProfitTotal:      pnl.Total,
			ProfitRealized:   pnl.Realized,
			ProfitUnrealized: pnl.Unrealized,
			OpenedAt:         openedAt,
			DurationMin:      durationMin,
			DurationSec:      durationSec,
			TakeProfits:      riskState.TakeProfits,
			StopLoss:         riskState.StopLoss,
		})
	}
	return positions, nil
}

func resolveDashboardPnLFromTrade(tr execution.Trade) (DashboardPnLCard, dashboardPnLProvenance) {
	card, provenance := readmodel.ResolvePnLFromTrade(tr)
	return DashboardPnLCard{Realized: card.Realized, Unrealized: card.Unrealized, Total: card.Total}, dashboardPnLProvenance{
		RealizedSource:   provenance.RealizedSource,
		UnrealizedSource: provenance.UnrealizedSource,
		TotalSource:      provenance.TotalSource,
	}
}

func reconcileDashboardPnL(pnl DashboardPnLCard) DashboardReconciliation {
	rc := readmodel.ReconcilePnL(readmodel.PnLCard{Realized: pnl.Realized, Unrealized: pnl.Unrealized, Total: pnl.Total})
	return DashboardReconciliation{Status: rc.Status, DriftAbs: rc.DriftAbs, DriftPct: rc.DriftPct, DriftThreshold: rc.DriftThreshold}
}

func (u portfolioUsecase) lookupRiskState(ctx context.Context, symbol string) dashboardRiskState {
	if u.store == nil {
		return dashboardRiskState{}
	}
	pos, ok, storeErr := u.store.FindPositionBySymbol(ctx, symbol, position.OpenPositionStatuses)
	if storeErr != nil || !ok {
		return dashboardRiskState{}
	}
	return loadDashboardRiskState(ctx, u.store, pos, dashboardRiskPlanTimelineLimit)
}

func (u portfolioUsecase) mapDashboardOverviewSymbol(ctx context.Context, tr execution.Trade) DashboardOverviewSymbol {
	symbol := normalizeFreqtradePair(tr.Pair)
	riskState := u.lookupRiskState(ctx, symbol)
	pnl, _ := resolveDashboardPnLFromTrade(tr)
	leverage := resolveDashboardLeverage(tr)

	side := "long"
	if tr.IsShort {
		side = "short"
	}

	return DashboardOverviewSymbol{
		Symbol: symbol,
		Position: DashboardPositionCard{
			Side:             side,
			Amount:           float64(tr.Amount),
			Leverage:         leverage,
			EntryPrice:       float64(tr.OpenRate),
			CurrentPrice:     float64(tr.CurrentRate),
			TakeProfits:      riskState.TakeProfits,
			StopLoss:         riskState.StopLoss,
			RiskPlanTimeline: riskState.Timeline,
		},
		PnL:            pnl,
		Reconciliation: reconcileDashboardPnL(pnl),
	}
}

func resolveDashboardLeverage(tr execution.Trade) float64 {
	return readmodel.ResolveLeverage(tr)
}

func (u portfolioUsecase) buildDashboardAccountPnL(ctx context.Context) (DashboardPnLCard, bool) {
	if u.execClient == nil {
		return DashboardPnLCard{}, false
	}
	quote, err := u.execClient.Balance(ctx)
	if err != nil {
		return DashboardPnLCard{}, false
	}
	totalProfit, ok := extractDashboardAccountTotalProfit(quote)
	if !ok {
		return DashboardPnLCard{}, false
	}
	unrealized := u.sumOpenUnrealizedPnL(ctx)
	realized := totalProfit - unrealized
	return DashboardPnLCard{Realized: realized, Unrealized: unrealized, Total: totalProfit}, true
}

func (u portfolioUsecase) sumOpenUnrealizedPnL(ctx context.Context) float64 {
	if u.execClient == nil {
		return 0
	}
	trades, err := u.execClient.ListOpenTrades(ctx)
	if err != nil {
		return 0
	}
	sum := 0.0
	for _, tr := range trades {
		symbol := normalizeFreqtradePair(tr.Pair)
		if symbol == "" {
			continue
		}
		if u.allowSymbol != nil && !u.allowSymbol(symbol) {
			continue
		}
		sum += float64(tr.ProfitAbs)
	}
	return sum
}

func extractDashboardAccountTotalProfit(quote map[string]any) (float64, bool) {
	return readmodel.ExtractAccountTotalProfit(quote)
}

func (u portfolioUsecase) buildDashboardOverview(ctx context.Context, rawSymbol string) (string, []DashboardOverviewSymbol, *usecaseError) {
	trimmedRaw := strings.TrimSpace(rawSymbol)
	normalizedSymbol := runtime.NormalizeSymbol(trimmedRaw)
	if trimmedRaw != "" && (normalizedSymbol == "" || !isValidDashboardSymbol(normalizedSymbol)) {
		return "", nil, &usecaseError{Status: 400, Code: "invalid_symbol", Message: "symbol 非法", Details: rawSymbol}
	}
	if normalizedSymbol != "" && u.allowSymbol != nil && !u.allowSymbol(normalizedSymbol) {
		return "", nil, &usecaseError{Status: 400, Code: "symbol_not_allowed", Message: "symbol 不在允许列表", Details: normalizedSymbol}
	}
	if u.execClient == nil {
		return "", nil, &usecaseError{Status: 502, Code: "dashboard_overview_failed", Message: "dashboard 概览获取失败", Details: "exec client missing"}
	}

	trades, err := u.execClient.ListOpenTrades(ctx)
	if err != nil {
		return "", nil, &usecaseError{Status: 502, Code: "dashboard_overview_failed", Message: "dashboard 概览获取失败", Details: err.Error()}
	}

	cards := make([]DashboardOverviewSymbol, 0, len(trades))
	for _, tr := range trades {
		symbol := normalizeFreqtradePair(tr.Pair)
		if symbol == "" {
			continue
		}
		if u.allowSymbol != nil && !u.allowSymbol(symbol) {
			continue
		}
		if normalizedSymbol != "" && symbol != normalizedSymbol {
			continue
		}
		cards = append(cards, u.mapDashboardOverviewSymbol(ctx, tr))
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Symbol < cards[j].Symbol
	})

	if len(cards) == 0 {
		return normalizedSymbol, []DashboardOverviewSymbol{}, nil
	}
	return normalizedSymbol, cards, nil
}

func isValidDashboardSymbol(symbol string) bool {
	if symbol == "" {
		return false
	}
	for _, r := range symbol {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func (u portfolioUsecase) buildTradeHistory(ctx context.Context, limit, offset int, symbolFilter string) ([]TradeHistoryItem, error) {
	if u.execClient == nil {
		return nil, fmt.Errorf("exec client missing")
	}
	normalizedFilter := runtime.NormalizeSymbol(strings.TrimSpace(symbolFilter))
	resp, err := u.execClient.ListTrades(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	if limit > 0 && offset <= 0 && resp.TotalTrades > limit {
		latestOffset := resp.TotalTrades - limit
		if latestOffset < 0 {
			latestOffset = 0
		}
		resp, err = u.execClient.ListTrades(ctx, limit, latestOffset)
		if err != nil {
			return nil, err
		}
	}

	positionByExecID := map[string]store.PositionRecord{}
	if u.store != nil {
		positions, err := u.store.ListPositionsByStatus(ctx, []string{position.PositionClosed})
		if err == nil {
			for _, pos := range positions {
				execID := strings.TrimSpace(pos.ExecutorPositionID)
				if execID == "" {
					continue
				}
				positionByExecID[execID] = pos
			}
		}
	}

	items := make([]TradeHistoryItem, 0, len(resp.Trades))
	for _, tr := range resp.Trades {
		symbol := normalizeFreqtradePair(tr.Pair)
		if symbol == "" {
			continue
		}
		if u.allowSymbol != nil && !u.allowSymbol(symbol) {
			continue
		}
		if normalizedFilter != "" && symbol != normalizedFilter {
			continue
		}
		profit := float64(tr.CloseProfitAbs)
		if profit == 0 {
			profit = float64(tr.RealizedProfit)
		}
		durationSec := int64(tr.TradeDuration)
		if tr.TradeDurationSeconds > 0 {
			durationSec = int64(tr.TradeDurationSeconds)
		}
		openedAt := parseMillisTimestamp(int64(tr.OpenFillTimestamp))
		closedAt := parseMillisTimestamp(int64(tr.CloseTimestamp))
		if closedAt.IsZero() && !openedAt.IsZero() && durationSec > 0 {
			closedAt = openedAt.Add(time.Duration(durationSec) * time.Second)
		}
		side := "long"
		if tr.IsShort {
			side = "short"
		}

		riskState := dashboardRiskState{}
		if u.store != nil {
			execID := strconv.Itoa(int(tr.ID))
			if pos, ok := positionByExecID[execID]; ok {
				riskState = loadDashboardRiskState(ctx, u.store, pos, dashboardRiskPlanTimelineLimit)
			}
		}

		items = append(items, TradeHistoryItem{
			Symbol:       symbol,
			Side:         side,
			Amount:       float64(tr.Amount),
			MarginAmount: float64(tr.StakeAmount),
			OpenedAt:     openedAt,
			ClosedAt:     closedAt,
			DurationSec:  durationSec,
			Profit:       profit,
			StopLoss:     riskState.StopLoss,
			TakeProfits:  riskState.TakeProfits,
			Timeline:     riskState.Timeline,
		})
	}
	return items, nil
}

func normalizeFreqtradePair(pair string) string {
	return readmodel.NormalizeFreqtradePair(pair)
}

func parseMillisTimestamp(ts int64) time.Time {
	return readmodel.ParseMillisTimestamp(ts)
}

func positionStatusTiming(openFillTimestamp int64) (string, int64, int64) {
	return readmodel.PositionStatusTiming(openFillTimestamp)
}
