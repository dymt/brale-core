package bootstrap

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"brale-core/internal/config"
	"brale-core/internal/memory"
	"brale-core/internal/transport/feishubot"

	"go.uber.org/zap"
)

func TestResolveDurationWithDefault(t *testing.T) {
	fallback := 5 * time.Minute
	if got := resolveDurationWithDefault("", fallback); got != fallback {
		t.Fatalf("empty should fallback, got=%v", got)
	}
	if got := resolveDurationWithDefault("bad", fallback); got != fallback {
		t.Fatalf("invalid should fallback, got=%v", got)
	}
	if got := resolveDurationWithDefault("0s", fallback); got != fallback {
		t.Fatalf("non-positive should fallback, got=%v", got)
	}
	if got := resolveDurationWithDefault("30s", fallback); got != 30*time.Second {
		t.Fatalf("valid should parse, got=%v", got)
	}
}

func TestBuildAllowSymbol(t *testing.T) {
	allow := buildAllowSymbol(config.SymbolIndexConfig{Symbols: []config.SymbolIndexEntry{{Symbol: "ETH/USDT:USDT"}, {Symbol: "BTC"}}})
	if !allow("ethusdt") {
		t.Fatalf("ethusdt should be allowed")
	}
	if !allow(" BTCUSDT ") {
		t.Fatalf("BTCUSDT should be allowed")
	}
	if !allow("btc") {
		t.Fatalf("btc should normalize to BTCUSDT and be allowed")
	}
	if allow("SOLUSDT") {
		t.Fatalf("SOLUSDT should not be allowed")
	}
}

func TestStartFeishuBot_FeishuOnly(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.SystemConfig{
		Notification: config.NotificationConfig{
			Enabled: true,
			Telegram: config.TelegramConfig{
				Enabled: false,
			},
			Feishu: config.FeishuConfig{
				BotEnabled:           true,
				BotMode:              "callback",
				AppID:                "cli_test",
				AppSecret:            "secret",
				VerificationToken:    "verify-token",
				EncryptKey:           "",
				DefaultReceiveIDType: "chat_id",
				DefaultReceiveID:     "oc_1",
			},
		},
	}

	startFeishuBot(context.Background(), zap.NewNop(), cfg, ":9991", mux)
	req := httptest.NewRequest(http.MethodPost, "/api/feishu/events", bytes.NewBufferString(`{"type":"url_verification","token":"verify-token","challenge":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected route attached and status 200, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestStartFeishuBot_InvalidConfig(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.SystemConfig{
		Notification: config.NotificationConfig{
			Enabled: true,
			Feishu: config.FeishuConfig{
				BotEnabled: true,
				BotMode:    "callback",
				AppID:      "cli_test",
				AppSecret:  "secret",
			},
		},
	}

	startFeishuBot(context.Background(), zap.NewNop(), cfg, ":9991", mux)
	req := httptest.NewRequest(http.MethodPost, "/api/feishu/events", bytes.NewBufferString(`{"type":"url_verification","token":"verify-token","challenge":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected route missing for invalid config, got %d", resp.Code)
	}
}

func TestStartFeishuBot_LongConnectionMode(t *testing.T) {
	mux := http.NewServeMux()
	runner := &fakeFeishuRunner{started: make(chan struct{}, 1)}
	original := newFeishuBot
	newFeishuBot = func(cfg feishubot.Config, logger *zap.Logger) (feishuBotRunner, error) {
		return runner, nil
	}
	defer func() {
		newFeishuBot = original
	}()

	cfg := config.SystemConfig{
		Notification: config.NotificationConfig{
			Enabled: true,
			Feishu: config.FeishuConfig{
				BotEnabled: true,
				BotMode:    "long_connection",
				AppID:      "cli_test",
				AppSecret:  "secret",
			},
		},
	}

	startFeishuBot(context.Background(), zap.NewNop(), cfg, ":9991", mux)
	select {
	case <-runner.started:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected long connection runner to start")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/feishu/events", bytes.NewBufferString(`{"type":"url_verification","token":"verify-token","challenge":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected callback route not attached in long mode, got %d", resp.Code)
	}
}

func TestBuildTopMuxRouteBoundaries(t *testing.T) {
	viewerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("viewer"))
	})
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("dashboard"))
	})
	runtimeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("runtime"))
	})

	mux := buildTopMux(viewerHandler, dashboardHandler, runtimeHandler)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "decision-view exact", path: "/decision-view", body: "viewer"},
		{name: "decision-view nested", path: "/decision-view/api/chains", body: "viewer"},
		{name: "dashboard exact", path: "/dashboard", body: "dashboard"},
		{name: "dashboard nested", path: "/dashboard/", body: "dashboard"},
		{name: "runtime fallback", path: "/api/runtime/schedule/status", body: "runtime"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Body.String() != tc.body {
				t.Fatalf("body=%q want=%q", rec.Body.String(), tc.body)
			}
		})
	}
}

type fakeFeishuRunner struct {
	started chan struct{}
}

func (r *fakeFeishuRunner) ensureStartedChan() {
	if r.started == nil {
		r.started = make(chan struct{}, 1)
	}
}

func (r *fakeFeishuRunner) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (r *fakeFeishuRunner) RunLongConnection(ctx context.Context) error {
	r.ensureStartedChan()
	select {
	case r.started <- struct{}{}:
	default:
	}
	return nil
}

func TestBuildPositionReflectorReturnsNilWhenNoEpisodicMemoryEnabled(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeReflectorTestConfig(t, dir, false, false)
	sys := reflectorTestSystemConfig()
	index := config.SymbolIndexConfig{
		Symbols: []config.SymbolIndexEntry{
			{Symbol: "BTCUSDT", Config: "symbols/BTCUSDT.toml", Strategy: "strategies/BTCUSDT.toml"},
		},
	}

	reflector, err := buildPositionReflector(sys, indexPath, index, nil)
	if err != nil {
		t.Fatalf("buildPositionReflector() error = %v", err)
	}
	if reflector != nil {
		t.Fatalf("reflector=%T want nil when episodic memory is disabled", reflector)
	}
}

func TestBuildPositionReflectorConfiguresEnabledSymbols(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeReflectorTestConfig(t, dir, true, true)
	sys := reflectorTestSystemConfig()
	index := config.SymbolIndexConfig{
		Symbols: []config.SymbolIndexEntry{
			{Symbol: "BTCUSDT", Config: "symbols/BTCUSDT.toml", Strategy: "strategies/BTCUSDT.toml"},
		},
	}

	reflector, err := buildPositionReflector(sys, indexPath, index, nil)
	if err != nil {
		t.Fatalf("buildPositionReflector() error = %v", err)
	}
	router, ok := reflector.(*symbolPositionReflector)
	if !ok {
		t.Fatalf("reflector type=%T want *symbolPositionReflector", reflector)
	}
	adapter := router.bySymbol["BTCUSDT"]
	if adapter == nil || adapter.Reflector == nil {
		t.Fatalf("expected BTCUSDT reflector adapter, got %#v", router.bySymbol)
	}
	if adapter.Reflector.Episodic == nil {
		t.Fatalf("episodic reflector memory should be wired")
	}
	if adapter.Reflector.Semantic == nil {
		t.Fatalf("semantic reflector memory should be wired")
	}
	if adapter.Reflector.LLM == nil {
		t.Fatalf("reflector llm should be wired")
	}
	if _, ok := any(adapter.Reflector.Episodic).(*memory.EpisodicMemory); !ok {
		t.Fatalf("episodic reflector type=%T", adapter.Reflector.Episodic)
	}
	if _, ok := any(adapter.Reflector.Semantic).(*memory.SemanticMemory); !ok {
		t.Fatalf("semantic reflector type=%T", adapter.Reflector.Semantic)
	}
}

func reflectorTestSystemConfig() config.SystemConfig {
	return config.SystemConfig{
		LLMModels: map[string]config.LLMModelConfig{
			"mock": {
				Endpoint: "http://localhost:11434/v1",
				APIKey:   "dummy",
			},
		},
	}
}

func writeReflectorTestConfig(t *testing.T, dir string, episodicEnabled bool, semanticEnabled bool) string {
	t.Helper()
	indexPath := filepath.Join(dir, "symbols-index.toml")
	symbolDir := filepath.Join(dir, "symbols")
	strategyDir := filepath.Join(dir, "strategies")
	if err := os.MkdirAll(symbolDir, 0o755); err != nil {
		t.Fatalf("mkdir symbols: %v", err)
	}
	if err := os.MkdirAll(strategyDir, 0o755); err != nil {
		t.Fatalf("mkdir strategies: %v", err)
	}
	writeTestFile(t, indexPath, `
[[symbols]]
symbol = "BTCUSDT"
config = "symbols/BTCUSDT.toml"
strategy = "strategies/BTCUSDT.toml"
`)
	writeTestFile(t, filepath.Join(symbolDir, "BTCUSDT.toml"), `
symbol = "BTCUSDT"
intervals = ["1h"]
kline_limit = 200
[agent]
indicator = true
structure = true
mechanics = true
[indicators]
ema_fast = 21
ema_mid = 50
ema_slow = 200
rsi_period = 14
atr_period = 14
stc_fast = 23
stc_slow = 50
bb_period = 20
bb_multiplier = 2.0
chop_period = 14
stoch_rsi_period = 14
aroon_period = 25
last_n = 5
[memory]
enabled = true
working_memory_size = 5
episodic_enabled = `+boolTOML(episodicEnabled)+`
episodic_ttl_days = 7
episodic_max_per_symbol = 4
semantic_enabled = `+boolTOML(semanticEnabled)+`
semantic_max_rules = 6
[llm.agent.indicator]
model = "mock"
temperature = 0.2
[llm.agent.structure]
model = "mock"
temperature = 0.2
[llm.agent.mechanics]
model = "mock"
temperature = 0.2
[llm.provider.indicator]
model = "mock"
temperature = 0.2
[llm.provider.structure]
model = "mock"
temperature = 0.2
[llm.provider.mechanics]
model = "mock"
temperature = 0.2
`)
	writeTestFile(t, filepath.Join(strategyDir, "BTCUSDT.toml"), `symbol = "BTCUSDT"`)
	return indexPath
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func boolTOML(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
