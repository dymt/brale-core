package ruleflow

import (
	"math"
	"strings"

	"brale-core/internal/decision/direction"
	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/fund"
	"brale-core/internal/strategy"
)

type GateReplayInput struct {
	State                    fsm.PositionState
	Providers                fund.ProviderBundle
	AgentIndicatorConfidence float64
	AgentStructureConfidence float64
	AgentMechanicsConfidence float64
	ConsensusScore           float64
	ConsensusConfidence      float64
	ConsensusAgreement       float64
	ConsensusCoverage        float64
	ConsensusResonance       float64
	ConsensusResonant        bool
	ScoreThreshold           float64
	ConfidenceThreshold      float64
	Binding                  strategy.StrategyBinding
	CurrentPrice             float64
}

func ReplayGateDecision(input GateReplayInput) fund.GateDecision {
	scoreThreshold := input.ScoreThreshold
	if scoreThreshold <= 0 {
		scoreThreshold = direction.ThresholdScore()
	}
	confidenceThreshold := input.ConfidenceThreshold
	if confidenceThreshold <= 0 {
		confidenceThreshold = direction.ThresholdConfidence()
	}
	structureDirection := resolveReplayStructureDirection(input.ConsensusScore, input.ConsensusConfidence, scoreThreshold, confidenceThreshold)
	indicatorTag := strings.ToLower(strings.TrimSpace(input.Providers.Indicator.SignalTag))
	structureTag := strings.ToLower(strings.TrimSpace(input.Providers.Structure.SignalTag))
	mechanicsTag := strings.ToLower(strings.TrimSpace(input.Providers.Mechanics.SignalTag))
	liqConfidence := strings.ToLower(strings.TrimSpace(string(input.Providers.Mechanics.LiquidationStress.Confidence)))
	crowdingAlign := (structureDirection == "long" && mechanicsTag == "crowded_long") || (structureDirection == "short" && mechanicsTag == "crowded_short")
	inputs := gateInputs{
		State:                         strings.ToUpper(strings.TrimSpace(string(input.State))),
		StructureDirection:            structureDirection,
		IndicatorTag:                  indicatorTag,
		StructureTag:                  structureTag,
		MechanicsTag:                  mechanicsTag,
		HardStopStructureInvalidation: toOptionalBool(input.Binding.RiskManagement.Gate.HardStop.StructureInvalidationEnabled()),
		HardStopLiquidationCascade:    toOptionalBool(input.Binding.RiskManagement.Gate.HardStop.LiquidationCascadeEnabled()),
		MomentumExpansion:             input.Providers.Indicator.MomentumExpansion,
		Alignment:                     input.Providers.Indicator.Alignment,
		MeanRevNoise:                  input.Providers.Indicator.MeanRevNoise,
		StructureClear:                input.Providers.Structure.ClearStructure,
		StructureIntegrity:            input.Providers.Structure.Integrity,
		LiquidationStress:             input.Providers.Mechanics.LiquidationStress.Value,
		LiqConfidence:                 liqConfidence,
		CrowdingAlign:                 crowdingAlign,
		AgentIndicatorConfidence:      input.AgentIndicatorConfidence,
		AgentStructureConfidence:      input.AgentStructureConfidence,
		AgentMechanicsConfidence:      input.AgentMechanicsConfidence,
		ConsensusScore:                input.ConsensusScore,
		ConsensusConfidence:           input.ConsensusConfidence,
		ConsensusAgreement:            input.ConsensusAgreement,
		ConsensusResonance:            input.ConsensusResonance,
		ConsensusResonant:             input.ConsensusResonant,
		ScoreThreshold:                scoreThreshold,
		ConfidenceThreshold:           confidenceThreshold,
		QualityThreshold:              input.Binding.RiskManagement.Gate.QualityThreshold,
		EdgeThreshold:                 input.Binding.RiskManagement.Gate.EdgeThreshold,
	}
	missingProviders := !input.Providers.Enabled.Indicator || !input.Providers.Enabled.Structure || !input.Providers.Enabled.Mechanics
	decision := evaluateGateDecision(inputs, missingProviders)
	gateActionBeforeSieve := decision.Action
	var sieveDecision sieveDecision
	if decision.Action == "ALLOW" {
		sieveDecision = resolveSieveDecision(replayRiskManagementMap(input.Binding), mechanicsTag, liqConfidence, crowdingAlign)
		if sieveAction := strings.ToUpper(strings.TrimSpace(sieveDecision.Action)); sieveAction != "" && sieveAction != "ALLOW" {
			decision.Action = sieveAction
			decision.Grade = gateGradeNone
			decision.ReasonCategory = gateCategoryRisk
			if reason := strings.TrimSpace(sieveDecision.Reason); reason != "" {
				decision.Reason = reason
			}
		}
	}
	ruleHit := &fund.GateRuleHit{
		Name:      decision.Reason,
		Priority:  decision.Priority,
		Action:    decision.Action,
		Reason:    decision.Reason,
		Grade:     decision.Grade,
		Direction: decision.Direction,
		Default:   false,
	}
	derived := map[string]any{
		"indicator_tag":                            indicatorTag,
		"structure_tag":                            structureTag,
		"mechanics_tag":                            mechanicsTag,
		"crowding_align":                           crowdingAlign,
		"consensus_score":                          input.ConsensusScore,
		"consensus_confidence":                     input.ConsensusConfidence,
		"consensus_agreement":                      input.ConsensusAgreement,
		"consensus_coverage":                       input.ConsensusCoverage,
		"consensus_resonance_bonus":                input.ConsensusResonance,
		"consensus_resonant":                       input.ConsensusResonant,
		"consensus_score_threshold":                scoreThreshold,
		"consensus_confidence_threshold":           confidenceThreshold,
		"gate_trace":                               decision.GateTrace,
		"gate_stop_step":                           decision.StopStep,
		"gate_stop_reason":                         decision.StopReason,
		"setup_quality":                            decision.SetupQuality,
		"risk_penalty":                             decision.RiskPenalty,
		"entry_edge":                               decision.EntryEdge,
		"gate_reason_category":                     decision.ReasonCategory,
		"gate_reason_code":                         decision.Reason,
		"script_name":                              decision.ScriptName,
		"script_bonus":                             decision.ScriptBonus,
		"quality_threshold":                        input.Binding.RiskManagement.Gate.QualityThreshold,
		"edge_threshold":                           input.Binding.RiskManagement.Gate.EdgeThreshold,
		"hard_stop_structure_invalidation_enabled": input.Binding.RiskManagement.Gate.HardStop.StructureInvalidationEnabled(),
		"hard_stop_liquidation_cascade_enabled":    input.Binding.RiskManagement.Gate.HardStop.LiquidationCascadeEnabled(),
		"gate_action_before_sieve":                 gateActionBeforeSieve,
		"sieve_action":                             sieveDecision.Action,
		"sieve_size_factor":                        sieveDecision.SizeFactor,
		"sieve_reason":                             sieveDecision.Reason,
		"sieve_hit":                                sieveDecision.Hit,
		"sieve_min_size_factor":                    sieveDecision.MinSizeFactor,
		"sieve_default_action":                     sieveDecision.DefaultAction,
		"sieve_default_size_factor":                sieveDecision.DefaultSizeFactor,
		"sieve_policy_hash":                        input.Binding.StrategyHash,
		"direction_consensus": map[string]any{
			"score":                input.ConsensusScore,
			"confidence":           input.ConsensusConfidence,
			"agreement":            input.ConsensusAgreement,
			"coverage":             input.ConsensusCoverage,
			"direction":            structureDirection,
			"resonance_bonus":      input.ConsensusResonance,
			"resonance_active":     input.ConsensusResonant,
			"score_threshold":      scoreThreshold,
			"confidence_threshold": confidenceThreshold,
			"score_passed":         math.Abs(input.ConsensusScore) >= scoreThreshold,
			"confidence_passed":    input.ConsensusConfidence >= confidenceThreshold,
			"passed":               direction.IsConsensusPassedWithThresholds(input.ConsensusScore, input.ConsensusConfidence, scoreThreshold, confidenceThreshold),
			"sources": map[string]any{
				"indicator": map[string]any{"confidence": input.AgentIndicatorConfidence, "raw_confidence": input.AgentIndicatorConfidence},
				"structure": map[string]any{"confidence": input.AgentStructureConfidence, "raw_confidence": input.AgentStructureConfidence},
				"mechanics": map[string]any{"confidence": input.AgentMechanicsConfidence, "raw_confidence": input.AgentMechanicsConfidence},
			},
		},
	}
	if input.CurrentPrice > 0 {
		derived["current_price"] = input.CurrentPrice
	}
	return fund.GateDecision{
		GlobalTradeable: decision.Action == "ALLOW",
		DecisionAction:  decision.Action,
		GateReason:      decision.Reason,
		Direction:       decision.Direction,
		Grade:           decision.Grade,
		RuleHit:         ruleHit,
		Derived:         derived,
	}
}

func resolveReplayStructureDirection(score, confidence, scoreThreshold, confidenceThreshold float64) string {
	if !direction.IsConsensusPassedWithThresholds(score, confidence, scoreThreshold, confidenceThreshold) {
		return "none"
	}
	if score > 0 {
		return "long"
	}
	if score < 0 {
		return "short"
	}
	return "none"
}

func replayRiskManagementMap(input strategy.StrategyBinding) map[string]any {
	rows := make([]any, 0, len(input.RiskManagement.Sieve.Rows))
	for _, row := range input.RiskManagement.Sieve.Rows {
		rowMap := map[string]any{
			"mechanics_tag":  row.MechanicsTag,
			"liq_confidence": row.LiqConfidence,
			"gate_action":    row.GateAction,
			"size_factor":    row.SizeFactor,
			"reason_code":    row.ReasonCode,
		}
		if row.CrowdingAlign != nil {
			rowMap["crowding_align"] = *row.CrowdingAlign
		}
		rows = append(rows, rowMap)
	}
	return map[string]any{
		"sieve": map[string]any{
			"min_size_factor":     input.RiskManagement.Sieve.MinSizeFactor,
			"default_gate_action": input.RiskManagement.Sieve.DefaultGateAction,
			"default_size_factor": input.RiskManagement.Sieve.DefaultSizeFactor,
			"rows":                rows,
		},
	}
}
