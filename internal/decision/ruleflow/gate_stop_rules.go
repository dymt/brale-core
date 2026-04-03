package ruleflow

const (
	breakContinuationMinResonanceBonus = 0.05
)

var gateDirectionRules = []gateDecisionRule{
	{
		Step: "direction",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   gateReasonDirectionUnclear,
			Priority: gatePriorityDirection,
			StopStep: "direction",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.StructureDirection == "" || ctx.Inputs.StructureDirection == "none"
		},
	},
}

var gateDataRules = []gateDecisionRule{
	{
		Step: "data",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   gateReasonDataMissing,
			Priority: gatePriorityDataMissing,
			StopStep: "data",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.MissingProviders && ctx.Inputs.State != "IN_POSITION"
		},
	},
}

var gateStructureStopRules = []gateDecisionRule{
	{
		Step: "structure",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   gateReasonStructInvalidation,
			Priority: gatePriorityStructInvalidation,
			StopStep: "structure",
		},
		Match: func(ctx gateDecisionContext) bool {
			if ctx.Inputs.StructureTag == "structure_broken" && allowStructureBreakContinuation(ctx) {
				return false
			}
			return ctx.Inputs.StructureTag == "structure_broken" || !ctx.Inputs.StructureIntegrity
		},
	},
}

func allowStructureBreakContinuation(ctx gateDecisionContext) bool {
	if ctx.Inputs.StructureTag != "structure_broken" {
		return false
	}
	if ctx.Inputs.StructureDirection != "long" && ctx.Inputs.StructureDirection != "short" {
		return false
	}
	if ctx.Inputs.IndicatorTag != "trend_surge" {
		return false
	}
	if !ctx.Inputs.MomentumExpansion || !ctx.Inputs.Alignment || ctx.Inputs.MeanRevNoise {
		return false
	}
	if !ctx.Inputs.ConsensusResonant || ctx.Inputs.ConsensusResonance < breakContinuationMinResonanceBonus {
		return false
	}
	if ctx.Inputs.ScoreThreshold <= 0 || ctx.Inputs.ConfidenceThreshold <= 0 {
		return false
	}
	return absGateScore(ctx.Inputs.ConsensusScore) >= ctx.Inputs.ScoreThreshold && ctx.Inputs.ConsensusConfidence >= ctx.Inputs.ConfidenceThreshold
}

func absGateScore(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
