package ruleflow

type gateScriptCondition struct {
	MomentumExpansion *bool
	Alignment         *bool
	MeanRevNoise      *bool
}

type gateScriptRule struct {
	IndicatorTag string
	StructureTag string
	Script       string
	Grade        int
	Allow        gateScriptCondition
}

var gateScriptRules = []gateScriptRule{
	{
		IndicatorTag: "trend_surge",
		StructureTag: "breakout_confirmed",
		Script:       "A",
		Grade:        gateGradeHigh,
		Allow: gateScriptCondition{
			MomentumExpansion: boolPtr(true),
			Alignment:         boolPtr(true),
			MeanRevNoise:      boolPtr(false),
		},
	},
	{
		IndicatorTag: "pullback_entry",
		StructureTag: "support_retest",
		Script:       "B",
		Grade:        gateGradeMedium,
		Allow: gateScriptCondition{
			MomentumExpansion: boolPtr(false),
			Alignment:         boolPtr(true),
			MeanRevNoise:      boolPtr(false),
		},
	},
	{
		IndicatorTag: "divergence_reversal",
		StructureTag: "support_retest",
		Script:       "C",
		Grade:        gateGradeLow,
		Allow: gateScriptCondition{
			Alignment:    boolPtr(false),
			MeanRevNoise: boolPtr(false),
		},
	},
	{
		IndicatorTag: "trend_surge",
		StructureTag: "support_retest",
		Script:       "D",
		Grade:        gateGradeMedium,
		Allow: gateScriptCondition{
			MomentumExpansion: boolPtr(true),
			Alignment:         boolPtr(true),
			MeanRevNoise:      boolPtr(false),
		},
	},
	{
		IndicatorTag: "pullback_entry",
		StructureTag: "breakout_confirmed",
		Script:       "E",
		Grade:        gateGradeLow,
		Allow: gateScriptCondition{
			MomentumExpansion: boolPtr(false),
			Alignment:         boolPtr(true),
			MeanRevNoise:      boolPtr(false),
		},
	},
	{
		IndicatorTag: "divergence_reversal",
		StructureTag: "breakout_confirmed",
		Script:       "F",
		Grade:        gateGradeLow,
		Allow: gateScriptCondition{
			Alignment:    boolPtr(false),
			MeanRevNoise: boolPtr(false),
		},
	},
}

func resolveEntryScript(indicatorTag, structureTag string) string {
	if rule, ok := findGateScriptRuleByTags(indicatorTag, structureTag); ok {
		return rule.Script
	}
	return ""
}

func isEntryScriptAllowed(script string, momentumExpansion, alignment, meanRevNoise bool) bool {
	rule, ok := findGateScriptRuleByScript(script)
	if !ok {
		return false
	}
	return rule.Allow.matches(momentumExpansion, alignment, meanRevNoise)
}

func resolveEntryGrade(script string) int {
	rule, ok := findGateScriptRuleByScript(script)
	if !ok {
		return gateGradeNone
	}
	return rule.Grade
}

func findGateScriptRuleByTags(indicatorTag, structureTag string) (gateScriptRule, bool) {
	for _, rule := range gateScriptRules {
		if rule.IndicatorTag == indicatorTag && rule.StructureTag == structureTag {
			return rule, true
		}
	}
	return gateScriptRule{}, false
}

func findGateScriptRuleByScript(script string) (gateScriptRule, bool) {
	for _, rule := range gateScriptRules {
		if rule.Script == script {
			return rule, true
		}
	}
	return gateScriptRule{}, false
}

func (c gateScriptCondition) matches(momentumExpansion, alignment, meanRevNoise bool) bool {
	if c.MomentumExpansion != nil && *c.MomentumExpansion != momentumExpansion {
		return false
	}
	if c.Alignment != nil && *c.Alignment != alignment {
		return false
	}
	if c.MeanRevNoise != nil && *c.MeanRevNoise != meanRevNoise {
		return false
	}
	return true
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
