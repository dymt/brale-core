package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"brale-core/internal/store"
)

type EpisodicMemory struct {
	store       store.EpisodicMemoryStore
	maxPerQuery int
	ttlDays     int
}

func NewEpisodicMemory(s store.EpisodicMemoryStore, maxPerQuery, ttlDays int) *EpisodicMemory {
	if maxPerQuery <= 0 {
		maxPerQuery = 3
	}
	if ttlDays <= 0 {
		ttlDays = 90
	}
	return &EpisodicMemory{store: s, maxPerQuery: maxPerQuery, ttlDays: ttlDays}
}

func (e *EpisodicMemory) ListEpisodes(symbol string, limit int) ([]Episode, error) {
	if limit <= 0 {
		limit = e.maxPerQuery
	}
	records, err := e.store.ListEpisodicMemories(context.Background(), symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("list episodic memories: %w", err)
	}
	episodes := make([]Episode, 0, len(records))
	for _, r := range records {
		episodes = append(episodes, recordToEpisode(r))
	}
	return episodes, nil
}

func (e *EpisodicMemory) SaveEpisode(ep Episode) error {
	rec := episodeToRecord(ep)
	return e.store.SaveEpisodicMemory(context.Background(), &rec)
}

func (e *EpisodicMemory) FormatForPrompt(symbol string, limit int) string {
	if limit <= 0 {
		limit = e.maxPerQuery
	}
	episodes, err := e.ListEpisodes(symbol, limit)
	if err != nil || len(episodes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("历史交易经验（仅供参考，禁止直接复用结论）:\n")
	for i, ep := range episodes {
		b.WriteString(fmt.Sprintf("[%d] %s %s | 入场 %s 出场 %s | PnL %s%% | 持仓 %s\n",
			i+1, ep.Symbol, ep.Direction, ep.EntryPrice, ep.ExitPrice, ep.PnLPercent, ep.Duration))
		if ep.Reflection != "" {
			b.WriteString(fmt.Sprintf("    反思: %s\n", ep.Reflection))
		}
		if len(ep.KeyLessons) > 0 {
			b.WriteString(fmt.Sprintf("    教训: %s\n", strings.Join(ep.KeyLessons, "; ")))
		}
	}
	return b.String()
}

func (e *EpisodicMemory) Cleanup(symbol string) (int64, error) {
	before := time.Now().UTC().AddDate(0, 0, -e.ttlDays)
	return e.store.DeleteEpisodicMemoriesOlderThan(context.Background(), symbol, before)
}

func recordToEpisode(r store.EpisodicMemoryRecord) Episode {
	var lessons []string
	if r.KeyLessons != "" {
		_ = json.Unmarshal([]byte(r.KeyLessons), &lessons)
	}
	return Episode{
		ID:            r.ID,
		Symbol:        r.Symbol,
		PositionID:    r.PositionID,
		Direction:     r.Direction,
		EntryPrice:    r.EntryPrice,
		ExitPrice:     r.ExitPrice,
		PnLPercent:    r.PnLPercent,
		Duration:      r.Duration,
		Reflection:    r.Reflection,
		KeyLessons:    lessons,
		MarketContext: r.MarketContext,
		CreatedAt:     r.CreatedAt,
	}
}

func episodeToRecord(ep Episode) store.EpisodicMemoryRecord {
	lessonsJSON, _ := json.Marshal(ep.KeyLessons)
	return store.EpisodicMemoryRecord{
		Symbol:        ep.Symbol,
		PositionID:    ep.PositionID,
		Direction:     ep.Direction,
		EntryPrice:    ep.EntryPrice,
		ExitPrice:     ep.ExitPrice,
		PnLPercent:    ep.PnLPercent,
		Duration:      ep.Duration,
		Reflection:    ep.Reflection,
		KeyLessons:    string(lessonsJSON),
		MarketContext: ep.MarketContext,
	}
}
