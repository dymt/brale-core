package runtime

import (
	"time"

	"brale-core/internal/config"
	"brale-core/internal/decision"
	"brale-core/internal/interval"
	"brale-core/internal/llm"
	llmapp "brale-core/internal/llm/app"
)

func buildSymbolAgents(sys config.SystemConfig, symbolCfg config.SymbolConfig) (decision.AgentService, decision.ProviderService, *llmapp.LLMRunTracker) {
	cache := llmapp.NewLLMStageCache()
	tracker := llmapp.NewLLMRunTracker()
	defaults := config.DefaultPromptDefaults()
	builder := llmapp.LLMPromptBuilder{
		AgentIndicatorSystem:      defaults.AgentIndicator,
		AgentStructureSystem:      defaults.AgentStructure,
		AgentMechanicsSystem:      defaults.AgentMechanics,
		ProviderIndicatorSystem:   defaults.ProviderIndicator,
		ProviderStructureSystem:   defaults.ProviderStructure,
		ProviderMechanicsSystem:   defaults.ProviderMechanics,
		ProviderInPosIndicatorSys: defaults.ProviderInPositionIndicator,
		ProviderInPosStructureSys: defaults.ProviderInPositionStructure,
		ProviderInPosMechanicsSys: defaults.ProviderInPositionMechanics,
		UserFormat:                llmapp.UserPromptFormatBullet,
	}
	agentRunner := &decision.AgentRunner{
		Indicator: newLLMClient(sys, symbolCfg.LLM.Agent.Indicator),
		Structure: newLLMClient(sys, symbolCfg.LLM.Agent.Structure),
		Mechanics: newLLMClient(sys, symbolCfg.LLM.Agent.Mechanics),
	}
	providerRunner := &decision.ProviderRunner{
		Indicator: newLLMClient(sys, symbolCfg.LLM.Provider.Indicator),
		Structure: newLLMClient(sys, symbolCfg.LLM.Provider.Structure),
		Mechanics: newLLMClient(sys, symbolCfg.LLM.Provider.Mechanics),
	}
	decisionInterval := ""
	if len(symbolCfg.Intervals) > 0 {
		decisionInterval = selectDecisionInterval(symbolCfg.Intervals)
	}
	return llmapp.LLMAgentService{Runner: agentRunner, Prompts: builder, Cache: cache, Tracker: tracker, DecisionInterval: decisionInterval}, llmapp.LLMProviderService{Runner: providerRunner, Prompts: builder, Cache: cache, Tracker: tracker}, tracker
}

func selectDecisionInterval(intervals []string) string {
	shortest := ""
	var shortestDur time.Duration
	for _, candidate := range intervals {
		dur, err := interval.ParseInterval(candidate)
		if err != nil {
			continue
		}
		if shortest == "" || dur < shortestDur {
			shortest = candidate
			shortestDur = dur
		}
	}
	if shortest != "" {
		return shortest
	}
	if len(intervals) == 0 {
		return ""
	}
	return intervals[0]
}

func newLLMClient(sys config.SystemConfig, role config.LLMRoleConfig) *llm.OpenAIClient {
	temp := 0.0
	if role.Temperature != nil {
		temp = *role.Temperature
	}
	modelCfg, _ := config.LookupLLMModelConfig(sys, role.Model)
	timeoutSec := 30
	if modelCfg.TimeoutSec != nil {
		timeoutSec = *modelCfg.TimeoutSec
	}
	structuredOutput := false
	if modelCfg.StructuredOutput != nil {
		structuredOutput = *modelCfg.StructuredOutput
	}
	return &llm.OpenAIClient{
		Endpoint:         modelCfg.Endpoint,
		Model:            role.Model,
		APIKey:           modelCfg.APIKey,
		Timeout:          time.Duration(timeoutSec) * time.Second,
		Temperature:      temp,
		StructuredOutput: structuredOutput,
	}
}
