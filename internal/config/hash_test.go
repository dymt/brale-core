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

func TestHashSymbolConfigChangesWhenMemoryConfigChanges(t *testing.T) {
	base := SymbolConfig{
		Symbol:    "BTCUSDT",
		Intervals: []string{"15m"},
		Memory: MemoryConfig{
			Enabled:           false,
			WorkingMemorySize: 5,
		},
	}

	baseHash, err := HashSymbolConfig(base)
	if err != nil {
		t.Fatalf("HashSymbolConfig(base): %v", err)
	}

	cases := map[string]SymbolConfig{
		"enabled changes": {
			Symbol:    "BTCUSDT",
			Intervals: []string{"15m"},
			Memory: MemoryConfig{
				Enabled:           true,
				WorkingMemorySize: 5,
			},
		},
		"size changes": {
			Symbol:    "BTCUSDT",
			Intervals: []string{"15m"},
			Memory: MemoryConfig{
				Enabled:           false,
				WorkingMemorySize: 8,
			},
		},
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			hash, err := HashSymbolConfig(cfg)
			if err != nil {
				t.Fatalf("HashSymbolConfig(%s): %v", name, err)
			}
			if hash == baseHash {
				t.Fatalf("hashes should differ when memory config changes")
			}
		})
	}
}
