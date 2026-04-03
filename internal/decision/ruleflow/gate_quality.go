package ruleflow

import "math"

func computeSetupQuality(structureClear, structureIntegrity, alignment, momentumExpansion, meanRevNoise bool, scriptBonus float64, indicatorTag string) float64 {
	score := 0.30*boolToFloat(structureClear) +
		0.25*boolToFloat(structureIntegrity) +
		0.20*boolToFloat(alignment) +
		0.15*boolToFloat(momentumExpansion) +
		0.10*scriptBonus -
		0.20*boolToFloat(meanRevNoise)
	score -= resolveConsistencyPenalty(indicatorTag, momentumExpansion, alignment, meanRevNoise)
	return clampGateFloat(score, 0, 1)
}

func resolveScriptBonus(indicatorTag, structureTag string) (bonus float64, scriptName string) {
	switch {
	case indicatorTag == "trend_surge" && structureTag == "breakout_confirmed":
		return 1.0, "A"
	case indicatorTag == "pullback_entry" && structureTag == "support_retest":
		return 0.7, "B"
	case indicatorTag == "divergence_reversal" && structureTag == "support_retest":
		return 0.5, "C"
	case indicatorTag == "trend_surge" && structureTag == "support_retest":
		return 0.6, "D"
	case indicatorTag == "pullback_entry" && structureTag == "breakout_confirmed":
		return 0.4, "E"
	case indicatorTag == "divergence_reversal" && structureTag == "breakout_confirmed":
		return 0.3, "F"
	case indicatorTag == "trend_surge" && structureTag == "structure_broken":
		return 0.2, "G"
	case structureTag == "fakeout_rejection":
		return 0, "FR"
	case indicatorTag == "momentum_weak":
		return 0, "MW"
	default:
		return 0, ""
	}
}

func computeRiskPenalty(mechanicsTag string, liquidationStress bool, liqConfidence string, crowdingAlign bool) float64 {
	if mechanicsTag == "liquidation_cascade" {
		return 1.0
	}
	if liquidationStress && liqConfidence == "high" {
		return 0.60
	}
	if liquidationStress && liqConfidence == "low" {
		return 0.35
	}
	if crowdingAlign {
		return 0.25
	}
	if mechanicsTag == "crowded_long" || mechanicsTag == "crowded_short" {
		return 0.10
	}
	return 0
}

func computeEntryEdge(directionScore, setupQuality, riskPenalty float64) float64 {
	edge := math.Abs(directionScore) * setupQuality * (1 - riskPenalty)
	return clampGateFloat(edge, 0, 1)
}

func resolveGradeFromQuality(setupQuality float64) int {
	switch {
	case setupQuality >= 0.75:
		return gateGradeHigh
	case setupQuality >= 0.55:
		return gateGradeMedium
	case setupQuality >= 0.35:
		return gateGradeLow
	default:
		return gateGradeNone
	}
}

func resolveIndicatorTagConsistency(indicatorTag string, momentumExpansion, alignment, meanRevNoise bool) bool {
	switch indicatorTag {
	case "trend_surge":
		return momentumExpansion && alignment && !meanRevNoise
	case "pullback_entry":
		return !momentumExpansion && alignment && !meanRevNoise
	case "divergence_reversal":
		return !alignment && !meanRevNoise
	case "noise":
		return meanRevNoise
	case "momentum_weak":
		return true
	default:
		return true
	}
}

const (
	consistencyPenaltyHard = 0.25
	consistencyPenaltySoft = 0.10
)

func resolveConsistencyPenalty(indicatorTag string, momentumExpansion, alignment, meanRevNoise bool) float64 {
	if resolveIndicatorTagConsistency(indicatorTag, momentumExpansion, alignment, meanRevNoise) {
		return 0
	}
	switch indicatorTag {
	case "trend_surge":
		// trend_surge + mean_rev_noise or !alignment = narrative contradiction
		if meanRevNoise || !alignment {
			return consistencyPenaltyHard
		}
		// trend_surge + !momentumExpansion only = weaker conflict
		return consistencyPenaltySoft
	case "pullback_entry":
		// pullback with expansion is a soft mismatch, could be transition
		if momentumExpansion {
			return consistencyPenaltySoft
		}
		return consistencyPenaltyHard
	case "divergence_reversal":
		// divergence + alignment = possibly converging, not a hard conflict
		if alignment {
			return consistencyPenaltySoft
		}
		return consistencyPenaltyHard
	case "noise":
		// noise tag but no mean_rev_noise = classification mismatch
		return consistencyPenaltySoft
	default:
		return 0
	}
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func clampGateFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
