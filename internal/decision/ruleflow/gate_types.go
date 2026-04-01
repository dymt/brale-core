package ruleflow

import "strings"

const (
	gateGradeNone   = 0
	gateGradeLow    = 1
	gateGradeMedium = 2
	gateGradeHigh   = 3

	gatePriorityConsensusFailed  = 0
	gatePriorityDataMissing      = 1
	gatePriorityStructBreak      = 1
	gatePriorityMechRisk         = 2
	gatePriorityIndicatorNoise   = 3
	gatePriorityIndicatorMixed   = 4
	gatePriorityTagInconsistent  = 5
	gatePriorityScriptMissing    = 6
	gatePriorityScriptNotAllowed = 7
	gatePrioritySieveOverride    = 8
	gatePriorityAllow            = 10
)

type gateInputs struct {
	State              string
	StructureDirection string
	IndicatorTag       string
	StructureTag       string
	MechanicsTag       string
	MomentumExpansion  bool
	Alignment          bool
	MeanRevNoise       bool
	StructureClear     bool
	StructureIntegrity bool
	LiquidationStress  bool
	LiqConfidence      string
}

type gateDecision struct {
	Action     string
	Reason     string
	Direction  string
	Grade      int
	Priority   int
	StopStep   string
	StopReason string
	GateTrace  []map[string]any
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
