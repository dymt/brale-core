package binance

import (
	"context"
	"testing"
	"time"

	"brale-core/internal/market"

	"github.com/adshao/go-binance/v2/futures"
)

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

	stream := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
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

	stream := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
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

	stream := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
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

	stream := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
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

	stream := NewMarkPriceStream(MarkPriceStreamOptions{Symbols: []string{"BTCUSDT"}})
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
