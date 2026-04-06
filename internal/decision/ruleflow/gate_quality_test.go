package ruleflow

import (
	"math"
	"testing"
)

func TestComputeSetupQuality(t *testing.T) {
	tests := []struct {
		name                        string
		clear, integrity, alignment bool
		momentum, noise             bool
		scriptBonus                 float64
		wantMin, wantMax            float64
	}{
		{
			name:        "perfect setup with best script",
			clear:       true,
			integrity:   true,
			alignment:   true,
			momentum:    true,
			noise:       false,
			scriptBonus: 1.0,
			wantMin:     0.95,
			wantMax:     1.0,
		},
		{
			name:        "worst case clamps to zero",
			clear:       false,
			integrity:   false,
			alignment:   false,
			momentum:    false,
			noise:       true,
			scriptBonus: 0.0,
			wantMin:     0.0,
			wantMax:     0.0,
		},
		{
			name:        "mixed structure with noise",
			clear:       true,
			integrity:   true,
			alignment:   false,
			momentum:    false,
			noise:       true,
			scriptBonus: 0.0,
			wantMin:     0.30,
			wantMax:     0.40,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeSetupQuality(tc.clear, tc.integrity, tc.alignment, tc.momentum, tc.noise, tc.scriptBonus, "", 1.0, 1.0)
			if got < tc.wantMin || got > tc.wantMax {
				t.Fatalf("setup_quality=%v want [%v, %v]", got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func TestComputeSetupQualityAppliesTagConsistencyPenalty(t *testing.T) {
	// trend_surge with alignment but no momentum → soft conflict (0.10 penalty)
	got := computeSetupQuality(true, true, true, false, false, 1.0, "trend_surge", 1.0, 1.0)
	if math.Abs(got-0.75) > 0.001 {
		t.Fatalf("setup_quality=%v want ~0.75 (soft penalty)", got)
	}
	// trend_surge with mean_rev_noise → hard conflict (0.25 penalty)
	got = computeSetupQuality(true, true, true, true, true, 1.0, "trend_surge", 1.0, 1.0)
	if math.Abs(got-0.55) > 0.001 {
		t.Fatalf("setup_quality=%v want ~0.55 (hard penalty)", got)
	}
}

func TestComputeSetupQualityWeightsByAgentConfidence(t *testing.T) {
	highConfidence := computeSetupQuality(true, true, true, false, false, 0.7, "pullback_entry", 0.9, 0.9)
	lowConfidence := computeSetupQuality(true, true, true, false, false, 0.7, "pullback_entry", 0.1, 0.1)

	if math.Abs(highConfidence-0.745) > 0.001 {
		t.Fatalf("high confidence setup_quality=%v want ~0.745", highConfidence)
	}
	if math.Abs(lowConfidence-0.295) > 0.001 {
		t.Fatalf("low confidence setup_quality=%v want ~0.295", lowConfidence)
	}
}

func TestComputeRiskPenalty(t *testing.T) {
	tests := []struct {
		name          string
		mechTag       string
		stress        bool
		confidence    string
		align         bool
		mechanicsConf float64
		want          float64
	}{
		{"liquidation_cascade", "liquidation_cascade", false, "", false, 1.0, 1.00},
		{"stress_high", "crowded_long", true, "high", true, 1.0, 0.60},
		{"stress_low", "neutral", true, "low", false, 1.0, 0.35},
		{"crowding_aligned", "crowded_long", false, "", true, 1.0, 0.25},
		{"crowding_not_aligned", "crowded_long", false, "", false, 1.0, 0.10},
		{"fuel_ready", "fuel_ready", false, "", false, 1.0, 0.00},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeRiskPenalty(tc.mechTag, tc.stress, tc.confidence, tc.align, tc.mechanicsConf)
			if got != tc.want {
				t.Fatalf("risk_penalty=%v want %v", got, tc.want)
			}
		})
	}
}

func TestComputeRiskPenaltyWeightsByMechanicsConfidence(t *testing.T) {
	highConfidence := computeRiskPenalty("crowded_long", true, "high", true, 1.0)
	lowConfidence := computeRiskPenalty("crowded_long", true, "high", true, 0.1)

	if highConfidence != 0.60 {
		t.Fatalf("high confidence risk_penalty=%v want 0.60", highConfidence)
	}
	if lowConfidence != 0.18 {
		t.Fatalf("low confidence risk_penalty=%v want 0.18", lowConfidence)
	}
}

func TestComputeSetupQualityMatchesLegacyWhenConfidenceIsOne(t *testing.T) {
	got := computeSetupQuality(true, true, true, true, false, 1.0, "trend_surge", 1.0, 1.0)
	if math.Abs(got-1.0) > 0.001 {
		t.Fatalf("setup_quality=%v want legacy-equivalent 1.0", got)
	}
}

func TestComputeEntryEdge(t *testing.T) {
	got := computeEntryEdge(0.6, 0.8, 0.25)
	if math.Abs(got-0.36) > 0.001 {
		t.Fatalf("entry_edge=%v want ~0.36", got)
	}
	got = computeEntryEdge(0.9, 0.9, 1.0)
	if got != 0 {
		t.Fatalf("entry_edge=%v want 0", got)
	}
}

func TestResolveGradeFromQuality(t *testing.T) {
	tests := []struct {
		quality float64
		want    int
	}{
		{0.80, gateGradeHigh},
		{0.75, gateGradeHigh},
		{0.60, gateGradeMedium},
		{0.55, gateGradeMedium},
		{0.40, gateGradeLow},
		{0.35, gateGradeLow},
		{0.20, gateGradeNone},
		{0.00, gateGradeNone},
	}
	for _, tc := range tests {
		got := resolveGradeFromQuality(tc.quality)
		if got != tc.want {
			t.Fatalf("quality=%v grade=%v want %v", tc.quality, got, tc.want)
		}
	}
}

func TestResolveScriptBonus(t *testing.T) {
	bonus, name := resolveScriptBonus("trend_surge", "breakout_confirmed")
	if bonus != 1.0 || name != "A" {
		t.Fatalf("bonus=%v name=%s want 1.0/A", bonus, name)
	}
	bonus, name = resolveScriptBonus("unknown", "unknown")
	if bonus != 0 || name != "" {
		t.Fatalf("bonus=%v name=%s want 0/empty", bonus, name)
	}
}

func TestResolveScriptBonusFakeoutRejection(t *testing.T) {
	bonus, name := resolveScriptBonus("pullback_entry", "fakeout_rejection")
	if bonus != 0 || name != "FR" {
		t.Fatalf("bonus=%v name=%s want 0/FR", bonus, name)
	}
	bonus, name = resolveScriptBonus("divergence_reversal", "fakeout_rejection")
	if bonus != 0 || name != "FR" {
		t.Fatalf("bonus=%v name=%s want 0/FR", bonus, name)
	}
}

func TestResolveScriptBonusMomentumWeak(t *testing.T) {
	bonus, name := resolveScriptBonus("momentum_weak", "support_retest")
	if bonus != 0 || name != "MW" {
		t.Fatalf("bonus=%v name=%s want 0/MW", bonus, name)
	}
}

func TestResolveIndicatorTagConsistencyMomentumWeak(t *testing.T) {
	// momentum_weak is always consistent — no penalty regardless of booleans
	if !resolveIndicatorTagConsistency("momentum_weak", true, true, false) {
		t.Fatal("momentum_weak should always be consistent")
	}
	if !resolveIndicatorTagConsistency("momentum_weak", false, false, true) {
		t.Fatal("momentum_weak should always be consistent")
	}
}

func TestResolveConsistencyPenaltyGrading(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		momentum bool
		align    bool
		noise    bool
		want     float64
	}{
		// consistent → no penalty
		{"trend_surge_consistent", "trend_surge", true, true, false, 0},
		{"pullback_consistent", "pullback_entry", false, true, false, 0},
		{"momentum_weak_always_ok", "momentum_weak", false, false, true, 0},
		// hard conflicts
		{"trend_surge_noise", "trend_surge", true, true, true, consistencyPenaltyHard},
		{"trend_surge_no_align", "trend_surge", true, false, false, consistencyPenaltyHard},
		// soft conflicts
		{"trend_surge_no_momentum_only", "trend_surge", false, true, false, consistencyPenaltySoft},
		{"pullback_with_expansion", "pullback_entry", true, true, false, consistencyPenaltySoft},
		{"divergence_with_alignment", "divergence_reversal", true, true, false, consistencyPenaltySoft},
		{"noise_without_noise_flag", "noise", false, false, false, consistencyPenaltySoft},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveConsistencyPenalty(tc.tag, tc.momentum, tc.align, tc.noise)
			if got != tc.want {
				t.Fatalf("penalty=%v want %v", got, tc.want)
			}
		})
	}
}
