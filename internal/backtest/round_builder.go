package backtest

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/provider"
	"brale-core/internal/pkg/parseutil"
	"brale-core/internal/store"
)

type roundAccumulator struct {
	round              Round
	hasGate            bool
	agentTimestamp     map[string]int64
	providerTimestamp  map[string]int64
	inPositionStageSet map[string]bool
}

func BuildRounds(agents []store.AgentEventRecord, providers []store.ProviderEventRecord, gates []store.GateEventRecord) ([]Round, error) {
	if len(gates) == 0 {
		return nil, nil
	}
	bySnapshot := make(map[uint]*roundAccumulator, len(gates))
	for _, rec := range gates {
		acc := ensureRoundAccumulator(bySnapshot, rec.SnapshotID)
		if !acc.hasGate || rec.Timestamp >= acc.round.Timestamp {
			acc.hasGate = true
			acc.round.SnapshotID = rec.SnapshotID
			acc.round.Timestamp = rec.Timestamp
			acc.round.Gate = rec
		}
	}
	for _, rec := range agents {
		acc := ensureRoundAccumulator(bySnapshot, rec.SnapshotID)
		if err := acc.applyAgent(rec); err != nil {
			return nil, err
		}
	}
	for _, rec := range providers {
		acc := ensureRoundAccumulator(bySnapshot, rec.SnapshotID)
		if err := acc.applyProvider(rec); err != nil {
			return nil, err
		}
	}

	rounds := make([]Round, 0, len(bySnapshot))
	for _, acc := range bySnapshot {
		if !acc.hasGate {
			continue
		}
		if err := acc.finalize(); err != nil {
			return nil, err
		}
		rounds = append(rounds, acc.round)
	}
	slices.SortFunc(rounds, func(a, b Round) int {
		if a.Timestamp < b.Timestamp {
			return -1
		}
		if a.Timestamp > b.Timestamp {
			return 1
		}
		if a.SnapshotID < b.SnapshotID {
			return -1
		}
		if a.SnapshotID > b.SnapshotID {
			return 1
		}
		return 0
	})
	return rounds, nil
}

func ensureRoundAccumulator(bySnapshot map[uint]*roundAccumulator, snapshotID uint) *roundAccumulator {
	acc, ok := bySnapshot[snapshotID]
	if ok {
		return acc
	}
	acc = &roundAccumulator{
		agentTimestamp:     map[string]int64{},
		providerTimestamp:  map[string]int64{},
		inPositionStageSet: map[string]bool{},
	}
	bySnapshot[snapshotID] = acc
	return acc
}

func (a *roundAccumulator) applyAgent(rec store.AgentEventRecord) error {
	stage := strings.ToLower(strings.TrimSpace(rec.Stage))
	lastTs, ok := a.agentTimestamp[stage]
	if ok && rec.Timestamp < lastTs {
		return nil
	}
	switch stage {
	case "indicator":
		var out agent.IndicatorSummary
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal agent indicator snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Agents.Indicator = out
		a.round.Agents.IndicatorSet = true
	case "structure":
		var out agent.StructureSummary
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal agent structure snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Agents.Structure = out
		a.round.Agents.StructureSet = true
	case "mechanics":
		var out agent.MechanicsSummary
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal agent mechanics snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Agents.Mechanics = out
		a.round.Agents.MechanicsSet = true
	default:
		return fmt.Errorf("unknown agent stage %q for snapshot %d", rec.Stage, rec.SnapshotID)
	}
	a.agentTimestamp[stage] = rec.Timestamp
	return nil
}

func (a *roundAccumulator) applyProvider(rec store.ProviderEventRecord) error {
	role := strings.ToLower(strings.TrimSpace(rec.Role))
	lastTs, ok := a.providerTimestamp[role]
	if ok && rec.Timestamp < lastTs {
		return nil
	}
	switch role {
	case "indicator":
		var out provider.IndicatorProviderOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal provider indicator snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.Bundle.Indicator = out
		a.round.Providers.Bundle.Enabled.Indicator = true
	case "structure":
		var out provider.StructureProviderOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal provider structure snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.Bundle.Structure = out
		a.round.Providers.Bundle.Enabled.Structure = true
	case "mechanics":
		var out provider.MechanicsProviderOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal provider mechanics snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.Bundle.Mechanics = out
		a.round.Providers.Bundle.Enabled.Mechanics = true
	case "indicator_in_position":
		var out provider.InPositionIndicatorOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal in-position indicator snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.InPosition.Indicator = out
		a.round.HasInPositionCtx = true
		a.inPositionStageSet["indicator"] = true
	case "structure_in_position":
		var out provider.InPositionStructureOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal in-position structure snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.InPosition.Structure = out
		a.round.HasInPositionCtx = true
		a.inPositionStageSet["structure"] = true
	case "mechanics_in_position":
		var out provider.InPositionMechanicsOut
		if err := json.Unmarshal(rec.OutputJSON, &out); err != nil {
			return fmt.Errorf("unmarshal in-position mechanics snapshot %d: %w", rec.SnapshotID, err)
		}
		a.round.Providers.InPosition.Mechanics = out
		a.round.HasInPositionCtx = true
		a.inPositionStageSet["mechanics"] = true
	default:
		return fmt.Errorf("unknown provider role %q for snapshot %d", rec.Role, rec.SnapshotID)
	}
	a.providerTimestamp[role] = rec.Timestamp
	return nil
}

func (a *roundAccumulator) finalize() error {
	derived, err := parseDerivedJSON(a.round.Gate.DerivedJSON)
	if err != nil {
		return fmt.Errorf("parse derived json snapshot %d: %w", a.round.SnapshotID, err)
	}
	a.round.Derived = derived
	a.round.PriceAtDecision = currentPriceFromDerived(derived)
	if a.round.HasInPositionCtx {
		a.round.Providers.InPosition.Ready = len(a.inPositionStageSet) > 0
	}
	a.round.State = inferRoundState(a.round)
	return nil
}

func parseDerivedJSON(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func currentPriceFromDerived(derived map[string]any) float64 {
	if len(derived) == 0 {
		return 0
	}
	price, ok := parseutil.FloatOK(derived["current_price"])
	if !ok {
		return 0
	}
	return price
}

func inferRoundState(round Round) fsm.PositionState {
	if round.HasInPositionCtx {
		return fsm.StateInPosition
	}
	switch strings.ToUpper(strings.TrimSpace(round.Gate.DecisionAction)) {
	case "TIGHTEN", "EXIT", "KEEP", "HOLD", "MANAGE", "REDUCE":
		return fsm.StateInPosition
	default:
		return fsm.StateFlat
	}
}
