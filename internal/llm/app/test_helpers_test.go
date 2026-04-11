package llmapp

import (
	"context"
)

type stubLLMProvider struct {
	resp string
	err  error
}

func (p stubLLMProvider) Call(context.Context, string, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return p.resp, nil
}
