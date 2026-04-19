package bootstrap

import "context"

type cleanupStack struct {
	steps []func(context.Context)
}

func (s *cleanupStack) Add(fn func(context.Context)) {
	if s == nil || fn == nil {
		return
	}
	s.steps = append(s.steps, fn)
}

func (s *cleanupStack) Run(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for i := len(s.steps) - 1; i >= 0; i-- {
		s.steps[i](ctx)
	}
}
