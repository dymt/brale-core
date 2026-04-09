package config

import "testing"

func TestHashSystemConfigChangesWhenStructuredOutputChanges(t *testing.T) {
	base := SystemConfig{
		LLMModels: map[string]LLMModelConfig{
			"gpt-4o": {
				Endpoint: "https://api.openai.com/v1",
				APIKey:   "secret",
			},
		},
	}
	enabled := true
	disabled := false

	withStructured := base
	withStructured.LLMModels = map[string]LLMModelConfig{
		"gpt-4o": {
			Endpoint:         "https://api.openai.com/v1",
			APIKey:           "secret",
			StructuredOutput: &enabled,
		},
	}

	withoutStructured := base
	withoutStructured.LLMModels = map[string]LLMModelConfig{
		"gpt-4o": {
			Endpoint:         "https://api.openai.com/v1",
			APIKey:           "secret",
			StructuredOutput: &disabled,
		},
	}

	hashA, err := HashSystemConfig(withStructured)
	if err != nil {
		t.Fatalf("HashSystemConfig(withStructured): %v", err)
	}
	hashB, err := HashSystemConfig(withoutStructured)
	if err != nil {
		t.Fatalf("HashSystemConfig(withoutStructured): %v", err)
	}
	if hashA == hashB {
		t.Fatalf("hashes should differ when structured_output changes")
	}
}
