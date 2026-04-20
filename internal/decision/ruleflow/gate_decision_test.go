package ruleflow

import "testing"

func TestGateVetoDirectionUnclear(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{StructureDirection: ""}, false)
	if decision.Action != "VETO" {
		t.Fatalf("action=%s want VETO", decision.Action)
	}
	if decision.Reason != "DIRECTION_UNCLEAR" {
		t.Fatalf("reason=%s want DIRECTION_UNCLEAR", decision.Reason)
	}
	if decision.StopStep != "direction" {
		t.Fatalf("stop_step=%s want direction", decision.StopStep)
	}
}

func TestGateVetoDataMissing(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{StructureDirection: "long"}, true)
	if decision.Action != "VETO" {
		t.Fatalf("action=%s want VETO", decision.Action)
	}
	if decision.Reason != "DATA_MISSING" {
		t.Fatalf("reason=%s want DATA_MISSING", decision.Reason)
	}
}

func TestGateVetoLiquidationCascade(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureTag:       "breakout_confirmed",
		StructureIntegrity: true,
		MechanicsTag:       "liquidation_cascade",
	}, false)
	if decision.Action != "VETO" {
		t.Fatalf("action=%s want VETO", decision.Action)
	}
	if decision.Reason != "LIQUIDATION_CASCADE" {
		t.Fatalf("reason=%s want LIQUIDATION_CASCADE", decision.Reason)
	}
}

func TestGateCanDisableStructureInvalidationHardStop(t *testing.T) {
	disabled := false
	decision := evaluateGateDecision(gateInputs{
		StructureDirection:            "long",
		StructureTag:                  "structure_broken",
		StructureIntegrity:            false,
		HardStopStructureInvalidation: &disabled,
		ConsensusScore:                0.6,
		QualityThreshold:              0.35,
		EdgeThreshold:                 0.10,
	}, false)
	if decision.Action == "VETO" {
		t.Fatalf("action=%s want non-VETO", decision.Action)
	}
	if decision.Reason != "QUALITY_TOO_LOW" {
		t.Fatalf("reason=%s want QUALITY_TOO_LOW", decision.Reason)
	}
}

func TestGateCanDisableLiquidationCascadeHardStop(t *testing.T) {
	disabled := false
	decision := evaluateGateDecision(gateInputs{
		StructureDirection:         "long",
		StructureClear:             true,
		StructureIntegrity:         true,
		Alignment:                  true,
		MomentumExpansion:          true,
		IndicatorTag:               "trend_surge",
		StructureTag:               "breakout_confirmed",
		MechanicsTag:               "liquidation_cascade",
		HardStopLiquidationCascade: &disabled,
		ConsensusScore:             0.7,
		QualityThreshold:           0.35,
		EdgeThreshold:              0.10,
	}, false)
	if decision.Action == "VETO" {
		t.Fatalf("action=%s want non-VETO", decision.Action)
	}
	if decision.Reason != "EDGE_TOO_LOW" {
		t.Fatalf("reason=%s want EDGE_TOO_LOW", decision.Reason)
	}
}

func TestGateWaitQualityTooLow(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureIntegrity: true,
		MeanRevNoise:       true,
		ConsensusScore:     0.5,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "WAIT" {
		t.Fatalf("action=%s want WAIT", decision.Action)
	}
	if decision.Reason != "QUALITY_TOO_LOW" {
		t.Fatalf("reason=%s want QUALITY_TOO_LOW", decision.Reason)
	}
}

func TestGateWaitEdgeTooLow(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: true,
		Alignment:          true,
		ConsensusScore:     0.05,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "WAIT" {
		t.Fatalf("action=%s want WAIT", decision.Action)
	}
	if decision.Reason != "EDGE_TOO_LOW" {
		t.Fatalf("reason=%s want EDGE_TOO_LOW", decision.Reason)
	}
}

func TestGateAllowWithGradeFromQuality(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: true,
		Alignment:          true,
		MomentumExpansion:  true,
		IndicatorTag:       "trend_surge",
		StructureTag:       "breakout_confirmed",
		ConsensusScore:     0.7,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "ALLOW" {
		t.Fatalf("action=%s want ALLOW", decision.Action)
	}
	if decision.Reason != "ALLOW" {
		t.Fatalf("reason=%s want ALLOW", decision.Reason)
	}
	if decision.Grade != gateGradeHigh {
		t.Fatalf("grade=%d want %d", decision.Grade, gateGradeHigh)
	}
	if decision.SetupQuality < 0.75 {
		t.Fatalf("setup_quality=%v want >= 0.75", decision.SetupQuality)
	}
}

func TestGateNoiseReducesQualityNotHardVeto(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: true,
		Alignment:          true,
		MomentumExpansion:  true,
		MeanRevNoise:       true,
		ConsensusScore:     0.7,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "ALLOW" {
		t.Fatalf("action=%s want ALLOW", decision.Action)
	}
}

func TestGateTagInconsistencyPenaltyCanTriggerWait(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: true,
		IndicatorTag:       "trend_surge",
		MomentumExpansion:  false,
		Alignment:          false,
		MeanRevNoise:       false,
		ConsensusScore:     0.8,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "WAIT" {
		t.Fatalf("action=%s want WAIT", decision.Action)
	}
	if decision.Reason != "QUALITY_TOO_LOW" {
		t.Fatalf("reason=%s want QUALITY_TOO_LOW", decision.Reason)
	}
}

func TestGateFakeoutRejectionWithIntegrityTrueAllowed(t *testing.T) {
	// With new prompt contract: fakeout_rejection where thesis held → integrity=true.
	// Gate should NOT veto this — structure was tested but recovered.
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: true, // new contract: fakeout recovery = thesis still valid
		StructureTag:       "fakeout_rejection",
		IndicatorTag:       "pullback_entry",
		Alignment:          true,
		ConsensusScore:     0.50,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action == "VETO" {
		t.Fatalf("fakeout_rejection with integrity=true should not be vetoed, got action=%s reason=%s", decision.Action, decision.Reason)
	}
}

func TestGateFakeoutRejectionWithIntegrityFalseVetoed(t *testing.T) {
	// fakeout_rejection but thesis actually broken → integrity=false → should veto.
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		StructureClear:     true,
		StructureIntegrity: false, // thesis failed despite fakeout tag
		StructureTag:       "fakeout_rejection",
		IndicatorTag:       "pullback_entry",
		Alignment:          true,
		ConsensusScore:     0.50,
		QualityThreshold:   0.35,
		EdgeThreshold:      0.10,
	}, false)
	if decision.Action != "VETO" {
		t.Fatalf("fakeout_rejection with integrity=false should veto, got action=%s", decision.Action)
	}
}

func TestGateAllowsStructureBreakContinuation(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		State:               "FLAT",
		StructureDirection:  "short",
		IndicatorTag:        "trend_surge",
		StructureTag:        "structure_broken",
		MechanicsTag:        "neutral",
		MomentumExpansion:   true,
		Alignment:           true,
		MeanRevNoise:        false,
		StructureClear:      false,
		StructureIntegrity:  false,
		LiqConfidence:       "low",
		ConsensusScore:      -0.66,
		ConsensusConfidence: 0.61,
		ConsensusResonance:  0.08,
		ConsensusResonant:   true,
		ScoreThreshold:      0.60,
		ConfidenceThreshold: 0.52,
		QualityThreshold:    0.10,
		EdgeThreshold:       0.05,
	}, false)
	if decision.Action != "ALLOW" {
		t.Fatalf("action=%s want ALLOW", decision.Action)
	}
}
