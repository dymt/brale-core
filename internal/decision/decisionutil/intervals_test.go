package decisionutil

import "testing"

func TestSelectDecisionIntervalPrefersShortestValidInterval(t *testing.T) {
	got := SelectDecisionInterval([]string{"4h", "15m", "1h"})
	if got != "15m" {
		t.Fatalf("got %q want 15m", got)
	}
}

func TestSelectDecisionIntervalFallsBackToFirstWhenNoValidIntervals(t *testing.T) {
	got := SelectDecisionInterval([]string{"foo", "bar"})
	if got != "foo" {
		t.Fatalf("got %q want foo", got)
	}
}

func TestSelectDecisionIntervalHandlesEmptyInput(t *testing.T) {
	if got := SelectDecisionInterval(nil); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
