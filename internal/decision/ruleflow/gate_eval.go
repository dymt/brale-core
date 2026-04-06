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
	e.evalLiquidationCascade()
	if e.hasAction() {
		return
	}
	e.computeScores()
	e.evalQuality()
	e.evalEdge()
	if !e.hasAction() {
		e.allow()
	}
}

func (e *gateDecisionEvaluator) evalDirection() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateDecisionRule(gateDirectionRules, e.context()); ok {
		e.decision.Direction = "none"
		e.decision.ReasonCategory = gateCategoryDirection
		e.applyOutcome(rule.Outcome)
		return
	}
	e.appendGateTrace("direction", true, "")
}

func (e *gateDecisionEvaluator) evalData() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateDecisionRule(gateDataRules, e.context()); ok {
		e.decision.ReasonCategory = gateCategoryData
		e.applyOutcome(rule.Outcome)
		return
	}
	e.appendGateTrace("data", true, "")
}

func (e *gateDecisionEvaluator) evalStructure() {
	if e.hasAction() {
		return
	}
	if rule, ok := findFirstGateDecisionRule(gateStructureStopRules, e.context()); ok {
		e.decision.ReasonCategory = gateCategoryStructure
		e.applyOutcome(rule.Outcome)
		return
	}
	e.appendGateTrace("structure", true, "")
}

func (e *gateDecisionEvaluator) evalLiquidationCascade() {
	if e.hasAction() {
		return
	}
	if e.inputs.MechanicsTag == "liquidation_cascade" {
		e.decision.Action = "VETO"
		e.decision.Reason = gateReasonLiquidationCascade
		e.decision.ReasonCategory = gateCategoryRisk
		e.decision.Priority = gatePriorityLiquidationCascade
		e.decision.StopStep = "liquidation_cascade"
		e.decision.StopReason = gateReasonLiquidationCascade
		e.appendGateTrace("liquidation_cascade", false, gateReasonLiquidationCascade)
		return
	}
	e.appendGateTrace("liquidation_cascade", true, "")
}

func (e *gateDecisionEvaluator) computeScores() {
	scriptBonus, scriptName := resolveScriptBonus(e.inputs.IndicatorTag, e.inputs.StructureTag)
	e.decision.ScriptBonus = scriptBonus
	e.decision.ScriptName = scriptName
	e.decision.SetupQuality = computeSetupQuality(
		e.inputs.StructureClear,
		e.inputs.StructureIntegrity,
		e.inputs.Alignment,
		e.inputs.MomentumExpansion,
		e.inputs.MeanRevNoise,
		scriptBonus,
		e.inputs.IndicatorTag,
		e.inputs.AgentIndicatorConfidence,
		e.inputs.AgentStructureConfidence,
	)
	e.decision.RiskPenalty = computeRiskPenalty(
		e.inputs.MechanicsTag,
		e.inputs.LiquidationStress,
		e.inputs.LiqConfidence,
		e.inputs.CrowdingAlign,
		e.inputs.AgentMechanicsConfidence,
	)
	e.decision.EntryEdge = computeEntryEdge(
		e.inputs.ConsensusScore,
		e.decision.SetupQuality,
		e.decision.RiskPenalty,
	)
}

func (e *gateDecisionEvaluator) evalQuality() {
	if e.hasAction() {
		return
	}
	threshold := e.inputs.QualityThreshold
	if threshold <= 0 {
		threshold = 0.35
	}
	if e.decision.SetupQuality < threshold {
		e.decision.Action = "WAIT"
		e.decision.Reason = gateReasonQualityTooLow
		e.decision.ReasonCategory = gateCategoryQuality
		e.decision.Priority = gatePriorityQualityTooLow
		e.decision.StopStep = "quality"
		e.decision.StopReason = gateReasonQualityTooLow
		e.appendGateTrace("quality", false, gateReasonQualityTooLow)
		return
	}
	e.appendGateTrace("quality", true, "")
}

func (e *gateDecisionEvaluator) evalEdge() {
	if e.hasAction() {
		return
	}
	threshold := e.inputs.EdgeThreshold
	if threshold <= 0 {
		threshold = 0.10
	}
	if e.decision.EntryEdge < threshold {
		e.decision.Action = "WAIT"
		e.decision.Reason = gateReasonEdgeTooLow
		e.decision.ReasonCategory = gateCategoryQuality
		e.decision.Priority = gatePriorityEdgeTooLow
		e.decision.StopStep = "edge"
		e.decision.StopReason = gateReasonEdgeTooLow
		e.appendGateTrace("edge", false, gateReasonEdgeTooLow)
		return
	}
	e.appendGateTrace("edge", true, "")
}

func (e *gateDecisionEvaluator) allow() {
	e.decision.Action = "ALLOW"
	e.decision.Reason = gateReasonAllow
	e.decision.ReasonCategory = gateCategoryAllow
	e.decision.Priority = gatePriorityAllow
	e.decision.Grade = resolveGradeFromQuality(e.decision.SetupQuality)
	e.decision.StopStep = "gate_allow"
	e.decision.StopReason = gateReasonAllow
	e.appendGateTrace("gate_allow", true, gateReasonAllow)
}
