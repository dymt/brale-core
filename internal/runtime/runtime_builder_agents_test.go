package runtime

import (
	"testing"
	"time"

	"brale-core/internal/config"
)

func TestNewLLMClientFindsSystemModelConfigCaseInsensitively(t *testing.T) {
	timeoutSec := 45
	structuredOutput := true
	sys := config.SystemConfig{
		LLMModels: map[string]config.LLMModelConfig{
			"minimax-m2.5": {
				Endpoint:         "https://llm.example.com/v1",
				APIKey:           "secret-key",
				TimeoutSec:       &timeoutSec,
				Concurrency:      nil,
				StructuredOutput: &structuredOutput,
			},
		},
	}
	temp := 0.2
	role := config.LLMRoleConfig{Model: "MiniMax-M2.5", Temperature: &temp}

	client := newLLMClient(sys, role)

	if client.Endpoint != "https://llm.example.com/v1" {
		t.Fatalf("endpoint=%q, want https://llm.example.com/v1", client.Endpoint)
	}
	if client.APIKey != "secret-key" {
		t.Fatalf("api key=%q, want secret-key", client.APIKey)
	}
	if client.Model != "MiniMax-M2.5" {
		t.Fatalf("model=%q, want original model name preserved", client.Model)
	}
	if client.Timeout != 45*time.Second {
		t.Fatalf("timeout=%v, want %v", client.Timeout, 45*time.Second)
	}
	if !client.StructuredOutput {
		t.Fatalf("StructuredOutput=%v want true", client.StructuredOutput)
	}
}
