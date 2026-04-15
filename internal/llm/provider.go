package llm

import "context"

type Provider interface {
	Call(ctx context.Context, system, user string) (string, error)
}

type StructuredProvider interface {
	Provider
	CallStructured(ctx context.Context, system, user string, schema *JSONSchema) (string, error)
}

// MultiTurnProvider extends Provider with support for multi-turn
// conversations. Used by reflection and interactive workflows.
type MultiTurnProvider interface {
	Provider
	CallMultiTurn(ctx context.Context, messages []ChatMessage) (string, error)
}
