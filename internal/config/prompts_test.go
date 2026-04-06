package config

import (
	"strings"
	"testing"
)

func TestDefaultPromptDefaultsIncludeMovementCalibrationAnchors(t *testing.T) {
	defaults := DefaultPromptDefaults()
	prompts := []string{
		defaults.AgentIndicator,
		defaults.AgentStructure,
		defaults.AgentMechanics,
	}
	for _, prompt := range prompts {
		if !strings.Contains(prompt, "校准锚点（必须参考）") {
			t.Fatalf("prompt missing calibration anchors:\n%s", prompt)
		}
		if !strings.Contains(prompt, "当前决策窗口（参见用户输入中的“决策窗口”字段）内") {
			t.Fatalf("prompt missing decision window reference:\n%s", prompt)
		}
	}
}

func TestDefaultPromptDefaultsIncludeRiskPromptConstraints(t *testing.T) {
	defaults := DefaultPromptDefaults()

	if !strings.Contains(defaults.RiskFlatInit, "0.5~3.0 倍 ATR") {
		t.Fatalf("flat init prompt missing ATR stop constraint:\n%s", defaults.RiskFlatInit)
	}
	if !strings.Contains(defaults.RiskFlatInit, "首档风险回报比 >= 1:1") {
		t.Fatalf("flat init prompt missing RR constraint:\n%s", defaults.RiskFlatInit)
	}
	if !strings.Contains(defaults.RiskFlatInit, "不得原样照搬") {
		t.Fatalf("flat init prompt missing anti-copy guidance:\n%s", defaults.RiskFlatInit)
	}

	if !strings.Contains(defaults.RiskTightenUpdate, "unrealized_pnl_pct") {
		t.Fatalf("tighten prompt missing unrealized_pnl_pct context:\n%s", defaults.RiskTightenUpdate)
	}
	if !strings.Contains(defaults.RiskTightenUpdate, "position_age_minutes") {
		t.Fatalf("tighten prompt missing position_age_minutes context:\n%s", defaults.RiskTightenUpdate)
	}
	if !strings.Contains(defaults.RiskTightenUpdate, "tp1_hit") {
		t.Fatalf("tighten prompt missing tp1_hit context:\n%s", defaults.RiskTightenUpdate)
	}
	if !strings.Contains(defaults.RiskTightenUpdate, "distance_to_liq_pct") {
		t.Fatalf("tighten prompt missing distance_to_liq_pct context:\n%s", defaults.RiskTightenUpdate)
	}
}
