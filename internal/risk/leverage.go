package risk

import (
	"math"
	"strings"
)

const (
	defaultLiquidationFee = 0.0002
	mmrThresholdMargin    = 10000
	mmrLow                = 0.005
	mmrHigh               = 0.015
)

type LeverageLiquidation struct {
	PositionSize     float64
	Leverage         float64
	LiquidationPrice float64
	MMR              float64
	Fee              float64
}

func ResolveLeverageAndLiquidation(entry float64, positionSize float64, maxInvestAmt float64, maxLeverage float64, direction string) LeverageLiquidation {
	result := LeverageLiquidation{
		PositionSize:     positionSize,
		Leverage:         1.0,
		LiquidationPrice: 0.0,
		MMR:              0.0,
		Fee:              defaultLiquidationFee,
	}
	if entry <= 0 || positionSize <= 0 || maxInvestAmt <= 0 {
		return result
	}
	notional := positionSize * entry
	leverage := math.Ceil(notional / maxInvestAmt)
	if leverage < 1 {
		leverage = 1
	}
	if maxLeverage > 0 {
		maxLev := math.Floor(maxLeverage)
		if maxLev < 1 {
			maxLev = 1
		}
		if leverage > maxLev {
			leverage = maxLev
		}
	}
	marginRequired := notional / leverage
	if marginRequired > maxInvestAmt {
		positionSize = (leverage * maxInvestAmt) / entry
		notional = positionSize * entry
		marginRequired = notional / leverage
	}
	result.PositionSize = positionSize
	result.Leverage = leverage
	result.MMR = maintenanceMarginRate(marginRequired)
	result.LiquidationPrice = CalcLiquidationPrice(entry, direction, leverage, result.MMR, result.Fee)
	return result
}

func CalcLiquidationPriceForPosition(entry float64, direction string, leverage float64, positionSize float64) float64 {
	if entry <= 0 || positionSize <= 0 || leverage <= 0 {
		return 0
	}
	marginRequired := (entry * positionSize) / leverage
	return CalcLiquidationPrice(entry, direction, leverage, maintenanceMarginRate(marginRequired), defaultLiquidationFee)
}

func CalcLiquidationPrice(entry float64, direction string, leverage float64, mmr float64, fee float64) float64 {
	if entry <= 0 || leverage <= 1 {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "long":
		return entry * (1 - 1/leverage + mmr + fee)
	case "short":
		return entry * (1 + 1/leverage - mmr - fee)
	default:
		return 0
	}
}

func IsStopBeyondLiquidation(direction string, stopLoss float64, liqPrice float64) bool {
	if liqPrice <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "long":
		return stopLoss <= liqPrice
	case "short":
		return stopLoss >= liqPrice
	default:
		return false
	}
}

func maintenanceMarginRate(marginRequired float64) float64 {
	if marginRequired > mmrThresholdMargin {
		return mmrHigh
	}
	return mmrLow
}
