package runtimeapi

import (
	"net/http"
	"sort"
	"time"

	"brale-core/internal/runtime"
)

func (s *Server) handleMonitorStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ensureMethod(ctx, w, r, http.MethodGet) {
		return
	}
	status := s.Scheduler.GetScheduleStatus()
	nextRuns := make(map[string]runtime.SymbolNextRun, len(status.NextRuns))
	for _, item := range status.NextRuns {
		nextRuns[item.Symbol] = item
	}
	balance := newPortfolioUsecase(s).balanceUSDT(ctx)
	keys := make([]string, 0, len(s.SymbolConfigs))
	for symbol := range s.SymbolConfigs {
		keys = append(keys, symbol)
	}
	sort.Strings(keys)
	symbols := make([]MonitorSymbolConfig, 0, len(keys))
	for _, symbol := range keys {
		bundle := s.SymbolConfigs[symbol]
		riskPct := bundle.Strategy.RiskManagement.RiskPerTradePct
		riskAmount := 0.0
		if balance > 0 && riskPct > 0 {
			riskAmount = balance * riskPct
		}
		var nextRun time.Time
		klineInterval := ""
		if item, ok := nextRuns[symbol]; ok {
			klineInterval = item.BarInterval
			if item.NextExecution != "" {
				parsed, err := time.ParseInLocation("2006-01-02 15:04", item.NextExecution, time.Local)
				if err == nil {
					nextRun = parsed
				}
			}
		}
		symbols = append(symbols, MonitorSymbolConfig{
			Symbol:        symbol,
			NextRun:       nextRun,
			KlineInterval: klineInterval,
			RiskPct:       riskPct,
			RiskAmount:    riskAmount,
			MaxLeverage:   bundle.Strategy.RiskManagement.MaxLeverage,
			RiskPlan:      buildMonitorRiskPlan(bundle),
		})
	}
	resp := MonitorStatusResponse{
		Status:    "ok",
		Symbols:   symbols,
		Summary:   status.Details,
		RequestID: requestIDFromContext(ctx),
	}
	writeJSON(w, resp)
}
