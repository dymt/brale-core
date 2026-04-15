package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"brale-core/internal/decision/decisionutil"
	"brale-core/internal/decision/direction"
	"brale-core/internal/decision/fsm"
	"brale-core/internal/decision/fund"
	"brale-core/internal/decision/ruleflow"
	"brale-core/internal/pkg/parseutil"
	"brale-core/internal/store"
	"brale-core/internal/strategy"
)

type RuleReplay struct {
	Store               store.TimelineQueryStore
	Binding             strategy.StrategyBinding
	ScoreThreshold      float64
	ConfidenceThreshold float64
}

func (r RuleReplay) Run(ctx context.Context, symbol string, tr TimeRange) (*ReplayResult, error) {
	if r.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if err := tr.Validate(); err != nil {
		return nil, err
	}
	agents, err := r.Store.ListAgentEventsByTimeRange(ctx, symbol, tr.StartUnix, tr.EndUnix)
	if err != nil {
		return nil, fmt.Errorf("load agent events: %w", err)
	}
	providers, err := r.Store.ListProviderEventsByTimeRange(ctx, symbol, tr.StartUnix, tr.EndUnix)
	if err != nil {
		return nil, fmt.Errorf("load provider events: %w", err)
	}
	gates, err := r.Store.ListGateEventsByTimeRange(ctx, symbol, tr.StartUnix, tr.EndUnix)
	if err != nil {
		return nil, fmt.Errorf("load gate events: %w", err)
	}
	if len(gates) == 0 {
		return nil, fmt.Errorf("no gate events found for %s in range", symbol)
	}
	rounds, err := BuildRounds(agents, providers, gates)
	if err != nil {
		return nil, fmt.Errorf("build rounds: %w", err)
	}
	result := &ReplayResult{Symbol: symbol, Rounds: make([]ReplayRound, 0, len(rounds))}
	for i, round := range rounds {
		item := ReplayRound{
			SnapshotID:      round.SnapshotID,
			Timestamp:       time.Unix(round.Timestamp, 0).UTC(),
			State:           round.State,
			OriginalGate:    originalGate(round.Gate, round.Derived),
			PriceAtDecision: round.PriceAtDecision,
			PriceAfter:      nextRoundPrice(rounds, i),
		}
		if round.State != fsm.StateFlat {
			item.Skipped = true
			item.SkipReason = "in_position_round"
			result.Rounds = append(result.Rounds, item)
			continue
		}
		input, buildErr := buildReplayGateInput(round, r.Binding, r.ScoreThreshold, r.ConfidenceThreshold)
		if buildErr != nil {
			item.Skipped = true
			item.SkipReason = buildErr.Error()
			result.Rounds = append(result.Rounds, item)
			continue
		}
		item.ReplayedGate = ruleflow.ReplayGateDecision(input)
		item.Changed = gateChanged(item.OriginalGate, item.ReplayedGate)
		result.Rounds = append(result.Rounds, item)
	}
	result.Metrics = computeMetrics(result.Rounds)
	return result, nil
}

func (tr TimeRange) Validate() error {
	if tr.StartUnix <= 0 {
		return fmt.Errorf("start time is required")
	}
	if tr.EndUnix <= 0 {
		return fmt.Errorf("end time is required")
	}
	if tr.EndUnix < tr.StartUnix {
		return fmt.Errorf("end time must be greater than or equal to start time")
	}
	return nil
}

type replayConsensus struct {
	Score               float64
	Confidence          float64
	Agreement           float64
	Coverage            float64
	ResonanceBonus      float64
	ResonanceActive     bool
	ScoreThreshold      float64
	ConfidenceThreshold float64
	AgentIndicatorConf  float64
	AgentStructureConf  float64
	AgentMechanicsConf  float64
}

func buildReplayGateInput(round Round, binding strategy.StrategyBinding, scoreThreshold, confidenceThreshold float64) (ruleflow.GateReplayInput, error) {
	consensus, err := resolveReplayConsensus(round, scoreThreshold, confidenceThreshold)
	if err != nil {
		return ruleflow.GateReplayInput{}, err
	}
	return ruleflow.GateReplayInput{
		State:                    round.State,
		Providers:                round.Providers.Bundle,
		AgentIndicatorConfidence: consensus.AgentIndicatorConf,
		AgentStructureConfidence: consensus.AgentStructureConf,
		AgentMechanicsConfidence: consensus.AgentMechanicsConf,
		ConsensusScore:           consensus.Score,
		ConsensusConfidence:      consensus.Confidence,
		ConsensusAgreement:       consensus.Agreement,
		ConsensusCoverage:        consensus.Coverage,
		ConsensusResonance:       consensus.ResonanceBonus,
		ConsensusResonant:        consensus.ResonanceActive,
		ScoreThreshold:           consensus.ScoreThreshold,
		ConfidenceThreshold:      consensus.ConfidenceThreshold,
		Binding:                  binding,
		CurrentPrice:             round.PriceAtDecision,
	}, nil
}

func resolveReplayConsensus(round Round, scoreThreshold, confidenceThreshold float64) (replayConsensus, error) {
	consensusRaw := mapFromNested(round.Derived, "direction_consensus")
	out := replayConsensus{}
	if value, ok := parseutil.FloatOK(consensusRaw["score"]); ok {
		out.Score = value
	}
	if value, ok := parseutil.FloatOK(consensusRaw["confidence"]); ok {
		out.Confidence = value
	}
	if value, ok := parseutil.FloatOK(consensusRaw["agreement"]); ok {
		out.Agreement = value
	}
	if value, ok := parseutil.FloatOK(consensusRaw["coverage"]); ok {
		out.Coverage = value
	}
	if value, ok := parseutil.FloatOK(consensusRaw["resonance_bonus"]); ok {
		out.ResonanceBonus = value
	}
	if value, ok := parseBool(consensusRaw["resonance_active"]); ok {
		out.ResonanceActive = value
	}
	out.ScoreThreshold = scoreThreshold
	if out.ScoreThreshold <= 0 {
		if value, ok := parseutil.FloatOK(consensusRaw["score_threshold"]); ok {
			out.ScoreThreshold = value
		}
	}
	if out.ScoreThreshold <= 0 {
		out.ScoreThreshold = direction.ThresholdScore()
	}
	out.ConfidenceThreshold = confidenceThreshold
	if out.ConfidenceThreshold <= 0 {
		if value, ok := parseutil.FloatOK(consensusRaw["confidence_threshold"]); ok {
			out.ConfidenceThreshold = value
		}
	}
	if out.ConfidenceThreshold <= 0 {
		out.ConfidenceThreshold = direction.ThresholdConfidence()
	}
	out.AgentIndicatorConf = resolveAgentConfidence(round, "indicator")
	out.AgentStructureConf = resolveAgentConfidence(round, "structure")
	out.AgentMechanicsConf = resolveAgentConfidence(round, "mechanics")

	if out.Confidence > 0 || out.Score != 0 || len(consensusRaw) > 0 {
		return out, nil
	}

	if !round.Agents.IndicatorSet && !round.Agents.StructureSet && !round.Agents.MechanicsSet {
		return replayConsensus{}, fmt.Errorf("direction_consensus missing for snapshot %d", round.SnapshotID)
	}
	computed := direction.ComputeConsensusWithThresholds(
		direction.Evidence{Source: direction.SourceIndicator, Score: round.Agents.Indicator.MovementScore, Confidence: out.AgentIndicatorConf},
		direction.Evidence{Source: direction.SourceStructure, Score: round.Agents.Structure.MovementScore, Confidence: out.AgentStructureConf},
		direction.Evidence{Source: direction.SourceMechanics, Score: round.Agents.Mechanics.MovementScore, Confidence: out.AgentMechanicsConf},
		out.ScoreThreshold,
		out.ConfidenceThreshold,
	)
	out.Score = computed.Score
	out.Confidence = computed.Confidence
	out.Agreement = computed.Agreement
	out.Coverage = computed.Coverage
	out.ResonanceBonus = computed.Resonance.Bonus
	out.ResonanceActive = computed.Resonance.Active
	return out, nil
}

func resolveAgentConfidence(round Round, stage string) float64 {
	switch stage {
	case "indicator":
		if round.Agents.IndicatorSet {
			return round.Agents.Indicator.MovementConfidence
		}
	case "structure":
		if round.Agents.StructureSet {
			return round.Agents.Structure.MovementConfidence
		}
	case "mechanics":
		if round.Agents.MechanicsSet {
			return round.Agents.Mechanics.MovementConfidence
		}
	}
	sources := mapFromNested(mapFromNested(round.Derived, "direction_consensus"), "sources")
	source := mapFromNested(sources, stage)
	if value, ok := parseutil.FloatOK(source["raw_confidence"]); ok {
		return value
	}
	if value, ok := parseutil.FloatOK(source["confidence"]); ok {
		return value
	}
	return 0
}

func mapFromNested(root map[string]any, key string) map[string]any {
	if len(root) == 0 {
		return nil
	}
	value, ok := root[key]
	if !ok {
		return nil
	}
	out, _ := value.(map[string]any)
	return out
}

func parseBool(value any) (bool, bool) {
	switch raw := value.(type) {
	case bool:
		return raw, true
	case string:
		switch strings.TrimSpace(strings.ToLower(raw)) {
		case "true":
			return true, true
		case "false":
			return false, true
		default:
			return false, false
		}
	case float64:
		return raw != 0, true
	case int:
		return raw != 0, true
	default:
		return false, false
	}
}

func originalGate(rec store.GateEventRecord, derived map[string]any) fund.GateDecision {
	gate := fund.GateDecision{
		GlobalTradeable: rec.GlobalTradeable,
		DecisionAction:  rec.DecisionAction,
		GateReason:      rec.GateReason,
		Direction:       rec.Direction,
		Grade:           rec.Grade,
		Derived:         cloneMap(derived),
	}
	if len(rec.RuleHitJSON) == 0 {
		return gate
	}
	var hit fund.GateRuleHit
	if err := json.Unmarshal(rec.RuleHitJSON, &hit); err == nil {
		gate.RuleHit = &hit
	}
	return gate
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func gateChanged(a, b fund.GateDecision) bool {
	return a.GlobalTradeable != b.GlobalTradeable ||
		!strings.EqualFold(strings.TrimSpace(a.DecisionAction), strings.TrimSpace(b.DecisionAction)) ||
		!strings.EqualFold(strings.TrimSpace(a.GateReason), strings.TrimSpace(b.GateReason)) ||
		!strings.EqualFold(strings.TrimSpace(a.Direction), strings.TrimSpace(b.Direction)) ||
		a.Grade != b.Grade
}

func nextRoundPrice(rounds []Round, idx int) float64 {
	for next := idx + 1; next < len(rounds); next++ {
		if rounds[next].PriceAtDecision > 0 && !math.IsNaN(rounds[next].PriceAtDecision) && !math.IsInf(rounds[next].PriceAtDecision, 0) {
			return rounds[next].PriceAtDecision
		}
	}
	return 0
}
