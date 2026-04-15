package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"brale-core/internal/llm"
	"brale-core/internal/pkg/logging"
	"brale-core/internal/store"

	"go.uber.org/zap"
)

type Reflector struct {
	LLM      llm.Provider
	Episodic *EpisodicMemory
	Semantic *SemanticMemory
	Store    store.Store
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

const reflectorSystemPrompt = "你是一个量化交易回顾分析师。基于提供的交易数据，提取关键经验教训。\n" +
	"硬性输出规则：\n" +
	"- 只输出一个 JSON 对象\n" +
	"- 字段：reflection (string, 2-3句交易总结), key_lessons (string array, 3-5条可操作经验), market_context (string, 当时市场环境简述)\n" +
	"- 禁止输出 markdown、代码块、注释\n" +
	"- 经验教训应具有可操作性和泛化性"

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
	raw, err := r.LLM.Call(ctx, reflectorSystemPrompt, userPrompt)
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
		r.autoCreateRules(ctx, input.Symbol, output.KeyLessons, logger)
	}
	return nil
}

func (r *Reflector) autoCreateRules(ctx context.Context, symbol string, lessons []string, logger *zap.Logger) {
	for _, lesson := range lessons {
		lesson = strings.TrimSpace(lesson)
		if lesson == "" {
			continue
		}
		rule := Rule{
			Symbol:     symbol,
			RuleText:   lesson,
			Source:     "reflector",
			Confidence: 0.5,
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
