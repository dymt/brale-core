package ruleflow

func evaluateGateDecision(inputs gateInputs, missingProviders bool) gateDecision {
	eval := gateDecisionEvaluator{
		inputs:           inputs,
		missingProviders: missingProviders,
		decision:         gateDecision{Direction: inputs.StructureDirection, Grade: gateGradeNone},
	}
	eval.evaluate()
	eval.decision.GateTrace = eval.gateTrace
	return eval.decision
}

func (e *gateDecisionEvaluator) evaluate() {
	e.evalDirection()
	e.evalData()
	e.evalStructure()
	e.evalMechRisk()
	e.evalIndicatorNoise()
	e.evalStructureClear()
	e.evalTagConsistency()
	e.evalScript()
}

func (e *gateDecisionEvaluator) evalDirection() {
	if e.hasAction() {
		return
	}
	if e.inputs.StructureDirection == "" || e.inputs.StructureDirection == "none" {
		e.decision.Direction = "none"
		e.setStop("direction", "VETO", "CONSENSUS_NOT_PASSED", gatePriorityConsensusFailed)
		return
	}
	e.appendGateTrace("direction", true, "")
}

func (e *gateDecisionEvaluator) evalData() {
	if e.hasAction() {
		return
	}
	if !e.missingProviders || e.inputs.State == "IN_POSITION" {
		e.appendGateTrace("data", true, "")
		return
	}
	e.setStop("data", "VETO", "DATA_MISSING", gatePriorityDataMissing)
}

func (e *gateDecisionEvaluator) evalStructure() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateStopRule(gateStructureStopRules, e.inputs); ok {
		e.setStop(rule.Step, rule.Action, rule.Code, rule.Priority)
		return
	}
	e.appendGateTrace("structure", true, "")
}

func (e *gateDecisionEvaluator) evalMechRisk() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateStopRule(gateMechanicsStopRules, e.inputs); ok {
		e.setStop(rule.Step, rule.Action, rule.Code, rule.Priority)
		return
	}
	e.appendGateTrace("mech_risk", true, "")
}

func (e *gateDecisionEvaluator) evalIndicatorNoise() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateStopRule(gateNoiseStopRules[:1], e.inputs); ok {
		e.setStop(rule.Step, rule.Action, rule.Code, rule.Priority)
		return
	}
	e.appendGateTrace("indicator_noise", true, "")
}

func (e *gateDecisionEvaluator) evalStructureClear() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateStopRule(gateNoiseStopRules[1:2], e.inputs); ok {
		e.setStop(rule.Step, rule.Action, rule.Code, rule.Priority)
		return
	}
	e.appendGateTrace("structure_clear", true, "")
}

func (e *gateDecisionEvaluator) evalTagConsistency() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateStopRule(gateNoiseStopRules[2:], e.inputs); ok {
		e.setStop(rule.Step, rule.Action, rule.Code, rule.Priority)
		return
	}
	e.appendGateTrace("tag_consistency", true, "")
}

func (e *gateDecisionEvaluator) evalScript() {
	if e.hasAction() {
		return
	}
	script := resolveEntryScript(e.inputs.IndicatorTag, e.inputs.StructureTag)
	if script == "" {
		e.setStop("script_select", "WAIT", "INDICATOR_MIXED", gatePriorityScriptMissing)
		return
	}
	e.appendGateTrace("script_select", true, "")
	if !isEntryScriptAllowed(script, e.inputs.MomentumExpansion, e.inputs.Alignment, e.inputs.MeanRevNoise) {
		e.setStop("script_allowed", "WAIT", "INDICATOR_MIXED", gatePriorityScriptNotAllowed)
		return
	}
	e.appendGateTrace("script_allowed", true, "")
	e.decision.Action = "ALLOW"
	e.decision.Reason = "PASS_STRONG"
	e.decision.Grade = resolveEntryGrade(script)
	e.decision.Priority = gatePriorityAllow
	e.decision.StopStep = "gate_allow"
	e.decision.StopReason = e.decision.Reason
}
