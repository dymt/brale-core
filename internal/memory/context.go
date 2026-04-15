package memory

import (
	"context"
	"strings"
)

type promptContextKey struct{}
type episodicContextKey struct{}
type semanticContextKey struct{}

func WithPromptContext(ctx context.Context, prompt string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(prompt) == "" {
		return ctx
	}
	return context.WithValue(ctx, promptContextKey{}, prompt)
}

func PromptContextFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(promptContextKey{}).(string)
	return strings.TrimSpace(value)
}

func WithEpisodicContext(ctx context.Context, prompt string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(prompt) == "" {
		return ctx
	}
	return context.WithValue(ctx, episodicContextKey{}, prompt)
}

func EpisodicContextFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(episodicContextKey{}).(string)
	return strings.TrimSpace(value)
}

func WithSemanticContext(ctx context.Context, prompt string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(prompt) == "" {
		return ctx
	}
	return context.WithValue(ctx, semanticContextKey{}, prompt)
}

func SemanticContextFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(semanticContextKey{}).(string)
	return strings.TrimSpace(value)
}
