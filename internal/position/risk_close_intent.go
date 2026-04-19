package position

import (
	"context"
	"fmt"
	"strings"

	"brale-core/internal/market"
	"brale-core/internal/risk"
	"brale-core/internal/store"

	"go.uber.org/zap"
)

func (m *RiskMonitor) resolveCloseIntent(ctx context.Context, pos store.PositionRecord, plan risk.RiskPlan, trigger risk.RiskTrigger, logger *zap.Logger) (CloseRequest, string) {
	statusQty := float64(0)
	if shouldFetchStatusAmount(pos, trigger) {
		qty, ok, err := m.fetchStatusAmount(ctx, pos.ExecutorPositionID)
		if err != nil {
			logger.Warn("freqtrade status amount fetch failed", zap.Error(err), zap.Duration("timeout", riskMonitorStatusFetchTimeout))
		} else if ok {
			statusQty = qty
		}
	}
	request, reason := resolveCloseRequest(pos, plan, trigger, statusQty)
	if request.EffectiveCloseQty <= 0 {
		logger.Warn("skip close intent because reliable position quantity is unavailable",
			zap.String("trigger_type", trigger.Type),
			zap.String("level_id", trigger.LevelID),
			zap.Float64("qty_pct", trigger.QtyPct),
			zap.Float64("position_qty", request.PositionQty),
			zap.Float64("status_qty", statusQty),
			zap.Float64("plan_initial_qty", plan.InitialQty),
		)
	}
	return request, reason
}

func (m *RiskMonitor) submitCloseIntent(ctx context.Context, pos store.PositionRecord, quote market.PriceQuote, request CloseRequest, reason string, logger *zap.Logger) error {
	if request.EffectiveCloseQty <= 0 {
		return nil
	}
	logger.Info("close intent arm",
		zap.String("reason", reason),
		zap.Float64("requested_close_qty", request.RequestedCloseQty),
		zap.Float64("effective_close_qty", request.EffectiveCloseQty),
		zap.Bool("forced_full_close", request.ForcedFullClose),
		zap.Float64("position_qty", request.PositionQty),
		zap.Float64("stored_position_qty", pos.Qty),
		zap.Float64("entry_price", pos.AvgEntry),
		zap.Float64("mark_price", quote.Price),
	)
	_, err := m.Positions.ArmClose(ctx, pos, reason, quote.Price, request)
	if err != nil {
		logger.Error("arm close failed", zap.Error(err), zap.String("reason", reason))
		return fmt.Errorf("arm close for %s: %w", strings.TrimSpace(pos.Symbol), err)
	}
	return nil
}
