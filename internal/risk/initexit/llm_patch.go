package initexit

import (
	"slices"

	"brale-core/internal/execution"
)

// BuildPatch is a narrow patch surface reserved for LLM post-processing.
type BuildPatch struct {
	Entry            *float64
	StopLoss         *float64
	TakeProfits      []float64
	TakeProfitRatios []float64
	Reason           *string
	Trace            *execution.LLMRiskTrace
}

func ApplyPatch(base BuildOutput, patch *BuildPatch) BuildOutput {
	if patch == nil {
		return base
	}
	if patch.StopLoss != nil {
		base.StopLoss = *patch.StopLoss
	}
	if len(patch.TakeProfits) > 0 {
		base.TakeProfits = slices.Clone(patch.TakeProfits)
	}
	if len(patch.TakeProfitRatios) > 0 {
		base.TakeProfitRatios = slices.Clone(patch.TakeProfitRatios)
	}
	return base
}
