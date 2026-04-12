package decision

import (
	"context"
	"strings"
	"testing"
	"time"

	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/features"
	"brale-core/internal/decision/fund"
	"brale-core/internal/decision/provider"
	"brale-core/internal/snapshot"
	"brale-core/internal/store"
)

func TestStoreHooksPersistStructuredInputsAndRenderTrace(t *testing.T) {
	db, err := store.OpenSQLite(t.TempDir() + "/trace.db")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	if err := store.Migrate(db, store.MigrateOptions{Full: true}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	s := store.NewStore(db)
	hooks := StoreHooks{Store: s, SourceVersion: "test"}

	ctx := context.Background()
	snap := snapshot.MarketSnapshot{Timestamp: time.Unix(1710000000, 0).UTC()}
	snapID := uint(1710000000)
	symbol := "BTCUSDT"
	inputs := AgentInputSet{
		Indicator: features.IndicatorJSON{Symbol: symbol, Interval: "multi", RawJSON: []byte(`{"multi_tf":[{"interval":"15m"}]}`)},
		Structure: features.TrendJSON{Symbol: symbol, Interval: "multi", RawJSON: []byte(`{"blocks":[{"interval":"1h"}]}`)},
		Mechanics: features.MechanicsSnapshot{Symbol: symbol, RawJSON: []byte(`{"mechanics_conflict":["oi_vs_price"]}`)},
	}
	if err := hooks.SaveAgent(
		ctx,
		snap,
		snapID,
		symbol,
		agent.IndicatorSummary{Expansion: agent.ExpansionExpanding},
		agent.StructureSummary{Regime: agent.RegimeRange},
		agent.MechanicsSummary{RiskLevel: agent.RiskLevelLow},
		inputs,
		AgentEnabled{Indicator: true, Structure: true, Mechanics: true},
		AgentPromptSet{Indicator: LLMStagePrompt{System: "agent-sys", User: "agent-user"}},
	); err != nil {
		t.Fatalf("SaveAgent: %v", err)
	}

	dataCtx := ProviderDataContext{
		IndicatorCrossTF: &IndicatorCrossTFContext{DecisionTFBias: "up", Alignment: "aligned"},
		StructureAnchorCtx: &StructureAnchorContext{
			LatestBreakType:    "bos_up",
			LatestBreakBarAge:  1,
			SupertrendState:    "UP",
			SupertrendInterval: "1h",
		},
		MechanicsCtx: &MechanicsDataContext{Conflicts: []string{"oi_vs_price"}, ReversalRisk: "low"},
	}
	if err := hooks.SaveProvider(
		ctx,
		snap,
		snapID,
		symbol,
		fund.ProviderBundle{
			Indicator: provider.IndicatorProviderOut{MomentumExpansion: true, Alignment: true, SignalTag: "trend_surge"},
			Structure: provider.StructureProviderOut{ClearStructure: true, Integrity: true, SignalTag: "breakout_confirmed"},
			Mechanics: provider.MechanicsProviderOut{SignalTag: "neutral"},
			Enabled:   fund.ProviderEnabled{Indicator: true, Structure: true, Mechanics: true},
		},
		dataCtx,
		ProviderPromptSet{Indicator: LLMStagePrompt{System: "provider-sys", User: "provider-user"}},
	); err != nil {
		t.Fatalf("SaveProvider: %v", err)
	}

	agentEvents, err := s.ListAgentEventsBySnapshot(ctx, symbol, snapID)
	if err != nil {
		t.Fatalf("ListAgentEventsBySnapshot: %v", err)
	}
	if len(agentEvents) != 3 {
		t.Fatalf("agent event count=%d want 3", len(agentEvents))
	}
	if string(agentEvents[0].InputJSON) == "" {
		t.Fatalf("agent input json should be persisted")
	}

	providerEvents, err := s.ListProviderEventsBySnapshot(ctx, symbol, snapID)
	if err != nil {
		t.Fatalf("ListProviderEventsBySnapshot: %v", err)
	}
	if len(providerEvents) != 3 {
		t.Fatalf("provider event count=%d want 3", len(providerEvents))
	}
	if string(providerEvents[0].DataContextJSON) == "" {
		t.Fatalf("provider data context json should be persisted")
	}

	gateRec := &store.GateEventRecord{
		ID:              1,
		SnapshotID:      snapID,
		Symbol:          symbol,
		Timestamp:       snap.Timestamp.Unix(),
		GlobalTradeable: true,
		GateReason:      "PASS_STRONG",
		Direction:       "long",
		DecisionAction:  "ALLOW",
		Grade:           1,
	}
	markdown := renderRoundTraceMarkdown(hooks, gateRec, agentEvents, providerEvents)
	if !strings.Contains(markdown, "## Agent Input Payloads") {
		t.Fatalf("trace missing agent input section:\n%s", markdown)
	}
	if !strings.Contains(markdown, `"interval": "15m"`) {
		t.Fatalf("trace missing agent input payload:\n%s", markdown)
	}
	if !strings.Contains(markdown, "## Provider Data Context") {
		t.Fatalf("trace missing provider data context section:\n%s", markdown)
	}
	if !strings.Contains(markdown, `"decision_tf_bias": "up"`) {
		t.Fatalf("trace missing provider data context payload:\n%s", markdown)
	}
}
