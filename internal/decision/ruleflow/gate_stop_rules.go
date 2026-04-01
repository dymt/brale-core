package ruleflow

var gateDirectionRules = []gateDecisionRule{
	{
		Step: "direction",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   "CONSENSUS_NOT_PASSED",
			Priority: gatePriorityConsensusFailed,
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
			Reason:   "DATA_MISSING",
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
			Action:   "WAIT",
			Reason:   "STRUCT_LAGGING",
			Priority: gatePriorityStructBreak,
			StopStep: "structure",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.StructureTag == "structure_broken" && ctx.Inputs.IndicatorTag == "divergence_reversal"
		},
	},
	{
		Step: "structure",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   "STRUCT_BREAK",
			Priority: gatePriorityStructBreak,
			StopStep: "structure",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.StructureTag == "structure_broken" || !ctx.Inputs.StructureIntegrity
		},
	},
}

var gateMechanicsStopRules = []gateDecisionRule{
	{
		Step: "mech_risk",
		Outcome: gateDecisionOutcome{
			Action:   "VETO",
			Reason:   "MECH_RISK",
			Priority: gatePriorityMechRisk,
			StopStep: "mech_risk",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.MechanicsTag == "liquidation_cascade"
		},
	},
	{
		Step: "mech_risk",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "MECH_RISK",
			Priority: gatePriorityMechRisk,
			StopStep: "mech_risk",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.LiquidationStress && ctx.Inputs.LiqConfidence != "low"
		},
	},
}

var gateNoiseStopRules = []gateDecisionRule{
	{
		Step: "indicator_noise",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "INDICATOR_NOISE",
			Priority: gatePriorityIndicatorNoise,
			StopStep: "indicator_noise",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Inputs.IndicatorTag == "noise"
		},
	},
	{
		Step: "structure_clear",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "INDICATOR_MIXED",
			Priority: gatePriorityIndicatorMixed,
			StopStep: "structure_clear",
		},
		Match: func(ctx gateDecisionContext) bool {
			return !ctx.Inputs.StructureClear
		},
	},
	{
		Step: "tag_consistency",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "INDICATOR_MIXED",
			Priority: gatePriorityTagInconsistent,
			StopStep: "tag_consistency",
		},
		Match: func(ctx gateDecisionContext) bool {
			return !resolveBoolTagConsistencyFromFlags(ctx.Inputs.IndicatorTag, ctx.Inputs.MomentumExpansion, ctx.Inputs.Alignment, ctx.Inputs.MeanRevNoise)
		},
	},
}

var gateScriptStopRules = []gateDecisionRule{
	{
		Step: "script_select",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "INDICATOR_MIXED",
			Priority: gatePriorityScriptMissing,
			StopStep: "script_select",
		},
		Match: func(ctx gateDecisionContext) bool {
			return ctx.Script == ""
		},
	},
	{
		Step: "script_allowed",
		Outcome: gateDecisionOutcome{
			Action:   "WAIT",
			Reason:   "INDICATOR_MIXED",
			Priority: gatePriorityScriptNotAllowed,
			StopStep: "script_allowed",
		},
		Match: func(ctx gateDecisionContext) bool {
			if ctx.Script == "" {
				return false
			}
			return !isEntryScriptAllowed(ctx.Script, ctx.Inputs.MomentumExpansion, ctx.Inputs.Alignment, ctx.Inputs.MeanRevNoise)
		},
	},
}
