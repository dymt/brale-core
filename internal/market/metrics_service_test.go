package market

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMetricsParseTimeframe(t *testing.T) {
	valid := []struct {
		input string
		want  time.Duration
	}{
		{input: "1m", want: time.Minute},
		{input: "2h", want: 2 * time.Hour},
		{input: "3d", want: 72 * time.Hour},
	}
	for _, tt := range valid {
		got, err := parseTimeframe(tt.input)
		if err != nil {
			t.Fatalf("parseTimeframe(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("parseTimeframe(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}

	invalid := []string{"1s", "0m", "-1m", "1H", " 1h", "1h ", "1w", ""}
	for _, input := range invalid {
		if _, err := parseTimeframe(input); err == nil {
			t.Fatalf("parseTimeframe(%q) expected error", input)
		}
	}
}

type blockingFundingSource struct {
	started      chan struct{}
	startedOnce  sync.Once
	inFlight     atomic.Int32
	maxInFlight  atomic.Int32
	fundingCalls atomic.Int32
}

func (s *blockingFundingSource) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	s.fundingCalls.Add(1)
	current := s.inFlight.Add(1)
	for {
		maxSeen := s.maxInFlight.Load()
		if current <= maxSeen {
			break
		}
		if s.maxInFlight.CompareAndSwap(maxSeen, current) {
			break
		}
	}
	s.startedOnce.Do(func() { close(s.started) })
	<-ctx.Done()
	s.inFlight.Add(-1)
	return 0, ctx.Err()
}

func (s *blockingFundingSource) GetOpenInterestHistory(ctx context.Context, symbol, period string, limit int) ([]OpenInterestPoint, error) {
	return nil, nil
}

func TestMetricsServiceStartCancellationBoundsInFlightWork(t *testing.T) {
	source := &blockingFundingSource{started: make(chan struct{})}
	svc, err := NewMetricsService(source, []string{"BTCUSDT"}, []string{"1h"})
	if err != nil {
		t.Fatalf("NewMetricsService error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewMetricsService returned nil service")
	}

	svc.baseOIHistoryPeriod = ""
	svc.oiHistoryLimit = 0
	svc.pollInterval = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		svc.Start(ctx)
		close(done)
	}()

	select {
	case <-source.started:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for first metrics update")
	}

	time.Sleep(2200 * time.Millisecond)

	if got := source.maxInFlight.Load(); got > 1 {
		t.Fatalf("in-flight funding updates = %d, want <= 1", got)
	}
	if got := source.fundingCalls.Load(); got != 1 {
		t.Fatalf("funding calls = %d, want 1 while first update is blocked", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("metrics service did not stop after cancellation")
	}
}

type cancelAwareSource struct{}

func (cancelAwareSource) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}

func (cancelAwareSource) GetOpenInterestHistory(ctx context.Context, symbol, period string, limit int) ([]OpenInterestPoint, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestMetricsServiceRefreshSymbolRespectsCallerCancellation(t *testing.T) {
	svc, err := NewMetricsService(cancelAwareSource{}, []string{"BTCUSDT"}, []string{"1h"})
	if err != nil {
		t.Fatalf("NewMetricsService error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewMetricsService returned nil service")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	svc.RefreshSymbol(ctx, "BTCUSDT")
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("RefreshSymbol took too long after cancellation: %v", elapsed)
	}

	data, ok := svc.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected refreshed symbol in cache")
	}
	if !strings.Contains(data.Error, context.Canceled.Error()) {
		t.Fatalf("expected cancellation error in cache, got %q", data.Error)
	}
}

func TestMetricsServiceRefreshSymbolAddsOperationContextToErrors(t *testing.T) {
	svc, err := NewMetricsService(cancelAwareSource{}, []string{"BTCUSDT"}, []string{"1h"})
	if err != nil {
		t.Fatalf("NewMetricsService error: %v", err)
	}
	if svc == nil {
		t.Fatal("NewMetricsService returned nil service")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc.RefreshSymbol(ctx, "BTCUSDT")

	data, ok := svc.Get("BTCUSDT")
	if !ok {
		t.Fatal("expected refreshed symbol in cache")
	}
	if !strings.Contains(data.Error, "get OI history for BTCUSDT") {
		t.Fatalf("expected OI operation context in error, got %q", data.Error)
	}
	if !strings.Contains(data.Error, "get funding rate for BTCUSDT") {
		t.Fatalf("expected funding operation context in error, got %q", data.Error)
	}
}
