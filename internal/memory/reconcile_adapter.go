package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"brale-core/internal/pkg/logging"
	"brale-core/internal/store"

	"go.uber.org/zap"
)

type ReconcileReflectorAdapter struct {
	Reflector *Reflector
}

func (a *ReconcileReflectorAdapter) ReflectOnClose(ctx context.Context, pos store.PositionRecord, exitPrice float64) {
	if a.Reflector == nil {
		return
	}
	logger := logging.FromContext(ctx).Named("reflector")

	entryPrice := fmt.Sprintf("%.8f", pos.AvgEntry)
	exitPriceStr := fmt.Sprintf("%.8f", exitPrice)

	pnlPct := float64(0)
	if pos.AvgEntry > 0 {
		diff := exitPrice - pos.AvgEntry
		if strings.EqualFold(pos.Side, "short") {
			diff = pos.AvgEntry - exitPrice
		}
		pnlPct = diff / pos.AvgEntry * 100
	}

	duration := "unknown"
	if !pos.CreatedAt.IsZero() {
		d := time.Since(pos.CreatedAt)
		if d > 0 {
			duration = d.Round(time.Minute).String()
		}
	}

	gateReason := ""
	if pos.Source != "" {
		gateReason = pos.Source
	}

	input := ReflectionInput{
		Symbol:     pos.Symbol,
		PositionID: pos.PositionID,
		Direction:  strings.TrimSpace(pos.Side),
		EntryPrice: entryPrice,
		ExitPrice:  exitPriceStr,
		PnLPercent: fmt.Sprintf("%.2f", pnlPct),
		Duration:   duration,
		GateReason: gateReason,
	}

	if err := a.Reflector.Reflect(ctx, input); err != nil {
		logger.Error("position reflection failed", zap.Error(err), zap.String("position_id", pos.PositionID))
	}
}
