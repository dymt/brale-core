package config

import (
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
