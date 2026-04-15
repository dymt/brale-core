package runtime

import (
	"brale-core/internal/config"
	"brale-core/internal/memory"
	"brale-core/internal/store"
)

func buildWorkingMemory(symbolCfg config.SymbolConfig) memory.Store {
	if !symbolCfg.Memory.Enabled {
		return nil
	}
	return memory.NewWorkingMemory(symbolCfg.Memory.WorkingMemorySize)
}

func buildEpisodicMemory(symbolCfg config.SymbolConfig, s store.Store) memory.EpisodicStore {
	if !symbolCfg.Memory.EpisodicEnabled {
		return nil
	}
	ttl := symbolCfg.Memory.EpisodicTTLDays
	if ttl <= 0 {
		ttl = config.DefaultEpisodicTTLDays
	}
	maxPer := symbolCfg.Memory.EpisodicMaxPerSymbol
	if maxPer <= 0 {
		maxPer = config.DefaultEpisodicMaxPerSymbol
	}
	return memory.NewEpisodicMemory(s, maxPer, ttl)
}

func buildSemanticMemory(symbolCfg config.SymbolConfig, s store.Store) memory.SemanticStore {
	if !symbolCfg.Memory.SemanticEnabled {
		return nil
	}
	maxRules := symbolCfg.Memory.SemanticMaxRules
	if maxRules <= 0 {
		maxRules = config.DefaultSemanticMaxRules
	}
	return memory.NewSemanticMemory(s, maxRules)
}
