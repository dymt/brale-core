package ruleflow

import "testing"

func TestEvaluateGateDecisionUsesRuleTablesForStops(t *testing.T) {
	testCases := []struct {
		name             string
		inputs           gateInputs
		missingProviders bool
		wantAction       string
		wantReason       string
		wantPriority     int
		wantStep         string
	}{
		{
			name:         "direction veto",
			inputs:       gateInputs{StructureDirection: "none"},
			wantAction:   "VETO",
			wantReason:   "CONSENSUS_NOT_PASSED",
			wantPriority: gatePriorityConsensusFailed,
			wantStep:     "direction",
		},
		{
			name:             "data missing veto",
			inputs:           gateInputs{StructureDirection: "long"},
			missingProviders: true,
			wantAction:       "VETO",
			wantReason:       "DATA_MISSING",
			wantPriority:     gatePriorityDataMissing,
			wantStep:         "data",
		},
		{
			name: "structure lagging wait",
			inputs: gateInputs{
				StructureDirection: "long",
				IndicatorTag:       "divergence_reversal",
				StructureTag:       "structure_broken",
				StructureIntegrity: true,
			},
			wantAction:   "WAIT",
			wantReason:   "STRUCT_LAGGING",
			wantPriority: gatePriorityStructBreak,
			wantStep:     "structure",
		},
		{
			name: "mechanics risk veto",
			inputs: gateInputs{
				StructureDirection: "long",
				IndicatorTag:       "trend_surge",
				StructureTag:       "breakout_confirmed",
				MechanicsTag:       "liquidation_cascade",
				StructureClear:     true,
				StructureIntegrity: true,
				MomentumExpansion:  true,
				Alignment:          true,
			},
			wantAction:   "VETO",
			wantReason:   "MECH_RISK",
			wantPriority: gatePriorityMechRisk,
			wantStep:     "mech_risk",
		},
		{
			name: "script missing wait",
			inputs: gateInputs{
				StructureDirection: "long",
				IndicatorTag:       "trend_surge",
				StructureTag:       "unknown_structure",
				StructureClear:     true,
				StructureIntegrity: true,
				MomentumExpansion:  true,
				Alignment:          true,
				MeanRevNoise:       false,
			},
			wantAction:   "WAIT",
			wantReason:   "INDICATOR_MIXED",
			wantPriority: gatePriorityScriptMissing,
			wantStep:     "script_select",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decision := evaluateGateDecision(tc.inputs, tc.missingProviders)
			if decision.Action != tc.wantAction {
				t.Fatalf("action=%s want %s", decision.Action, tc.wantAction)
			}
			if decision.Reason != tc.wantReason {
				t.Fatalf("reason=%s want %s", decision.Reason, tc.wantReason)
			}
			if decision.Priority != tc.wantPriority {
				t.Fatalf("priority=%d want %d", decision.Priority, tc.wantPriority)
			}
			if decision.StopStep != tc.wantStep {
				t.Fatalf("stop_step=%s want %s", decision.StopStep, tc.wantStep)
			}
		})
	}
}

func TestEvaluateGateDecisionUsesRuleTableForAllowOutcome(t *testing.T) {
	decision := evaluateGateDecision(gateInputs{
		StructureDirection: "long",
		IndicatorTag:       "trend_surge",
		StructureTag:       "breakout_confirmed",
		MechanicsTag:       "neutral",
		MomentumExpansion:  true,
		Alignment:          true,
		MeanRevNoise:       false,
		StructureClear:     true,
		StructureIntegrity: true,
		LiqConfidence:      "low",
	}, false)

	if decision.Action != "ALLOW" {
		t.Fatalf("action=%s want ALLOW", decision.Action)
	}
	if decision.Reason != "PASS_STRONG" {
		t.Fatalf("reason=%s want PASS_STRONG", decision.Reason)
	}
	if decision.Priority != gatePriorityAllow {
		t.Fatalf("priority=%d want %d", decision.Priority, gatePriorityAllow)
	}
	if decision.Grade != gateGradeHigh {
		t.Fatalf("grade=%d want %d", decision.Grade, gateGradeHigh)
	}
	if decision.StopStep != "gate_allow" {
		t.Fatalf("stop_step=%s want gate_allow", decision.StopStep)
	}
	if decision.StopReason != "PASS_STRONG" {
		t.Fatalf("stop_reason=%s want PASS_STRONG", decision.StopReason)
	}
}
