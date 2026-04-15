package binance

import (
	"testing"
	"time"
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
