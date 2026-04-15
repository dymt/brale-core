package llmapp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"brale-core/internal/decision"
	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/provider"
	"brale-core/internal/llm"
	"brale-core/internal/memory"
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

func TestLLMProviderServiceJudgeBypassesCacheWhenWorkingMemoryContextChanges(t *testing.T) {
	providerStub := &stubRiskSessionProvider{callResp: `{"clear_structure":true,"integrity":true,"reason":"ok","signal_tag":"support_retest"}`}
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: providerStub},
		Prompts: LLMPromptBuilder{
			ProviderStructureSystem: "provider-structure-system",
			UserFormat:              UserPromptFormatBullet,
		},
		Cache: NewLLMStageCache(),
	}
	input := agent.StructureSummary{
		Regime:         agent.RegimeTrendUp,
		LastBreak:      agent.LastBreakBosUp,
		Quality:        agent.QualityClean,
		Pattern:        agent.PatternFlag,
		VolumeAction:   "vol ok",
		CandleReaction: "retest ok",
	}

	if _, _, _, _, err := svc.Judge(context.Background(), "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() error = %v", err)
	}
	ctx := memory.WithPromptContext(context.Background(), "- [1h前] dir=short")
	if _, _, _, _, err := svc.Judge(ctx, "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() with memory error = %v", err)
	}

	if providerStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when memory context changes", providerStub.callCount)
	}
	if !strings.Contains(providerStub.lastUser, "近期决策记忆（仅供参考，禁止直接复用结论）") {
		t.Fatalf("user prompt missing memory block: %s", providerStub.lastUser)
	}
	if !strings.Contains(providerStub.lastUser, "dir=short") {
		t.Fatalf("user prompt missing memory payload: %s", providerStub.lastUser)
	}
}

func TestLLMProviderServiceJudgeBypassesCacheWhenEpisodicContextChanges(t *testing.T) {
	providerStub := &stubRiskSessionProvider{callResp: `{"clear_structure":true,"integrity":true,"reason":"ok","signal_tag":"support_retest"}`}
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: providerStub},
		Prompts: LLMPromptBuilder{
			ProviderStructureSystem: "provider-structure-system",
			UserFormat:              UserPromptFormatBullet,
		},
		Cache: NewLLMStageCache(),
	}
	input := agent.StructureSummary{
		Regime:         agent.RegimeTrendUp,
		LastBreak:      agent.LastBreakBosUp,
		Quality:        agent.QualityClean,
		Pattern:        agent.PatternFlag,
		VolumeAction:   "vol ok",
		CandleReaction: "retest ok",
	}

	if _, _, _, _, err := svc.Judge(context.Background(), "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() error = %v", err)
	}
	ctx := memory.WithEpisodicContext(context.Background(), "[1] 上次结构假突破后快速回落")
	if _, _, _, _, err := svc.Judge(ctx, "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() with episodic memory error = %v", err)
	}

	if providerStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when episodic context changes", providerStub.callCount)
	}
	if !strings.Contains(providerStub.lastUser, "历史交易经验（仅供参考，禁止直接复用结论）") {
		t.Fatalf("user prompt missing episodic block: %s", providerStub.lastUser)
	}
}

func TestLLMProviderServiceJudgeAppendsSemanticRulesToSystem(t *testing.T) {
	providerStub := &stubRiskSessionProvider{callResp: `{"clear_structure":true,"integrity":true,"reason":"ok","signal_tag":"support_retest"}`}
	svc := LLMProviderService{
		Runner: &provider.Runner{Structure: providerStub},
		Prompts: LLMPromptBuilder{
			ProviderStructureSystem: "provider-structure-system",
			UserFormat:              UserPromptFormatBullet,
		},
		Cache: NewLLMStageCache(),
	}
	input := agent.StructureSummary{
		Regime:         agent.RegimeTrendUp,
		LastBreak:      agent.LastBreakBosUp,
		Quality:        agent.QualityClean,
		Pattern:        agent.PatternFlag,
		VolumeAction:   "vol ok",
		CandleReaction: "retest ok",
	}

	if _, _, _, _, err := svc.Judge(context.Background(), "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() error = %v", err)
	}
	ctx := memory.WithSemanticContext(context.Background(), "交易规则与经验:\n[1] 结构突破前若成交量衰减则降权")
	if _, _, _, _, err := svc.Judge(ctx, "ETHUSDT", agent.IndicatorSummary{}, input, agent.MechanicsSummary{}, decision.AgentEnabled{Structure: true}, decision.ProviderDataContext{}); err != nil {
		t.Fatalf("Judge() with semantic rules error = %v", err)
	}

	if providerStub.callCount != 2 {
		t.Fatalf("call_count=%d want 2 when semantic rules change", providerStub.callCount)
	}
	if !strings.Contains(providerStub.lastSystem, "结构突破前若成交量衰减则降权") {
		t.Fatalf("system prompt missing semantic rules: %s", providerStub.lastSystem)
	}
}
