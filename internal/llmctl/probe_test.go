package llmctl

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brale-core/internal/config"
	"brale-core/internal/llm"
)

func TestBuildProbeTargetsReturnsThreeStagesInOrder(t *testing.T) {
	sys := config.SystemConfig{
		LLMModels: map[string]config.LLMModelConfig{
			"model-indicator": {Endpoint: "https://indicator.example/v1", APIKey: "k1"},
			"model-structure": {Endpoint: "https://structure.example/v1", APIKey: "k2"},
			"model-mechanics": {Endpoint: "https://mechanics.example/v1", APIKey: "k3"},
		},
	}
	sym := config.SymbolConfig{
		LLM: config.SymbolLLMConfig{
			Agent: config.LLMRoleSet{
				Indicator: config.LLMRoleConfig{Model: "model-indicator"},
				Structure: config.LLMRoleConfig{Model: "model-structure"},
				Mechanics: config.LLMRoleConfig{Model: "model-mechanics"},
			},
		},
	}

	targets, err := BuildProbeTargets(sys, sym, "")
	if err != nil {
		t.Fatalf("BuildProbeTargets() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("targets=%d want 3", len(targets))
	}
	if targets[0].Stage != "indicator" || targets[1].Stage != "structure" || targets[2].Stage != "mechanics" {
		t.Fatalf("unexpected stage order: %#v", targets)
	}
}

func TestBuildProbeTargetsFiltersStage(t *testing.T) {
	sys := config.SystemConfig{
		LLMModels: map[string]config.LLMModelConfig{
			"model-structure": {Endpoint: "https://structure.example/v1", APIKey: "k2"},
		},
	}
	sym := config.SymbolConfig{
		LLM: config.SymbolLLMConfig{
			Agent: config.LLMRoleSet{
				Structure: config.LLMRoleConfig{Model: "model-structure"},
			},
		},
	}

	targets, err := BuildProbeTargets(sys, sym, "structure")
	if err != nil {
		t.Fatalf("BuildProbeTargets() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets=%d want 1", len(targets))
	}
	if targets[0].Stage != "structure" {
		t.Fatalf("stage=%q want structure", targets[0].Stage)
	}
}

func TestLoadProbeTargetsIgnoresUnrelatedSystemEnvPlaceholders(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("PROBE_LLM_MODEL_IND", "model-indicator")
	t.Setenv("PROBE_LLM_MODEL_STR", "model-structure")
	t.Setenv("PROBE_LLM_MODEL_MEC", "model-mechanics")
	t.Setenv("PROBE_LLM_ENDPOINT_IND", "https://indicator.example/v1")
	t.Setenv("PROBE_LLM_ENDPOINT_STR", "https://structure.example/v1")
	t.Setenv("PROBE_LLM_ENDPOINT_MEC", "https://mechanics.example/v1")
	t.Setenv("PROBE_LLM_API_KEY_IND", "k1")
	t.Setenv("PROBE_LLM_API_KEY_STR", "k2")
	t.Setenv("PROBE_LLM_API_KEY_MEC", "k3")

	writeProbeFile(t, filepath.Join(repo, "configs", "system.toml"), strings.Join([]string{
		`execution_system = "freqtrade"`,
		`exec_endpoint = "${PROBE_UNUSED_EXEC_ENDPOINT}"`,
		``,
		`[database]`,
		`dsn = "${PROBE_UNUSED_DATABASE_DSN}"`,
		``,
		`[llm_models.'${PROBE_LLM_MODEL_IND}']`,
		`endpoint = "${PROBE_LLM_ENDPOINT_IND}"`,
		`api_key = "${PROBE_LLM_API_KEY_IND}"`,
		``,
		`[llm_models.'${PROBE_LLM_MODEL_STR}']`,
		`endpoint = "${PROBE_LLM_ENDPOINT_STR}"`,
		`api_key = "${PROBE_LLM_API_KEY_STR}"`,
		``,
		`[llm_models.'${PROBE_LLM_MODEL_MEC}']`,
		`endpoint = "${PROBE_LLM_ENDPOINT_MEC}"`,
		`api_key = "${PROBE_LLM_API_KEY_MEC}"`,
	}, "\n"))
	writeProbeFile(t, filepath.Join(repo, "configs", "symbols", "default.toml"), strings.Join([]string{
		`symbol = "BTCUSDT"`,
		`intervals = ["1h"]`,
		`kline_limit = 300`,
		``,
		`[agent]`,
		`indicator = true`,
		`structure = true`,
		`mechanics = true`,
		``,
		`[indicators]`,
		`engine = "ta"`,
		`ema_fast = 21`,
		`ema_mid = 50`,
		`ema_slow = 200`,
		`rsi_period = 14`,
		`atr_period = 14`,
		`stc_fast = 23`,
		`stc_slow = 50`,
		`bb_period = 20`,
		`bb_multiplier = 2.0`,
		`chop_period = 14`,
		`stoch_rsi_period = 14`,
		`aroon_period = 25`,
		`last_n = 5`,
		``,
		`[consensus]`,
		`score_threshold = 0.35`,
		`confidence_threshold = 0.52`,
		``,
		`[llm.agent.indicator]`,
		`model = "${PROBE_LLM_MODEL_IND}"`,
		`temperature = 0.2`,
		``,
		`[llm.agent.structure]`,
		`model = "${PROBE_LLM_MODEL_STR}"`,
		`temperature = 0.2`,
		``,
		`[llm.agent.mechanics]`,
		`model = "${PROBE_LLM_MODEL_MEC}"`,
		`temperature = 0.2`,
		``,
		`[llm.provider.indicator]`,
		`model = "${PROBE_LLM_MODEL_IND}"`,
		`temperature = 0.2`,
		``,
		`[llm.provider.structure]`,
		`model = "${PROBE_LLM_MODEL_STR}"`,
		`temperature = 0.2`,
		``,
		`[llm.provider.mechanics]`,
		`model = "${PROBE_LLM_MODEL_MEC}"`,
		`temperature = 0.2`,
	}, "\n"))

	targets, err := LoadProbeTargets(repo, "")
	if err != nil {
		t.Fatalf("LoadProbeTargets() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("targets=%d want 3", len(targets))
	}
	if targets[0].Endpoint != "https://indicator.example/v1" {
		t.Fatalf("indicator endpoint=%q want https://indicator.example/v1", targets[0].Endpoint)
	}
	if targets[2].APIKey != "k3" {
		t.Fatalf("mechanics api_key=%q want k3", targets[2].APIKey)
	}
}

func TestProbeStructuredSupportWithClientSuccess(t *testing.T) {
	client := &llm.OpenAIClient{
		Endpoint:         "https://llm.example/v1",
		Model:            "m",
		APIKey:           "k",
		Timeout:          time.Second,
		StructuredOutput: true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		})},
	}

	if err := ProbeStructuredSupportWithClient(context.Background(), client); err != nil {
		t.Fatalf("ProbeStructuredSupportWithClient() error = %v", err)
	}
}

func TestProbeStructuredSupportWithClientRejectsInvalidPayload(t *testing.T) {
	client := &llm.OpenAIClient{
		Endpoint:         "https://llm.example/v1",
		Model:            "m",
		APIKey:           "k",
		Timeout:          time.Second,
		StructuredOutput: true,
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(`{"choices":[{"message":{"content":"not-json"}}]}`), nil
		})},
	}

	if err := ProbeStructuredSupportWithClient(context.Background(), client); err == nil {
		t.Fatalf("expected error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func writeProbeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
