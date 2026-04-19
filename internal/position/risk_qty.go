package position

import (
	"math"
	"strings"

	"brale-core/internal/risk"
	"brale-core/internal/store"
)

const closeQtyPrecision = 1e-8

type CloseRequest struct {
	RequestedCloseQty float64
	EffectiveCloseQty float64
	PositionQty       float64
	BaseQty           float64
	ForcedFullClose   bool
}

type normalizedCloseRequest = CloseRequest

func resolveCloseQty(pos store.PositionRecord, plan risk.RiskPlan, trigger risk.RiskTrigger, statusQty float64) (float64, float64, string) {
	request, reason := resolveCloseRequest(pos, plan, trigger, statusQty)
	return request.EffectiveCloseQty, request.PositionQty, reason
}

func resolveCloseRequest(pos store.PositionRecord, plan risk.RiskPlan, trigger risk.RiskTrigger, statusQty float64) (CloseRequest, string) {
	positionQty := resolveBaseCloseQty(pos, statusQty)
	requestedCloseQty := positionQty
	reason := resolveCloseReason(trigger)
	if isPartialTakeProfit(trigger) {
		baseQty := resolvePartialTPBaseQty(plan)
		if baseQty > 0 {
			requestedCloseQty = baseQty * trigger.QtyPct
		} else {
			requestedCloseQty = 0
		}
	}
	requestedCloseQty = floorCloseQty(requestedCloseQty)
	requestedCloseQty = clipCloseQty(requestedCloseQty, positionQty)
	baseQty := resolveResidualBaseQty(pos, plan, positionQty)
	return normalizeCloseRequest(baseQty, positionQty, requestedCloseQty), reason
}

func resolveBaseCloseQty(pos store.PositionRecord, statusQty float64) float64 {
	if statusQty > 0 {
		return statusQty
	}
	if pos.Qty > 0 {
		return pos.Qty
	}
	return 0
}

func resolveCloseReason(trigger risk.RiskTrigger) string {
	switch trigger.Type {
	case "FORCE_EXIT":
		reason := strings.TrimSpace(trigger.Reason)
		if reason == "" {
			return "force_exit"
		}
		return reason
	case "TAKE_PROFIT":
		return risk.FormatTPReason(trigger.LevelID)
	default:
		return "stop_loss_hit"
	}
}

func isPartialTakeProfit(trigger risk.RiskTrigger) bool {
	return trigger.Type == "TAKE_PROFIT" && trigger.QtyPct > 0 && trigger.QtyPct < 1
}

func isFinalTakeProfit(trigger risk.RiskTrigger) bool {
	return trigger.Type == "TAKE_PROFIT" && trigger.QtyPct == 1.0
}

func resolvePartialTPBaseQty(plan risk.RiskPlan) float64 {
	if plan.InitialQty > 0 {
		return plan.InitialQty
	}
	return 0
}

func resolveResidualBaseQty(pos store.PositionRecord, plan risk.RiskPlan, currentQty float64) float64 {
	if plan.InitialQty > 0 {
		return plan.InitialQty
	}
	if pos.InitialStake > 0 && pos.AvgEntry > 0 {
		return pos.InitialStake / pos.AvgEntry
	}
	if currentQty > 0 {
		return currentQty
	}
	if pos.Qty > 0 {
		return pos.Qty
	}
	return 0
}

func normalizeCloseRequest(baseQty, positionQty, requestedCloseQty float64) CloseRequest {
	request := CloseRequest{
		RequestedCloseQty: requestedCloseQty,
		EffectiveCloseQty: requestedCloseQty,
		PositionQty:       positionQty,
		BaseQty:           baseQty,
	}
	if request.EffectiveCloseQty <= 0 || positionQty <= 0 {
		return request
	}
	request.EffectiveCloseQty = cleanupDustCloseQty(request.EffectiveCloseQty, positionQty)
	request.ForcedFullClose = shouldForceFullCloseByResidual(baseQty, positionQty, request.RequestedCloseQty)
	if request.ForcedFullClose {
		request.EffectiveCloseQty = positionQty
	}
	return request
}

func shouldForceFullCloseByResidual(baseQty, positionQty, requestedCloseQty float64) bool {
	if baseQty <= 0 || positionQty <= 0 || requestedCloseQty <= 0 || requestedCloseQty >= positionQty {
		return false
	}
	remainingQty := positionQty - requestedCloseQty
	if remainingQty <= 0 {
		return false
	}
	thresholdQty := math.Max(baseQty*ResidualFullCloseRatio, closeQtyPrecision)
	return remainingQty <= thresholdQty
}

func IsResidualCloseFlowQty(pos store.PositionRecord, currentQty float64) bool {
	if currentQty <= 0 {
		return false
	}
	plan := risk.RiskPlan{}
	if len(pos.RiskJSON) > 0 {
		if decoded, err := DecodeRiskPlan(pos.RiskJSON); err == nil {
			plan = decoded
		}
	}
	// currentQty here is the observed residual after close-flow reconciliation.
	// When we need a fallback base, use the stored position quantity rather than
	// the residual itself, otherwise the 3% rule becomes mathematically impossible
	// to hit (currentQty <= currentQty*0.03).
	baseQty := resolveResidualBaseQty(pos, plan, 0)
	if baseQty <= 0 {
		return false
	}
	thresholdQty := math.Max(baseQty*ResidualFullCloseRatio, closeQtyPrecision)
	return currentQty <= thresholdQty
}

func cleanupDustCloseQty(closeQty, limitQty float64) float64 {
	if closeQty <= 0 || limitQty <= 0 || closeQty >= limitQty {
		return closeQty
	}
	dust := math.Max(limitQty*DustThresholdRatio, closeQtyPrecision)
	if limitQty-closeQty <= dust {
		return limitQty
	}
	return closeQty
}

func clipCloseQty(closeQty, limitQty float64) float64 {
	if limitQty > 0 && closeQty > limitQty {
		return limitQty
	}
	return closeQty
}

func floorCloseQty(value float64) float64 {
	if value <= 0 || closeQtyPrecision <= 0 {
		return value
	}
	return math.Floor(value/closeQtyPrecision) * closeQtyPrecision
}

func shouldFetchStatusAmount(pos store.PositionRecord, trigger risk.RiskTrigger) bool {
	if pos.Qty <= 0 {
		return true
	}
	return isPartialTakeProfit(trigger)
}

func clampMinOneFloat(value float64) float64 {
	return math.Max(value, 1)
}
