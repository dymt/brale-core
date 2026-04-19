package position

import (
	"context"
	"strings"

	"brale-core/internal/pkg/logging"
	"brale-core/internal/store"

	"go.uber.org/zap"
)

type closeSummary struct {
	stopPrice  float64
	tpPrices   []float64
	qty        float64
	entryPrice float64
}

func (s *PositionService) buildCloseSummary(pos store.PositionRecord, triggerPrice float64, request CloseRequest) closeSummary {
	stopPrice := float64(0)
	tpPrices := []float64{}
	if len(pos.RiskJSON) > 0 {
		if plan, err := DecodeRiskPlan(pos.RiskJSON); err == nil {
			stopPrice = plan.StopPrice
			for _, level := range plan.TPLevels {
				if level.Price > 0 {
					tpPrices = append(tpPrices, level.Price)
				}
			}
		}
	}
	qty := pos.Qty
	if qty <= 0 {
		qty = request.EffectiveCloseQty
	}
	if qty <= 0 && pos.InitialStake > 0 && pos.AvgEntry > 0 {
		qty = pos.InitialStake / pos.AvgEntry
	}
	entryPrice := pos.AvgEntry
	if entryPrice <= 0 && pos.LastPrice > 0 {
		entryPrice = pos.LastPrice
	}
	if entryPrice <= 0 && triggerPrice > 0 {
		entryPrice = triggerPrice
	}
	return closeSummary{
		stopPrice:  stopPrice,
		tpPrices:   tpPrices,
		qty:        qty,
		entryPrice: entryPrice,
	}
}

func (s *PositionService) logAndNotifyClose(ctx context.Context, pos store.PositionRecord, reason string, triggerPrice float64, request CloseRequest, intentKind string) {
	summary := s.buildCloseSummary(pos, triggerPrice, request)
	logging.FromContext(ctx).Named("execution").Info("position close detail",
		zap.String("symbol", pos.Symbol),
		zap.String("direction", strings.TrimSpace(pos.Side)),
		zap.Float64("qty", summary.qty),
		zap.Float64("requested_close_qty", request.RequestedCloseQty),
		zap.Float64("effective_close_qty", request.EffectiveCloseQty),
		zap.Bool("forced_full_close", request.ForcedFullClose),
		zap.String("intent_kind", intentKind),
		zap.Float64("entry", summary.entryPrice),
		zap.Float64("trigger_price", triggerPrice),
		zap.Float64("stop", summary.stopPrice),
		zap.Float64s("take_profits", summary.tpPrices),
		zap.String("reason", strings.TrimSpace(reason)),
		zap.Float64("risk_pct", pos.RiskPct),
		zap.Float64("leverage", pos.Leverage),
	)
	if s.Notifier == nil {
		return
	}
	err := s.Notifier.SendPositionClose(ctx, PositionCloseNotice{
		Symbol:             pos.Symbol,
		Direction:          strings.TrimSpace(pos.Side),
		Qty:                summary.qty,
		CloseQty:           request.EffectiveCloseQty,
		RequestedCloseQty:  request.RequestedCloseQty,
		ForcedFullClose:    request.ForcedFullClose,
		IntentKind:         intentKind,
		EntryPrice:         summary.entryPrice,
		TriggerPrice:       triggerPrice,
		StopPrice:          summary.stopPrice,
		TakeProfits:        summary.tpPrices,
		Reason:             strings.TrimSpace(reason),
		RiskPct:            pos.RiskPct,
		Leverage:           pos.Leverage,
		PositionID:         pos.PositionID,
		ExecutorPositionID: strings.TrimSpace(pos.ExecutorPositionID),
	})
	if err != nil {
		logging.FromContext(ctx).Named("execution").Error("position close notify failed", zap.Error(err))
	}
}

func (s *PositionService) logCloseSubmitError(ctx context.Context, pos store.PositionRecord, clientOrderID string, intentKind string, err error) {
	logger := logging.FromContext(ctx).Named("execution")
	logger.Error("order submit failed",
		zap.String("position_id", pos.PositionID),
		zap.String("symbol", pos.Symbol),
		zap.String("client_order_id", clientOrderID),
		zap.String("intent_kind", intentKind),
		zap.Error(err),
	)
	if s.Notifier != nil {
		if notifyErr := s.Notifier.SendError(ctx, ErrorNotice{Severity: "error", Component: "execution", Symbol: pos.Symbol, Message: err.Error()}); notifyErr != nil {
			logger.Error("notify error failed", zap.Error(notifyErr))
		}
	}
}
