package config

import "testing"

func TestTrendPresetRequiredBars_EmptyIntervals(t *testing.T) {
	got := TrendPresetRequiredBars(nil)
	if got != 1 {
		t.Fatalf("required bars=%d, want 1", got)
	}
}
