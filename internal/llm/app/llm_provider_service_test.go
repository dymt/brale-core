package llmapp

import (
	"context"
	"errors"
	"testing"

	"brale-core/internal/decision"
	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/provider"
	"brale-core/internal/llm"
	"brale-core/internal/prompt/positionprompt"
)

func TestLLMProviderServiceJudgeReturnsStageError(t *testing.T) {
	boom := errors.New("boom")
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: stubLLMProvider{err: boom}},
		Prompts: LLMPromptBuilder{
			ProviderStructureSystem: "provider-structure-system",
		},
	}

	_, _, _, _, err := svc.Judge(context.Background(), "ETHUSDT", agent.IndicatorSummary{}, agent.StructureSummary{}, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v want=%v", err, boom)
	}
	if got := err.Error(); got != "llm provider stage failed: symbol=ETHUSDT stage=structure: boom" {
		t.Fatalf("error=%q", got)
	}
}

func TestLLMProviderServiceJudgeInPositionReturnsStageError(t *testing.T) {
	boom := errors.New("boom")
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: stubLLMProvider{err: boom}},
		Prompts: LLMPromptBuilder{
			ProviderInPosStructureSys: "provider-in-pos-structure-system",
		},
	}

	_, _, _, _, err := svc.JudgeInPosition(context.Background(), "ETHUSDT", agent.IndicatorSummary{}, agent.StructureSummary{}, agent.MechanicsSummary{}, positionprompt.Summary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v want=%v", err, boom)
	}
	if got := err.Error(); got != "llm provider stage failed: symbol=ETHUSDT stage=structure_in_position: boom" {
		t.Fatalf("error=%q", got)
	}
}

func TestLLMProviderServiceJudgeUsesStatelessCalls(t *testing.T) {
	providerStub := &stubRiskSessionProvider{callResp: `{"clear_structure":true,"integrity":true,"reason":"ok","signal_tag":"support_retest"}`}
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: providerStub},
		Prompts: LLMPromptBuilder{
			ProviderStructureSystem: "provider-structure-system",
		},
	}
	roundID, err := llm.NewRoundID("round-provider-stateless")
	if err != nil {
		t.Fatalf("round id: %v", err)
	}
	ctx := llm.WithSessionRoundID(context.Background(), roundID)
	ctx = llm.WithSessionFlow(ctx, llm.LLMFlowFlat)
	_, stOut, _, _, err := svc.Judge(ctx, "ETHUSDT", agent.IndicatorSummary{}, agent.StructureSummary{Regime: agent.RegimeTrendUp, LastBreak: agent.LastBreakBosUp, Quality: agent.QualityClean, Pattern: agent.PatternFlag, VolumeAction: "vol ok", CandleReaction: "retest ok"}, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{})
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	if !stOut.ClearStructure || !stOut.Integrity {
		t.Fatalf("unexpected structure output: %+v", stOut)
	}
	if providerStub.callCount != 1 {
		t.Fatalf("call_count=%d, want 1", providerStub.callCount)
	}
}
