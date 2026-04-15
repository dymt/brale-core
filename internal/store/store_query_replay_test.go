package store

import (
	"context"
	"testing"
)

func TestReplayEventQueriesByTimeRange(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	agentRows := []AgentEventRecord{
		{SnapshotID: 11, Symbol: "BTCUSDT", Timestamp: 100, Stage: "indicator"},
		{SnapshotID: 11, Symbol: "BTCUSDT", Timestamp: 100, Stage: "structure"},
		{SnapshotID: 22, Symbol: "BTCUSDT", Timestamp: 200, Stage: "mechanics"},
		{SnapshotID: 33, Symbol: "BTCUSDT", Timestamp: 300, Stage: "indicator"},
	}
	for _, rec := range agentRows {
		rec := rec
		if err := store.SaveAgentEvent(ctx, &rec); err != nil {
			t.Fatalf("save agent event: %v", err)
		}
	}

	providerRows := []ProviderEventRecord{
		{SnapshotID: 11, Symbol: "BTCUSDT", Timestamp: 100, ProviderID: "provider-a", Role: "indicator"},
		{SnapshotID: 22, Symbol: "BTCUSDT", Timestamp: 200, ProviderID: "provider-a", Role: "structure"},
		{SnapshotID: 33, Symbol: "BTCUSDT", Timestamp: 300, ProviderID: "provider-a", Role: "mechanics"},
	}
	for _, rec := range providerRows {
		rec := rec
		if err := store.SaveProviderEvent(ctx, &rec); err != nil {
			t.Fatalf("save provider event: %v", err)
		}
	}

	gateRows := []GateEventRecord{
		{SnapshotID: 11, Symbol: "BTCUSDT", Timestamp: 100, DecisionAction: "WAIT"},
		{SnapshotID: 22, Symbol: "BTCUSDT", Timestamp: 200, DecisionAction: "ALLOW"},
		{SnapshotID: 33, Symbol: "BTCUSDT", Timestamp: 300, DecisionAction: "VETO"},
	}
	for _, rec := range gateRows {
		rec := rec
		if err := store.SaveGateEvent(ctx, &rec); err != nil {
			t.Fatalf("save gate event: %v", err)
		}
	}

	agents, err := store.ListAgentEventsByTimeRange(ctx, " btcusdt ", 100, 200)
	if err != nil {
		t.Fatalf("ListAgentEventsByTimeRange: %v", err)
	}
	if len(agents) != 3 {
		t.Fatalf("len(agents)=%d want=3", len(agents))
	}
	if agents[0].Timestamp != 100 || agents[len(agents)-1].Timestamp != 200 {
		t.Fatalf("unexpected timestamps: %+v", agents)
	}

	providers, err := store.ListProviderEventsByTimeRange(ctx, "BTCUSDT", 150, 350)
	if err != nil {
		t.Fatalf("ListProviderEventsByTimeRange: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("len(providers)=%d want=2", len(providers))
	}
	if providers[0].Role != "structure" || providers[1].Role != "mechanics" {
		t.Fatalf("unexpected provider roles: %+v", providers)
	}

	gates, err := store.ListGateEventsByTimeRange(ctx, "BTCUSDT", 1, 299)
	if err != nil {
		t.Fatalf("ListGateEventsByTimeRange: %v", err)
	}
	if len(gates) != 2 {
		t.Fatalf("len(gates)=%d want=2", len(gates))
	}
	if gates[0].DecisionAction != "WAIT" || gates[1].DecisionAction != "ALLOW" {
		t.Fatalf("unexpected gate decisions: %+v", gates)
	}

	snapshotIDs, err := store.ListDistinctSnapshotIDs(ctx, "BTCUSDT", 50, 250)
	if err != nil {
		t.Fatalf("ListDistinctSnapshotIDs: %v", err)
	}
	if len(snapshotIDs) != 2 || snapshotIDs[0] != 11 || snapshotIDs[1] != 22 {
		t.Fatalf("snapshotIDs=%v want=[11 22]", snapshotIDs)
	}
}

func TestReplayEventQueriesRequireSymbol(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	if _, err := store.ListAgentEventsByTimeRange(ctx, "", 1, 2); err == nil {
		t.Fatalf("expected agent query error")
	}
	if _, err := store.ListProviderEventsByTimeRange(ctx, "", 1, 2); err == nil {
		t.Fatalf("expected provider query error")
	}
	if _, err := store.ListGateEventsByTimeRange(ctx, "", 1, 2); err == nil {
		t.Fatalf("expected gate query error")
	}
	if _, err := store.ListDistinctSnapshotIDs(ctx, "", 1, 2); err == nil {
		t.Fatalf("expected distinct snapshot id query error")
	}
}
