package onboarding

import (
	"reflect"
	"testing"
)

func TestRunStartupMonitorPrefersComposeServiceTargets(t *testing.T) {
	oldProbe := startupProbeReachable
	defer func() { startupProbeReachable = oldProbe }()

	var calls [][]string
	startupProbeReachable = func(targets []string) bool {
		copied := append([]string(nil), targets...)
		calls = append(calls, copied)
		return len(calls) == 1
	}

	got := runStartupMonitor()

	wantCalls := [][]string{
		{"http://brale:9991/healthz", "http://127.0.0.1:9991/healthz"},
		{"http://freqtrade:8080/api/v1/ping", "http://127.0.0.1:8080/api/v1/ping"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("runStartupMonitor() probe targets = %#v, want %#v", calls, wantCalls)
	}
	if !got.BraleRunning {
		t.Fatal("runStartupMonitor() BraleRunning = false, want true")
	}
	if got.FreqtradeRunning {
		t.Fatal("runStartupMonitor() FreqtradeRunning = true, want false")
	}
	if got.BraleURL != braleDashboardURL {
		t.Fatalf("runStartupMonitor() BraleURL = %q, want %q", got.BraleURL, braleDashboardURL)
	}
	if got.FreqtradeURL != freqtradeURL {
		t.Fatalf("runStartupMonitor() FreqtradeURL = %q, want %q", got.FreqtradeURL, freqtradeURL)
	}
}
