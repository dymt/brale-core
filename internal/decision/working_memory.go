package decision

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"brale-core/internal/config"
	"brale-core/internal/decision/decisionutil"
	"brale-core/internal/decision/features"
	"brale-core/internal/llm"
	"brale-core/internal/memory"
	"brale-core/internal/pkg/logging"
	"go.uber.org/zap"
)

type workingMemoryMarket struct {
	CurrentPrice float64
	ATR          float64
}

func (r *Runner) withWorkingMemoryPromptContext(ctx context.Context, symbol string, comp features.CompressionResult) context.Context {
	if r == nil {
		return ctx
	}
	cfg, err := r.getConfig(symbol)
	if err != nil {
		return ctx
	}
	if r.WorkingMemory != nil {
		prompt := workingMemoryPromptContext(r.WorkingMemory, cfg, symbol, comp)
		if prompt != "" {
			ctx = memory.WithPromptContext(ctx, prompt)
		}
	}
	if r.EpisodicMemory != nil {
		episodicPrompt := r.EpisodicMemory.FormatForPrompt(symbol, 0)
		if episodicPrompt != "" {
			ctx = memory.WithEpisodicContext(ctx, episodicPrompt)
		}
	}
	if r.SemanticMemory != nil {
		semanticPrompt := r.SemanticMemory.FormatForPrompt(symbol, 0)
		if semanticPrompt != "" {
			ctx = memory.WithSemanticContext(ctx, semanticPrompt)
		}
	}
	return ctx
}

func (p *Pipeline) withWorkingMemoryPromptContext(ctx context.Context, symbol string, comp features.CompressionResult) context.Context {
	if p == nil || p.Runner == nil {
		return ctx
	}
	cfg, err := p.Runner.getConfig(symbol)
	if err != nil {
		return ctx
	}
	if p.WorkingMemory != nil {
		prompt := workingMemoryPromptContext(p.WorkingMemory, cfg, symbol, comp)
		if prompt != "" {
			ctx = memory.WithPromptContext(ctx, prompt)
		}
	}
	if p.EpisodicMemory != nil {
		episodicPrompt := p.EpisodicMemory.FormatForPrompt(symbol, 0)
		if episodicPrompt != "" {
			ctx = memory.WithEpisodicContext(ctx, episodicPrompt)
		}
	}
	if p.SemanticMemory != nil {
		semanticPrompt := p.SemanticMemory.FormatForPrompt(symbol, 0)
		if semanticPrompt != "" {
			ctx = memory.WithSemanticContext(ctx, semanticPrompt)
		}
	}
	return ctx
}

func workingMemoryPromptContext(store memory.Store, cfg config.SymbolConfig, symbol string, comp features.CompressionResult) string {
	if store == nil {
		return ""
	}
	market, ok := workingMemoryMarketFromCompression(comp, symbol, decisionutil.SelectDecisionInterval(cfg.Intervals))
	if !ok {
		return ""
	}
	return store.FormatForPrompt(symbol, market.CurrentPrice)
}

func workingMemoryMarketFromCompression(comp features.CompressionResult, symbol, preferredInterval string) (workingMemoryMarket, bool) {
	indJSON, ok := decisionutil.PickIndicatorJSONForInterval(comp, symbol, preferredInterval)
	if !ok {
		return workingMemoryMarket{}, false
	}
	var raw struct {
		Close  float64 `json:"close"`
		ATR    float64 `json:"atr"`
		Market struct {
			CurrentPrice float64 `json:"current_price"`
		} `json:"market"`
		Data struct {
			ATR struct {
				Latest float64 `json:"latest"`
			} `json:"atr"`
		} `json:"data"`
	}
	if err := json.Unmarshal(indJSON.RawJSON, &raw); err != nil {
		return workingMemoryMarket{}, false
	}
	price := raw.Market.CurrentPrice
	if price == 0 {
		price = raw.Close
	}
	if price <= 0 {
		return workingMemoryMarket{}, false
	}
	atr := raw.ATR
	if atr == 0 {
		atr = raw.Data.ATR.Latest
	}
	return workingMemoryMarket{CurrentPrice: price, ATR: atr}, true
}

func (p *Pipeline) recordWorkingMemory(ctx context.Context, res SymbolResult, comp features.CompressionResult) {
	if p == nil || p.WorkingMemory == nil || p.Runner == nil {
		return
	}
	entry, ok := p.buildWorkingMemoryEntry(ctx, res, comp)
	if !ok {
		return
	}
	p.WorkingMemory.Push(res.Symbol, entry)
}

func (p *Pipeline) buildWorkingMemoryEntry(ctx context.Context, res SymbolResult, comp features.CompressionResult) (memory.Entry, bool) {
	cfg, err := p.Runner.getConfig(res.Symbol)
	if err != nil {
		return memory.Entry{}, false
	}
	market, ok := workingMemoryMarketFromCompression(comp, res.Symbol, decisionutil.SelectDecisionInterval(cfg.Intervals))
	if !ok {
		logging.FromContext(ctx).Named("pipeline").Warn("working memory skipped: market snapshot unavailable",
			zap.String("symbol", res.Symbol),
		)
		return memory.Entry{}, false
	}
	roundID := ""
	if currentRoundID, ok := llm.SessionRoundIDFromContext(ctx); ok {
		roundID = currentRoundID.String()
	}
	return memory.Entry{
		RoundID:     roundID,
		Timestamp:   time.Now().UTC(),
		GateAction:  workingMemoryAction(res),
		GateReason:  strings.TrimSpace(res.Gate.GateReason),
		Direction:   workingMemoryDirection(res),
		Score:       res.ConsensusScore,
		PriceAtTime: market.CurrentPrice,
		ATR:         market.ATR,
		KeySignal:   workingMemorySignal(res),
		Outcome:     memory.OutcomePending,
	}, true
}

func workingMemoryAction(res SymbolResult) string {
	action := strings.ToUpper(strings.TrimSpace(res.Gate.DecisionAction))
	if action != "" {
		return action
	}
	if res.Gate.GlobalTradeable {
		return "ALLOW"
	}
	return "VETO"
}

func workingMemoryDirection(res SymbolResult) string {
	switch strings.ToLower(strings.TrimSpace(res.ConsensusDirection)) {
	case "long":
		return "long"
	case "short":
		return "short"
	}
	if res.Plan != nil {
		switch strings.ToLower(strings.TrimSpace(res.Plan.Direction)) {
		case "long":
			return "long"
		case "short":
			return "short"
		}
	}
	return "neutral"
}

func workingMemorySignal(res SymbolResult) string {
	if reason := strings.TrimSpace(res.Gate.GateReason); reason != "" {
		return reason
	}
	if action := strings.TrimSpace(res.Gate.DecisionAction); action != "" {
		return action
	}
	return "decision"
}
