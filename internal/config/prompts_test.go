package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultAgentPromptsRemoveCalibrationAnchors(t *testing.T) {
	defaults := DefaultPromptDefaults()
	prompts := map[string]string{
		"indicator": defaults.AgentIndicator,
		"structure": defaults.AgentStructure,
		"mechanics": defaults.AgentMechanics,
	}

	for name, prompt := range prompts {
		if strings.Contains(prompt, "校准锚点（必须参考）") {
			t.Fatalf("%s prompt should not contain calibration anchors", name)
		}
		if !strings.Contains(prompt, "movement_score") {
			t.Fatalf("%s prompt should keep movement_score constraints", name)
		}
		if !strings.Contains(prompt, "movement_confidence") {
			t.Fatalf("%s prompt should keep movement_confidence constraints", name)
		}
	}
}

func TestAgentAndProviderPromptsShareCommonPreamble(t *testing.T) {
	for name, prompt := range promptSnapshotInputs() {
		if !strings.HasPrefix(prompt, agentOutputPreamble) {
			t.Fatalf("%s prompt should start with shared preamble", name)
		}
	}
}

func TestPromptDefaultsMatchSnapshots(t *testing.T) {
	snapshots := loadPromptSnapshots(t)
	for name, prompt := range promptSnapshotInputs() {
		if snapshots[name] != prompt {
			t.Fatalf("%s prompt snapshot mismatch", name)
		}
	}
}

func promptSnapshotInputs() map[string]string {
	defaults := DefaultPromptDefaults()
	return map[string]string{
		"agent_indicator":       defaults.AgentIndicator,
		"agent_structure":       defaults.AgentStructure,
		"agent_mechanics":       defaults.AgentMechanics,
		"provider_indicator":    defaults.ProviderIndicator,
		"provider_structure":    defaults.ProviderStructure,
		"provider_mechanics":    defaults.ProviderMechanics,
		"in_position_indicator": defaults.ProviderInPositionIndicator,
		"in_position_structure": defaults.ProviderInPositionStructure,
		"in_position_mechanics": defaults.ProviderInPositionMechanics,
	}
}

func loadPromptSnapshots(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("testdata", "prompt_snapshots", "default_prompts.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read prompt snapshots: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal prompt snapshots: %v", err)
	}
	return out
}
