package llmapp

import (
	"context"
	"strings"

	"brale-core/internal/memory"
)

func appendRecentContextPrompt(ctx context.Context, format UserPromptFormat, user string) (string, bool) {
	recent := strings.TrimSpace(memory.PromptContextFromContext(ctx))
	if recent == "" {
		return user, false
	}
	block := formatPayloads(format, payloadBlock{
		label:   "近期决策记忆（仅供参考，禁止直接复用结论）:",
		payload: recent,
	})
	if strings.TrimSpace(user) == "" {
		return block, true
	}
	return user + "\n" + block, true
}

func appendEpisodicContextPrompt(ctx context.Context, format UserPromptFormat, user string) (string, bool) {
	episodic := strings.TrimSpace(memory.EpisodicContextFromContext(ctx))
	if episodic == "" {
		return user, false
	}
	block := formatPayloads(format, payloadBlock{
		label:   "历史交易经验（仅供参考，禁止直接复用结论）:",
		payload: episodic,
	})
	if strings.TrimSpace(user) == "" {
		return block, true
	}
	return user + "\n" + block, true
}

func appendSemanticRulesPrompt(ctx context.Context, system string) (string, bool) {
	rules := strings.TrimSpace(memory.SemanticContextFromContext(ctx))
	if rules == "" {
		return system, false
	}
	return system + "\n\n" + rules, true
}

func applyMemoryPromptContext(ctx context.Context, format UserPromptFormat, system, user string) (string, string) {
	system, _ = appendSemanticRulesPrompt(ctx, system)
	user, _ = appendRecentContextPrompt(ctx, format, user)
	user, _ = appendEpisodicContextPrompt(ctx, format, user)
	return system, user
}

func promptCacheInput(system, user string) []byte {
	system = strings.TrimSpace(system)
	user = strings.TrimSpace(user)
	switch {
	case system == "":
		return []byte(user)
	case user == "":
		return []byte(system)
	default:
		return []byte(system + "\n\n---\n\n" + user)
	}
}
