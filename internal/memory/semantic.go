package memory

import (
	"context"
	"fmt"
	"strings"

	"brale-core/internal/store"
)

type SemanticMemory struct {
	store    store.SemanticMemoryStore
	maxRules int
}

func NewSemanticMemory(s store.SemanticMemoryStore, maxRules int) *SemanticMemory {
	if maxRules <= 0 {
		maxRules = 10
	}
	return &SemanticMemory{store: s, maxRules: maxRules}
}

func (m *SemanticMemory) ListRules(symbol string, activeOnly bool, limit int) ([]Rule, error) {
	if limit <= 0 {
		limit = m.maxRules
	}
	records, err := m.store.ListSemanticMemories(context.Background(), symbol, activeOnly, limit)
	if err != nil {
		return nil, fmt.Errorf("list semantic memories: %w", err)
	}
	rules := make([]Rule, 0, len(records))
	for _, r := range records {
		rules = append(rules, recordToRule(r))
	}
	return rules, nil
}

func (m *SemanticMemory) SaveRule(rule Rule) error {
	rec := ruleToRecord(rule)
	return m.store.SaveSemanticMemory(context.Background(), &rec)
}

func (m *SemanticMemory) UpdateRule(id uint, updates map[string]any) error {
	return m.store.UpdateSemanticMemory(context.Background(), id, updates)
}

func (m *SemanticMemory) DeleteRule(id uint) error {
	return m.store.DeleteSemanticMemory(context.Background(), id)
}

func (m *SemanticMemory) ToggleRule(id uint) error {
	rec, ok, err := m.store.FindSemanticMemory(context.Background(), id)
	if err != nil {
		return fmt.Errorf("find semantic memory: %w", err)
	}
	if !ok {
		return fmt.Errorf("semantic memory not found: %d", id)
	}
	return m.store.UpdateSemanticMemory(context.Background(), id, map[string]any{"active": !rec.Active})
}

func (m *SemanticMemory) FormatForPrompt(symbol string, limit int) string {
	if limit <= 0 {
		limit = m.maxRules
	}
	rules, err := m.ListRules(symbol, true, limit)
	if err != nil || len(rules) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("交易规则与经验（必须参考，但可基于当前证据推翻）:\n")
	for i, rule := range rules {
		scope := "全局"
		if rule.Symbol != "" {
			scope = rule.Symbol
		}
		b.WriteString(fmt.Sprintf("[%d] [%s] (置信度 %.1f) %s\n", i+1, scope, rule.Confidence, rule.RuleText))
	}
	return b.String()
}

func recordToRule(r store.SemanticMemoryRecord) Rule {
	return Rule{
		ID:         r.ID,
		Symbol:     r.Symbol,
		RuleText:   r.RuleText,
		Source:     r.Source,
		Confidence: r.Confidence,
		Active:     r.Active,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

func ruleToRecord(r Rule) store.SemanticMemoryRecord {
	return store.SemanticMemoryRecord{
		Symbol:     r.Symbol,
		RuleText:   r.RuleText,
		Source:     r.Source,
		Confidence: r.Confidence,
		Active:     r.Active,
	}
}
