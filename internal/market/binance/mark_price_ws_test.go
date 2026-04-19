package binance

import (
	"context"
	"testing"
	"time"

	"brale-core/internal/market"

	"github.com/adshao/go-binance/v2/futures"
)

func mustNewMarkPriceStream(t *testing.T, opts MarkPriceStreamOptions) *MarkPriceStream {
	t.Helper()
	stream, err := NewMarkPriceStream(opts)
	if err != nil {
		t.Fatalf("NewMarkPriceStream() error = %v", err)
	}
	return stream
}

func TestNewMarkPriceStreamRejectsInvalidRate(t *testing.T) {
	t.Parallel()

	tests := []time.Duration{
		2 * time.Second,
		-time.Second,
	}
	for _, rate := range tests {
		rate := rate
		t.Run(rate.String(), func(t *testing.T) {
			if _, err := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}, Rate: rate}); err == nil {
				t.Fatalf("expected error for invalid rate %v", rate)
			}
		})
	}
}

func TestSignalMarkPriceStopReturnsWhenDoneClosed(t *testing.T) {
	t.Parallel()

	doneC := make(chan struct{})
	stopC := make(chan struct{})
	close(doneC)

	start := time.Now()
	if sent := signalMarkPriceStop(doneC, stopC); sent {
		t.Fatal("signalMarkPriceStop() = true, want false when stream is already done")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("signalMarkPriceStop() took too long after done close: %v", elapsed)
	}
}

func TestSignalMarkPriceStopDeliversStopSignal(t *testing.T) {
	t.Parallel()

	doneC := make(chan struct{})
	stopC := make(chan struct{})
	received := make(chan struct{})

	go func() {
		<-stopC
		close(received)
	}()

	if sent := signalMarkPriceStop(doneC, stopC); !sent {
		t.Fatal("signalMarkPriceStop() = false, want true when stop signal is delivered")
	}
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stop signal")
	}
}

func TestSignalMarkPriceStopDoesNotBlockWithoutReceiver(t *testing.T) {
	t.Parallel()

	doneC := make(chan struct{})
	stopC := make(chan struct{})

	start := time.Now()
	if sent := signalMarkPriceStop(doneC, stopC); !sent {
		t.Fatal("signalMarkPriceStop() = false, want true when stop channel is closed")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("signalMarkPriceStop() blocked too long without a stop receiver: %v", elapsed)
	}
	select {
	case <-stopC:
	default:
		t.Fatal("expected stop channel to be closed")
	}
}

func TestMarkPriceRejectsStaleQuote(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     100,
		Timestamp: time.Now().Add(-(defaultMarkPriceMaxAge + time.Second)).UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	_, err := stream.MarkPrice(context.Background(), "BTCUSDT")
	if err == nil {
		t.Fatal("expected stale quote error")
	}
}

func TestMarkPriceAcceptsFreshQuote(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     101,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	quote, err := stream.MarkPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("MarkPrice() error = %v", err)
	}
	if quote.Price != 101 {
		t.Fatalf("price=%v want 101", quote.Price)
	}
}

func TestHandleEventRejectsSpike(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     100,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	stream.handleEvent(&futures.WsMarkPriceEvent{
		Symbol:    "BTCUSDT",
		MarkPrice: "140",
		Time:      time.Now().UnixMilli(),
	})

	quote, err := stream.MarkPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("MarkPrice() error = %v", err)
	}
	if quote.Price != 100 {
		t.Fatalf("price=%v want previous 100 after spike rejection", quote.Price)
	}
}

func TestHandleEventAcceptsNormalMove(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     100,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	stream.handleEvent(&futures.WsMarkPriceEvent{
		Symbol:    "BTCUSDT",
		MarkPrice: "110",
		Time:      time.Now().UnixMilli(),
	})

	quote, err := stream.MarkPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("MarkPrice() error = %v", err)
	}
	if quote.Price != 110 {
		t.Fatalf("price=%v want 110", quote.Price)
	}
}

func TestHandleEventAcceptsLargeMoveWhenPreviousQuoteIsStale(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     100,
		Timestamp: time.Now().Add(-(defaultMarkPriceMaxAge + time.Second)).UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	stream.handleEvent(&futures.WsMarkPriceEvent{
		Symbol:    "BTCUSDT",
		MarkPrice: "140",
		Time:      time.Now().UnixMilli(),
	})

	quote, err := stream.MarkPrice(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("MarkPrice() error = %v", err)
	}
	if quote.Price != 140 {
		t.Fatalf("price=%v want 140 after replacing stale quote", quote.Price)
	}
}

func TestStreamStatusReturnsNotFoundWithoutQuote(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	if _, found := stream.StreamStatus("BTCUSDT"); found {
		t.Fatal("StreamStatus() found=true, want false when no quote has been cached")
	}
}

func TestStreamStatusSeparatesRunningFromConnection(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.running.Store(true)
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     101,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	status, found := stream.StreamStatus("BTCUSDT")
	if !found {
		t.Fatal("StreamStatus() found=false, want true when a fresh quote exists")
	}
	if status.Connected {
		t.Fatalf("Connected=%v want false before websocket connection is established", status.Connected)
	}
}

func TestStreamStatusReportsConnectedFreshQuote(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stream.running.Store(true)
	stream.connected.Store(true)
	stream.mu.Lock()
	stream.quotes["BTCUSDT"] = market.PriceQuote{
		Symbol:    "BTCUSDT",
		Price:     102,
		Timestamp: time.Now().UnixMilli(),
		Source:    "test",
	}
	stream.mu.Unlock()

	status, found := stream.StreamStatus("BTCUSDT")
	if !found {
		t.Fatal("StreamStatus() found=false, want true when a fresh quote exists")
	}
	if !status.Connected {
		t.Fatal("Connected=false want true after websocket connection is marked active")
	}
	if !status.Fresh {
		t.Fatal("Fresh=false want true for a fresh quote")
	}
	if status.LastPrice != 102 {
		t.Fatalf("LastPrice=%v want 102", status.LastPrice)
	}
}

func TestSetConnectedIgnoresStaleRun(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	staleRunID := stream.nextRunID()
	freshRunID := stream.nextRunID()

	stream.setConnected(staleRunID, true)
	if stream.connected.Load() {
		t.Fatal("stale run unexpectedly updated connected state")
	}

	stream.setConnected(freshRunID, true)
	if !stream.connected.Load() {
		t.Fatal("fresh run failed to set connected state")
	}

	stream.setConnected(staleRunID, false)
	if !stream.connected.Load() {
		t.Fatal("stale run cleared connected state for the active run")
	}
}

func TestWaitRetryUsesRunStopChannel(t *testing.T) {
	t.Parallel()

	stream := mustNewMarkPriceStream(t, MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
	stopCh := make(chan struct{})
	close(stopCh)

	if ok := stream.waitRetry(context.Background(), stopCh, time.Second); ok {
		t.Fatal("waitRetry() = true, want false when the run stop channel is closed")
	}
}
