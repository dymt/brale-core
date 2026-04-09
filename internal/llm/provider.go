package llm

import "context"

type Provider interface {
	Call(ctx context.Context, system, user string) (string, error)
}

type StructuredProvider interface {
	Provider
	CallStructured(ctx context.Context, system, user string, schema *JSONSchema) (string, error)
}
