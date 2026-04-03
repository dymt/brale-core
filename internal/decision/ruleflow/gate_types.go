package ruleflow

import "strings"

const (
	gateGradeNone   = 0
	gateGradeLow    = 1
	gateGradeMedium = 2
	gateGradeHigh   = 3

	gatePriorityDirection           = 0
	gatePriorityDataMissing         = 1
	gatePriorityStructInvalidation  = 2
	gatePriorityLiquidationCascade  = 3
	gatePriorityQualityTooLow       = 4
	gatePriorityEdgeTooLow          = 5
	gatePriorityAllow               = 10

	gateReasonDirectionUnclear   = "DIRECTION_UNCLEAR"
	gateReasonDataMissing        = "DATA_MISSING"
	gateReasonStructInvalidation = "STRUCT_HARD_INVALIDATION"
	gateReasonLiquidationCascade = "LIQUIDATION_CASCADE"
	gateReasonQualityTooLow      = "QUALITY_TOO_LOW"
	gateReasonEdgeTooLow         = "EDGE_TOO_LOW"
	gateReasonAllow              = "ALLOW"

	gateCategoryDirection = "direction"
	gateCategoryData      = "data"
	gateCategoryStructure = "structure"
	gateCategoryRisk      = "risk"
	gateCategoryQuality   = "quality"
	gateCategoryAllow     = "allow"
)

type gateInputs struct {
	State               string
	StructureDirection  string
	IndicatorTag        string
	StructureTag        string
	MechanicsTag        string
	MomentumExpansion   bool
	Alignment           bool
	MeanRevNoise        bool
	StructureClear      bool
	StructureIntegrity  bool
	LiquidationStress   bool
	LiqConfidence       string
	CrowdingAlign       bool
	ConsensusScore      float64
	ConsensusConfidence float64
	ConsensusAgreement  float64
	ConsensusResonance  float64
	ConsensusResonant   bool
	ScoreThreshold      float64
	ConfidenceThreshold float64
	QualityThreshold    float64
	EdgeThreshold       float64
}

type gateDecision struct {
	Action         string
	Reason         string
	ReasonCategory string
	Direction      string
	Grade          int
	Priority       int
	StopStep       string
	StopReason     string
	GateTrace      []map[string]any
	SetupQuality   float64
	RiskPenalty    float64
	EntryEdge      float64
	ScriptName     string
	ScriptBonus    float64
}

type gateDecisionContext struct {
	Inputs           gateInputs
	MissingProviders bool
}

type gateDecisionOutcome struct {
	Action   string
	Reason   string
	Priority int
	StopStep string
	Grade    int
}

type gateDecisionRule struct {
	Step    string
	Outcome gateDecisionOutcome
	Match   func(gateDecisionContext) bool
}

type gateDecisionEvaluator struct {
	inputs           gateInputs
	missingProviders bool
	decision         gateDecision
	gateTrace        []map[string]any
}

func (e *gateDecisionEvaluator) hasAction() bool {
	return strings.TrimSpace(e.decision.Action) != ""
}

func (e *gateDecisionEvaluator) appendGateTrace(step string, ok bool, code string) {
	entry := map[string]any{
		"step": step,
		"ok":   ok,
	}
	if strings.TrimSpace(code) != "" {
		entry["reason"] = code
	}
	e.gateTrace = append(e.gateTrace, entry)
}

func (e *gateDecisionEvaluator) setStop(step string, action string, code string, priority int) {
	e.decision.Action = action
	e.decision.Reason = code
	e.decision.Priority = priority
	e.decision.StopStep = step
	e.decision.StopReason = code
	e.appendGateTrace(step, false, code)
}

func (e *gateDecisionEvaluator) context() gateDecisionContext {
	return gateDecisionContext{
		Inputs:           e.inputs,
		MissingProviders: e.missingProviders,
	}
}

func (e *gateDecisionEvaluator) applyOutcome(outcome gateDecisionOutcome) {
	if strings.TrimSpace(outcome.Action) == "" {
		return
	}
	e.decision.Action = outcome.Action
	e.decision.Reason = outcome.Reason
	e.decision.Priority = outcome.Priority
	e.decision.StopStep = outcome.StopStep
	e.decision.StopReason = outcome.Reason
	if outcome.Grade > 0 {
		e.decision.Grade = outcome.Grade
	}
	e.appendGateTrace(outcome.StopStep, false, outcome.Reason)
}

func findFirstGateDecisionRule(rules []gateDecisionRule, ctx gateDecisionContext) (gateDecisionRule, bool) {
	for _, rule := range rules {
		if rule.Match != nil && rule.Match(ctx) {
			return rule, true
		}
	}
	return gateDecisionRule{}, false
}
