package memory

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWorkingMemoryPushCapsEntriesAndUpdatesPreviousOutcome(t *testing.T) {
	base := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	wm := NewWorkingMemory(2)
	wm.now = func() time.Time { return base.Add(3 * time.Hour) }

	wm.Push("BTCUSDT", Entry{
		RoundID:     "round-1",
		Timestamp:   base,
		GateAction:  "ALLOW",
		GateReason:  "BREAKOUT",
		Direction:   "long",
		Score:       0.62,
		PriceAtTime: 100,
		ATR:         5,
	})
	wm.Push("BTCUSDT", Entry{
		RoundID:     "round-2",
		Timestamp:   base.Add(time.Hour),
		GateAction:  "ALLOW",
		GateReason:  "RETEST",
		Direction:   "short",
		Score:       0.58,
		PriceAtTime: 107,
		ATR:         4,
	})
	wm.Push("BTCUSDT", Entry{
		RoundID:     "round-3",
		Timestamp:   base.Add(2 * time.Hour),
		GateAction:  "VETO",
		GateReason:  "MIXED",
		Direction:   "neutral",
		Score:       0.31,
		PriceAtTime: 95,
		ATR:         4,
	})

	entries := wm.Entries("BTCUSDT")
	if len(entries) != 2 {
		t.Fatalf("entries=%d want 2", len(entries))
	}
	if entries[0].RoundID != "round-2" {
		t.Fatalf("oldest round=%q want round-2", entries[0].RoundID)
	}
	if entries[0].PriceNow != 95 {
		t.Fatalf("round-2 price_now=%v want 95", entries[0].PriceNow)
	}
	if entries[0].Outcome != OutcomeCorrect {
		t.Fatalf("round-2 outcome=%q want %q", entries[0].Outcome, OutcomeCorrect)
	}
	if entries[1].RoundID != "round-3" {
		t.Fatalf("latest round=%q want round-3", entries[1].RoundID)
	}
	if entries[1].Outcome != OutcomePending {
		t.Fatalf("round-3 outcome=%q want %q", entries[1].Outcome, OutcomePending)
	}
}

func TestWorkingMemoryFormatForPromptUsesCurrentPricePreview(t *testing.T) {
	base := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	wm := NewWorkingMemory(3)
	wm.now = func() time.Time { return base.Add(2 * time.Hour) }

	wm.Push("BTCUSDT", Entry{
		RoundID:     "round-1",
		Timestamp:   base,
		GateAction:  "ALLOW",
		GateReason:  "BREAKOUT",
		Direction:   "long",
		Score:       0.62,
		PriceAtTime: 100,
		ATR:         5,
	})
	wm.Push("BTCUSDT", Entry{
		RoundID:     "round-2",
		Timestamp:   base.Add(time.Hour),
		GateAction:  "ALLOW",
		GateReason:  "REVERSAL",
		Direction:   "short",
		Score:       0.57,
		PriceAtTime: 112,
		ATR:         4,
	})

	formatted := wm.FormatForPrompt("BTCUSDT", 104)
	if !strings.Contains(formatted, "price=100.00->112.00 outcome=correct") {
		t.Fatalf("formatted prompt missing settled first entry: %s", formatted)
	}
	if !strings.Contains(formatted, "price=112.00->104.00 outcome=correct") {
		t.Fatalf("formatted prompt missing preview for latest entry: %s", formatted)
	}
	if !strings.Contains(formatted, "reason=REVERSAL") {
		t.Fatalf("formatted prompt missing gate reason: %s", formatted)
	}
}

func TestWorkingMemoryConcurrentAccess(t *testing.T) {
	wm := NewWorkingMemory(5)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				wm.Push("BTCUSDT", Entry{
					RoundID:     "round",
					Timestamp:   time.Unix(int64(j), 0).UTC(),
					GateAction:  "ALLOW",
					GateReason:  "TEST",
					Direction:   "long",
					Score:       0.5,
					PriceAtTime: float64(100 + id + j),
					ATR:         2,
				})
				_ = wm.FormatForPrompt("BTCUSDT", 110)
			}
		}(i)
	}
	wg.Wait()

	if got := len(wm.Entries("BTCUSDT")); got > 5 {
		t.Fatalf("entries=%d want <= 5", got)
	}
}
