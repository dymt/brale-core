package memory

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const defaultMaxSize = 5

type WorkingMemory struct {
	mu      sync.RWMutex
	history map[string][]Entry
	maxSize int
	now     func() time.Time
}

func NewWorkingMemory(maxSize int) *WorkingMemory {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &WorkingMemory{
		history: make(map[string][]Entry),
		maxSize: maxSize,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (wm *WorkingMemory) Push(symbol string, entry Entry) {
	if wm == nil {
		return
	}
	key := normalizeSymbol(symbol)
	if key == "" {
		return
	}
	entry = wm.normalizeEntry(entry)

	wm.mu.Lock()
	defer wm.mu.Unlock()

	entries := wm.history[key]
	if n := len(entries); n > 0 && entry.PriceAtTime > 0 {
		prev := entries[n-1]
		prev.PriceNow = entry.PriceAtTime
		prev.Outcome = classifyOutcome(prev, entry.PriceAtTime)
		entries[n-1] = prev
	}
	entries = append(entries, entry)
	if len(entries) > wm.maxSize {
		entries = append([]Entry(nil), entries[len(entries)-wm.maxSize:]...)
	}
	wm.history[key] = entries
}

func (wm *WorkingMemory) Entries(symbol string) []Entry {
	if wm == nil {
		return nil
	}
	key := normalizeSymbol(symbol)
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return append([]Entry(nil), wm.history[key]...)
}

func (wm *WorkingMemory) FormatForPrompt(symbol string, currentPrice float64) string {
	if wm == nil {
		return ""
	}
	key := normalizeSymbol(symbol)
	wm.mu.RLock()
	entries := append([]Entry(nil), wm.history[key]...)
	now := wm.currentTime()
	wm.mu.RUnlock()
	if len(entries) == 0 {
		return ""
	}
	lines := make([]string, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		if i == len(entries)-1 && currentPrice > 0 {
			entry.PriceNow = currentPrice
			entry.Outcome = classifyOutcome(entry, currentPrice)
		}
		lines = append(lines, formatPromptEntry(now, entry))
	}
	return strings.Join(lines, "\n")
}

func (wm *WorkingMemory) normalizeEntry(entry Entry) Entry {
	entry.RoundID = strings.TrimSpace(entry.RoundID)
	entry.GateAction = strings.ToUpper(strings.TrimSpace(entry.GateAction))
	entry.GateReason = strings.TrimSpace(entry.GateReason)
	entry.Direction = normalizeDirection(entry.Direction)
	entry.KeySignal = strings.TrimSpace(entry.KeySignal)
	if entry.Timestamp.IsZero() {
		entry.Timestamp = wm.currentTime()
	}
	entry.Timestamp = entry.Timestamp.UTC()
	if entry.Outcome == "" {
		entry.Outcome = OutcomePending
	}
	return entry
}

func (wm *WorkingMemory) currentTime() time.Time {
	if wm != nil && wm.now != nil {
		return wm.now().UTC()
	}
	return time.Now().UTC()
}

func formatPromptEntry(now time.Time, entry Entry) string {
	age := formatAge(now, entry.Timestamp)
	priceNow := "pending"
	if entry.PriceNow > 0 {
		priceNow = fmt.Sprintf("%.2f", entry.PriceNow)
	}
	parts := []string{
		fmt.Sprintf("- [%s]", age),
		fmt.Sprintf("dir=%s", normalizeDirection(entry.Direction)),
	}
	if entry.GateAction != "" {
		parts = append(parts, fmt.Sprintf("action=%s", entry.GateAction))
	}
	parts = append(parts, fmt.Sprintf("score=%.2f", entry.Score))
	if entry.PriceAtTime > 0 {
		parts = append(parts, fmt.Sprintf("price=%.2f->%s", entry.PriceAtTime, priceNow))
	}
	parts = append(parts, fmt.Sprintf("outcome=%s", entry.Outcome))
	if entry.GateReason != "" {
		parts = append(parts, fmt.Sprintf("reason=%s", entry.GateReason))
	}
	if entry.KeySignal != "" && entry.KeySignal != entry.GateReason {
		parts = append(parts, fmt.Sprintf("signal=%s", entry.KeySignal))
	}
	return strings.Join(parts, " ")
}

func formatAge(now, ts time.Time) string {
	if ts.IsZero() {
		return "unknown"
	}
	if now.Before(ts) {
		return "0m前"
	}
	age := now.Sub(ts)
	if age < time.Hour {
		return fmt.Sprintf("%dm前", int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%dh前", int(age.Hours()))
	}
	return fmt.Sprintf("%dd前", int(age.Hours()/24))
}

func classifyOutcome(entry Entry, currentPrice float64) Outcome {
	if currentPrice <= 0 || entry.PriceAtTime <= 0 || entry.ATR <= 0 {
		return OutcomePending
	}
	move := currentPrice - entry.PriceAtTime
	switch normalizeDirection(entry.Direction) {
	case "long":
		if move >= entry.ATR {
			return OutcomeCorrect
		}
		if move <= -entry.ATR {
			return OutcomeWrong
		}
	case "short":
		if move <= -entry.ATR {
			return OutcomeCorrect
		}
		if move >= entry.ATR {
			return OutcomeWrong
		}
	case "neutral":
		if math.Abs(move) <= entry.ATR {
			return OutcomeCorrect
		}
		return OutcomeWrong
	}
	return OutcomePending
}

func normalizeDirection(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "long":
		return "long"
	case "short":
		return "short"
	default:
		return "neutral"
	}
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}
