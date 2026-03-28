package config

import "strings"

func CanonicalLLMModelKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func LookupLLMModelConfig(sys SystemConfig, model string) (LLMModelConfig, bool) {
	canonical := CanonicalLLMModelKey(model)
	if canonical == "" {
		return LLMModelConfig{}, false
	}
	if cfg, ok := sys.LLMModels[model]; ok {
		return cfg, true
	}
	if cfg, ok := sys.LLMModels[canonical]; ok {
		return cfg, true
	}
	for key, cfg := range sys.LLMModels {
		if CanonicalLLMModelKey(key) == canonical {
			return cfg, true
		}
	}
	return LLMModelConfig{}, false
}
