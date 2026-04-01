package ruleflow

type gateStopRule struct {
	Step     string
	Action   string
	Code     string
	Priority int
	Match    func(gateInputs) bool
}

var gateStructureStopRules = []gateStopRule{
	{
		Step:     "structure",
		Action:   "WAIT",
		Code:     "STRUCT_LAGGING",
		Priority: gatePriorityStructBreak,
		Match: func(inputs gateInputs) bool {
			return inputs.StructureTag == "structure_broken" && inputs.IndicatorTag == "divergence_reversal"
		},
	},
	{
		Step:     "structure",
		Action:   "VETO",
		Code:     "STRUCT_BREAK",
		Priority: gatePriorityStructBreak,
		Match: func(inputs gateInputs) bool {
			return inputs.StructureTag == "structure_broken" || !inputs.StructureIntegrity
		},
	},
}

var gateMechanicsStopRules = []gateStopRule{
	{
		Step:     "mech_risk",
		Action:   "VETO",
		Code:     "MECH_RISK",
		Priority: gatePriorityMechRisk,
		Match: func(inputs gateInputs) bool {
			return inputs.MechanicsTag == "liquidation_cascade"
		},
	},
	{
		Step:     "mech_risk",
		Action:   "WAIT",
		Code:     "MECH_RISK",
		Priority: gatePriorityMechRisk,
		Match: func(inputs gateInputs) bool {
			return inputs.LiquidationStress && inputs.LiqConfidence != "low"
		},
	},
}

var gateNoiseStopRules = []gateStopRule{
	{
		Step:     "indicator_noise",
		Action:   "WAIT",
		Code:     "INDICATOR_NOISE",
		Priority: gatePriorityIndicatorNoise,
		Match: func(inputs gateInputs) bool {
			return inputs.IndicatorTag == "noise"
		},
	},
	{
		Step:     "structure_clear",
		Action:   "WAIT",
		Code:     "INDICATOR_MIXED",
		Priority: gatePriorityIndicatorMixed,
		Match: func(inputs gateInputs) bool {
			return !inputs.StructureClear
		},
	},
	{
		Step:     "tag_consistency",
		Action:   "WAIT",
		Code:     "INDICATOR_MIXED",
		Priority: gatePriorityTagInconsistent,
		Match: func(inputs gateInputs) bool {
			return !resolveBoolTagConsistencyFromFlags(inputs.IndicatorTag, inputs.MomentumExpansion, inputs.Alignment, inputs.MeanRevNoise)
		},
	},
}

func findFirstGateStopRule(rules []gateStopRule, inputs gateInputs) (gateStopRule, bool) {
	for _, rule := range rules {
		if rule.Match != nil && rule.Match(inputs) {
			return rule, true
		}
	}
	return gateStopRule{}, false
}
