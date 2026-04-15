package backtest

import (
	"time"

	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/fund"
	"brale-core/internal/decision/ruleflow"
	"brale-core/internal/store"
)

type TimeRange struct {
	StartUnix int64
	EndUnix   int64
}

type ReplayResult struct {
	Symbol  string        `json:"symbol"`
	Rounds  []ReplayRound `json:"rounds"`
	Metrics ReplayMetrics `json:"metrics"`
}

type ReplayRound struct {
	SnapshotID      uint              `json:"snapshot_id"`
	Timestamp       time.Time         `json:"timestamp"`
	State           fsm.PositionState `json:"state"`
	OriginalGate    fund.GateDecision `json:"original_gate"`
	ReplayedGate    fund.GateDecision `json:"replayed_gate"`
	PriceAtDecision float64           `json:"price_at_decision"`
	PriceAfter      float64           `json:"price_after"`
	Changed         bool              `json:"changed"`
	Skipped         bool              `json:"skipped"`
	SkipReason      string            `json:"skip_reason,omitempty"`
}

type Round struct {
	SnapshotID       uint
	Timestamp        int64
	State            fsm.PositionState
	Gate             store.GateEventRecord
	Derived          map[string]any
	PriceAtDecision  float64
	Agents           RoundAgents
	Providers        RoundProviders
	HasInPositionCtx bool
}

type RoundAgents struct {
	Indicator    agent.IndicatorSummary
	IndicatorSet bool
	Structure    agent.StructureSummary
	StructureSet bool
	Mechanics    agent.MechanicsSummary
	MechanicsSet bool
}

type RoundProviders struct {
	Bundle     fund.ProviderBundle
	InPosition ruleflow.InPositionOutputs
}
