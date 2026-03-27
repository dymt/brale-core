package runtime

import (
	"context"
	"sync"

	"brale-core/internal/market"
)

type RuntimeService interface {
	Start(ctx context.Context)
	Stop()
}

type metricsRuntimeService struct {
	service *market.MetricsService
	symbol  string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func newMetricsRuntimeService(service *market.MetricsService, symbol string) RuntimeService {
	if service == nil {
		return nil
	}
	return &metricsRuntimeService{service: service, symbol: NormalizeSymbol(symbol)}
}

func (s *metricsRuntimeService) Start(ctx context.Context) {
	if s == nil || s.service == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	serviceCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()
	s.service.RefreshSymbol(serviceCtx, s.symbol)
	go s.service.Start(serviceCtx)
}

func (s *metricsRuntimeService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
