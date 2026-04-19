package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"brale-core/internal/config"
	"brale-core/internal/llm"
	"brale-core/internal/pkg/logging"
	"brale-core/internal/store"

	"go.uber.org/zap"
)

type Reflector struct {
	LLM           llm.Provider
	Episodic      *EpisodicMemory
	Semantic      *SemanticMemory
	Store         store.Store
	SystemPrompt  string
	PromptVersion string
}

type ReflectionInput struct {
	Symbol     string
	PositionID string
	Direction  string
	EntryPrice string
	ExitPrice  string
	PnLPercent string
	Duration   string
	GateReason string
}

type reflectionOutput struct {
	Reflection    string   `json:"reflection"`
	KeyLessons    []string `json:"key_lessons"`
	MarketContext string   `json:"market_context"`
}

func (r *Reflector) Reflect(ctx context.Context, input ReflectionInput) error {
	if r.LLM == nil || r.Episodic == nil {
		return nil
	}
	logger := logging.FromContext(ctx).Named("reflector")

	_, exists, err := r.Episodic.store.FindEpisodicMemoryByPosition(ctx, input.PositionID)
	if err != nil {
		return fmt.Errorf("check existing episode: %w", err)
	}
	if exists {
		logger.Debug("episode already exists", zap.String("position_id", input.PositionID))
		return nil
	}

	userPrompt := formatReflectionUserPrompt(input)
	raw, err := r.LLM.Call(ctx, r.resolveSystemPrompt(), userPrompt)
	if err != nil {
		return fmt.Errorf("reflector llm call: %w", err)
	}

	var output reflectionOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &output); err != nil {
		return fmt.Errorf("parse reflector output: %w", err)
	}

	episode := Episode{
		Symbol:        input.Symbol,
		PositionID:    input.PositionID,
		Direction:     input.Direction,
		EntryPrice:    input.EntryPrice,
		ExitPrice:     input.ExitPrice,
		PnLPercent:    input.PnLPercent,
		Duration:      input.Duration,
		Reflection:    output.Reflection,
		KeyLessons:    output.KeyLessons,
		MarketContext: output.MarketContext,
		CreatedAt:     time.Now().UTC(),
	}
	if err := r.Episodic.SaveEpisode(episode); err != nil {
		return fmt.Errorf("save episode: %w", err)
	}

	logger.Info("reflection completed",
		zap.String("symbol", input.Symbol),
		zap.String("position_id", input.PositionID),
		zap.Int("lessons", len(output.KeyLessons)),
	)

	if r.Semantic != nil {
		r.autoCreateRules(input.Symbol, output.KeyLessons, logger)
	}
	return nil
}

func (r *Reflector) autoCreateRules(symbol string, lessons []string, logger *zap.Logger) {
	for _, lesson := range lessons {
		lesson = strings.TrimSpace(lesson)
		if lesson == "" {
			continue
		}
		rule := Rule{
			Symbol:     symbol,
			RuleText:   lesson,
			Source:     "reflector",
			Confidence: reflectionRuleConfidence(lesson),
			Active:     true,
		}
		if err := r.Semantic.SaveRule(rule); err != nil {
			logger.Warn("auto-create semantic rule failed", zap.Error(err), zap.String("lesson", lesson))
		}
	}
}

func formatReflectionUserPrompt(input ReflectionInput) string {
	return fmt.Sprintf(
		"交易对: %s\n方向: %s\n入场价: %s\n出场价: %s\nPnL%%: %s\n持仓时间: %s\n入场原因: %s",
		input.Symbol, input.Direction, input.EntryPrice, input.ExitPrice,
		input.PnLPercent, input.Duration, input.GateReason,
	)
}

func (r *Reflector) resolveSystemPrompt() string {
	if prompt := strings.TrimSpace(r.SystemPrompt); prompt != "" {
		return prompt
	}
	return config.DefaultPromptDefaults().ReflectorAnalysis
}

func reflectionRuleConfidence(lesson string) float64 {
	text := strings.ToLower(strings.TrimSpace(lesson))
	if text == "" {
		return 0.3
	}

	score := 0.45
	if utf8.RuneCountInString(text) >= 16 {
		score += 0.05
	}
	if strings.ContainsAny(text, "0123456789") {
		score += 0.05
	}
	if containsAny(text, "如果", "当", "若", "一旦", "避免", "不要", "必须", "应当", "需要") {
		score += 0.05
	}
	if containsAny(text, "止损", "止盈", "入场", "出场", "仓位", "回踩", "突破", "趋势", "风险", "确认", "失效", "结构") {
		score += 0.05
	}
	if containsAny(text, "保持纪律", "控制情绪", "顺势而为", "注意风险", "关注市场", "耐心等待") {
		score -= 0.10
	}
	if utf8.RuneCountInString(text) < 10 {
		score -= 0.05
	}
	return clampReflectionConfidence(score)
}

func containsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func clampReflectionConfidence(score float64) float64 {
	switch {
	case score < 0.3:
		return 0.3
	case score > 0.7:
		return 0.7
	default:
		return score
	}
}
