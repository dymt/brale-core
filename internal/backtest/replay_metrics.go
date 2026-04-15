package backtest

import (
	"math"
	"strings"

	"brale-core/internal/decision/fund"
)

type ReplayMetrics struct {
	TotalRounds      int     `json:"total_rounds"`
	ReplayableRounds int     `json:"replayable_rounds"`
	SkippedCount     int     `json:"skipped_count"`
	AllowCount       int     `json:"allow_count"`
	VetoCount        int     `json:"veto_count"`
	WaitCount        int     `json:"wait_count"`
	TruePositive     int     `json:"true_positive"`
	FalsePositive    int     `json:"false_positive"`
	TrueNegative     int     `json:"true_negative"`
	FalseNegative    int     `json:"false_negative"`
	Precision        float64 `json:"precision"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`
	MaxDrawdown      float64 `json:"max_drawdown_pct"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	ChangedCount     int     `json:"changed_count"`
	ChangeRate       float64 `json:"change_rate"`
}

func computeMetrics(rounds []ReplayRound) ReplayMetrics {
	metrics := ReplayMetrics{TotalRounds: len(rounds)}
	if len(rounds) == 0 {
		return metrics
	}
	returns := make([]float64, 0, len(rounds))
	positiveSum := 0.0
	negativeSum := 0.0
	equity := 1.0
	peak := 1.0

	for _, round := range rounds {
		if round.Changed {
			metrics.ChangedCount++
		}
		if round.Skipped {
			metrics.SkippedCount++
			continue
		}
		metrics.ReplayableRounds++
		action := strings.ToUpper(strings.TrimSpace(round.ReplayedGate.DecisionAction))
		switch action {
		case "ALLOW":
			metrics.AllowCount++
		case "VETO":
			metrics.VetoCount++
		case "WAIT":
			metrics.WaitCount++
		}

		priceValid := round.PriceAtDecision > 0 && round.PriceAfter > 0
		tradeDirection := replayTradeDirection(round)
		marketDirection := priceMoveDirection(round.PriceAtDecision, round.PriceAfter)
		if priceValid && tradeDirection != "none" {
			if action == "ALLOW" {
				if tradeDirection == marketDirection {
					metrics.TruePositive++
				} else {
					metrics.FalsePositive++
				}
			} else {
				if tradeDirection == marketDirection {
					metrics.FalseNegative++
				} else {
					metrics.TrueNegative++
				}
			}
		}

		ret := 0.0
		if priceValid && action == "ALLOW" && tradeDirection != "none" {
			ret = normalizedReturn(tradeDirection, round.PriceAtDecision, round.PriceAfter)
			if ret > 0 {
				positiveSum += ret
			} else if ret < 0 {
				negativeSum += -ret
			}
		}
		returns = append(returns, ret)
		equity *= 1 + ret
		if equity > peak {
			peak = equity
		}
		if peak > 0 {
			drawdown := (peak - equity) / peak
			if drawdown > metrics.MaxDrawdown {
				metrics.MaxDrawdown = drawdown
			}
		}
	}

	if metrics.AllowCount > 0 {
		metrics.WinRate = float64(metrics.TruePositive) / float64(metrics.AllowCount)
	}
	if denom := metrics.TruePositive + metrics.FalsePositive; denom > 0 {
		metrics.Precision = float64(metrics.TruePositive) / float64(denom)
	}
	if negativeSum > 0 {
		metrics.ProfitFactor = positiveSum / negativeSum
	}
	metrics.SharpeRatio = sharpeRatio(returns)
	if metrics.MaxDrawdown > 0 {
		metrics.CalmarRatio = ((equity - 1.0) / metrics.MaxDrawdown)
	}
	if metrics.TotalRounds > 0 {
		metrics.ChangeRate = float64(metrics.ChangedCount) / float64(metrics.TotalRounds)
	}
	return metrics
}

func replayTradeDirection(round ReplayRound) string {
	direction := strings.ToLower(strings.TrimSpace(round.ReplayedGate.Direction))
	if direction == "long" || direction == "short" {
		return direction
	}
	score := scoreFromGate(round.ReplayedGate)
	switch {
	case score > 0:
		return "long"
	case score < 0:
		return "short"
	default:
		return "none"
	}
}

func scoreFromGate(gate fund.GateDecision) float64 {
	raw, ok := gate.Derived["direction_consensus"]
	if !ok {
		return 0
	}
	consensus, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	switch value := consensus["score"].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	default:
		return 0
	}
}

func priceMoveDirection(priceAtDecision, priceAfter float64) string {
	switch {
	case priceAfter > priceAtDecision:
		return "long"
	case priceAfter < priceAtDecision:
		return "short"
	default:
		return "none"
	}
}

func normalizedReturn(direction string, priceAtDecision, priceAfter float64) float64 {
	if priceAtDecision <= 0 {
		return 0
	}
	change := (priceAfter - priceAtDecision) / priceAtDecision
	if direction == "short" {
		return -change
	}
	return change
}

func sharpeRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))
	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))
	if variance <= 0 {
		return 0
	}
	return mean / math.Sqrt(variance)
}
