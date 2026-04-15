package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brale-core/internal/store"

	"gorm.io/datatypes"
)

func TestBacktestRulesCommandJSONOutput(t *testing.T) {
	systemPath, indexPath, dbPath := writeBacktestConfigTree(t)
	seedBacktestReplayDB(t, dbPath)

	out, errOut, err := executeRootCommand(
		t,
		"backtest", "rules",
		"--symbol", "BTCUSDT",
		"--system", systemPath,
		"--index", indexPath,
		"--db", dbPath,
		"--from", "2026-04-13",
		"--to", "2026-04-13",
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("execute command: %v\nstderr=%s", err, errOut)
	}
	if !strings.Contains(out, `"symbol": "BTCUSDT"`) || !strings.Contains(out, `"metrics"`) {
		t.Fatalf("stdout=%s", out)
	}
}

func TestBacktestRulesCommandTextOutput(t *testing.T) {
	systemPath, indexPath, dbPath := writeBacktestConfigTree(t)
	seedBacktestReplayDB(t, dbPath)

	out, errOut, err := executeRootCommand(
		t,
		"backtest", "rules",
		"--symbol", "BTCUSDT",
		"--system", systemPath,
		"--index", indexPath,
		"--db", dbPath,
		"--from", "2026-04-13",
		"--to", "2026-04-13",
	)
	if err != nil {
		t.Fatalf("execute command: %v\nstderr=%s", err, errOut)
	}
	for _, needle := range []string{"Gate Replay Report: BTCUSDT", "Rounds:", "Changed Decisions:"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in stdout=%s", needle, out)
		}
	}
}

func TestBacktestRulesCommandHTMLRequiresOutput(t *testing.T) {
	systemPath, indexPath, dbPath := writeBacktestConfigTree(t)
	seedBacktestReplayDB(t, dbPath)

	_, _, err := executeRootCommand(
		t,
		"backtest", "rules",
		"--symbol", "BTCUSDT",
		"--system", systemPath,
		"--index", indexPath,
		"--db", dbPath,
		"--from", "2026-04-13",
		"--to", "2026-04-13",
		"--format", "html",
	)
	if err == nil || !strings.Contains(err.Error(), "--output is required for html format") {
		t.Fatalf("err=%v", err)
	}
}

func TestBacktestRulesCommandWritesHTMLFile(t *testing.T) {
	systemPath, indexPath, dbPath := writeBacktestConfigTree(t)
	seedBacktestReplayDB(t, dbPath)
	reportPath := filepath.Join(t.TempDir(), "report.html")

	out, errOut, err := executeRootCommand(
		t,
		"backtest", "rules",
		"--symbol", "BTCUSDT",
		"--system", systemPath,
		"--index", indexPath,
		"--db", dbPath,
		"--from", "2026-04-13",
		"--to", "2026-04-13",
		"--format", "html",
		"--output", reportPath,
	)
	if err != nil {
		t.Fatalf("execute command: %v\nstdout=%s\nstderr=%s", err, out, errOut)
	}
	raw, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("read report: %v", readErr)
	}
	if !strings.Contains(string(raw), "<html") || !strings.Contains(string(raw), "Gate Replay Report: BTCUSDT") {
		t.Fatalf("report=%s", string(raw))
	}
}

func writeBacktestConfigTree(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "brale.db")
	systemPath := filepath.Join(dir, "system.toml")
	indexPath := filepath.Join(dir, "symbols-index.toml")
	symbolDir := filepath.Join(dir, "symbols")
	strategyDir := filepath.Join(dir, "strategies")
	if err := os.MkdirAll(symbolDir, 0o755); err != nil {
		t.Fatalf("mkdir symbols: %v", err)
	}
	if err := os.MkdirAll(strategyDir, 0o755); err != nil {
		t.Fatalf("mkdir strategies: %v", err)
	}
	writeTestFile(t, systemPath, `
db_path = "`+dbPath+`"
execution_system = "freqtrade"
exec_endpoint = "http://localhost:8080"

[llm_models.mock]
endpoint = "http://localhost:11434/v1"
api_key = "dummy"
`)
	writeTestFile(t, indexPath, `
[[symbols]]
symbol = "BTCUSDT"
config = "symbols/BTCUSDT.toml"
strategy = "strategies/BTCUSDT.toml"
`)
	writeTestFile(t, filepath.Join(symbolDir, "BTCUSDT.toml"), `
symbol = "BTCUSDT"
intervals = ["15m", "1h", "4h"]
kline_limit = 300

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

[consensus]
score_threshold = 0.35
confidence_threshold = 0.52

[cooldown]
enabled = false

[llm.agent.indicator]
model = "mock"
temperature = 0.2
[llm.agent.structure]
model = "mock"
temperature = 0.1
[llm.agent.mechanics]
model = "mock"
temperature = 0.2
[llm.provider.indicator]
model = "mock"
temperature = 0.2
[llm.provider.structure]
model = "mock"
temperature = 0.1
[llm.provider.mechanics]
model = "mock"
temperature = 0.2
`)
	writeTestFile(t, filepath.Join(strategyDir, "BTCUSDT.toml"), `
symbol = "BTCUSDT"
id = "default-BTCUSDT"
rule_chain = "configs/rules/default.json"

[risk_management]
risk_per_trade_pct = 0.01
max_invest_pct = 1.0
max_leverage = 3.0
grade_1_factor = 0.3
grade_2_factor = 0.6
grade_3_factor = 1.0
entry_offset_atr = 0.1
entry_mode = "atr_offset"
orderbook_depth = 5
breakeven_fee_pct = 0.0
slippage_buffer_pct = 0.0005

[risk_management.risk_strategy]
mode = "native"

[risk_management.initial_exit]
policy = "atr_structure_v1"
structure_interval = "auto"

[risk_management.initial_exit.params]
stop_atr_multiplier = 2.0
stop_min_distance_pct = 0.005
take_profit_rr = [1.5, 3.0]

[risk_management.tighten_atr]
structure_threatened = 0.5
tp1_atr = 0.0
tp2_atr = 0.0
min_tp_distance_pct = 0.0
min_tp_gap_pct = 0.0
min_update_interval_sec = 300

[risk_management.gate]
quality_threshold = 0.35
edge_threshold = 0.1

[risk_management.sieve]
min_size_factor = 0.1
default_gate_action = "ALLOW"
default_size_factor = 1.0
`)
	return systemPath, indexPath, dbPath
}

func seedBacktestReplayDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := store.Migrate(db, store.MigrateOptions{Full: true}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st := store.NewStore(db)
	ctx := t.Context()
	at1 := time.Date(2026, 4, 13, 1, 0, 0, 0, time.UTC).Unix()
	at2 := time.Date(2026, 4, 13, 2, 0, 0, 0, time.UTC).Unix()
	for _, rec := range []store.AgentEventRecord{
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Stage: "indicator", OutputJSON: datatypes.JSON([]byte(`{"expansion":"expanding","alignment":"aligned","noise":"low","movement_score":0.7,"movement_confidence":0.8}`))},
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Stage: "structure", OutputJSON: datatypes.JSON([]byte(`{"regime":"trend_up","last_break":"bos_up","quality":"clean","pattern":"none","movement_score":0.85,"movement_confidence":0.9}`))},
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Stage: "mechanics", OutputJSON: datatypes.JSON([]byte(`{"leverage_state":"stable","crowding":"balanced","risk_level":"low","movement_score":0.1,"movement_confidence":0.6}`))},
	} {
		rec := rec
		if err := st.SaveAgentEvent(ctx, &rec); err != nil {
			t.Fatalf("save agent event: %v", err)
		}
	}
	for _, rec := range []store.ProviderEventRecord{
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Role: "indicator", OutputJSON: datatypes.JSON([]byte(`{"momentum_expansion":true,"alignment":true,"mean_rev_noise":false,"signal_tag":"trend_surge"}`))},
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Role: "structure", OutputJSON: datatypes.JSON([]byte(`{"clear_structure":true,"integrity":true,"reason":"ok","signal_tag":"support_retest"}`))},
		{SnapshotID: 101, Symbol: "BTCUSDT", Timestamp: at1, Role: "mechanics", OutputJSON: datatypes.JSON([]byte(`{"liquidation_stress":{"value":false,"confidence":"low","reason":"stable"},"signal_tag":"neutral"}`))},
	} {
		rec := rec
		if err := st.SaveProviderEvent(ctx, &rec); err != nil {
			t.Fatalf("save provider event: %v", err)
		}
	}
	for _, rec := range []store.GateEventRecord{
		{
			SnapshotID:      101,
			Symbol:          "BTCUSDT",
			Timestamp:       at1,
			GlobalTradeable: true,
			DecisionAction:  "ALLOW",
			GateReason:      "ALLOW",
			Direction:       "long",
			Grade:           3,
			DerivedJSON:     datatypes.JSON([]byte(`{"current_price":100.0,"direction_consensus":{"score":0.82,"confidence":0.74,"agreement":0.90,"coverage":0.81,"resonance_bonus":0.05,"resonance_active":true,"score_threshold":0.35,"confidence_threshold":0.52,"sources":{"indicator":{"confidence":0.8,"raw_confidence":0.8},"structure":{"confidence":0.9,"raw_confidence":0.9},"mechanics":{"confidence":0.6,"raw_confidence":0.6}}}}`)),
		},
		{
			SnapshotID:      102,
			Symbol:          "BTCUSDT",
			Timestamp:       at2,
			GlobalTradeable: false,
			DecisionAction:  "WAIT",
			GateReason:      "QUALITY_TOO_LOW",
			Direction:       "long",
			DerivedJSON:     datatypes.JSON([]byte(`{"current_price":108.0,"direction_consensus":{"score":0.20,"confidence":0.40,"agreement":0.40,"coverage":0.50,"resonance_bonus":0.0,"resonance_active":false,"score_threshold":0.35,"confidence_threshold":0.52}}`)),
		},
	} {
		rec := rec
		if err := st.SaveGateEvent(ctx, &rec); err != nil {
			t.Fatalf("save gate event: %v", err)
		}
	}
}
