package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"brale-core/internal/config"
	"brale-core/internal/execution"
	"brale-core/internal/runtime"
	"brale-core/internal/transport/notify"

	"go.uber.org/zap"
)

type startupNotifyStub struct {
	calls    int
	err      error
	shutdown int
	info     notify.StartupInfo
}

func (s *startupNotifyStub) SendStartup(_ context.Context, info notify.StartupInfo) error {
	s.calls++
	s.info = info
	return s.err
}

func (s *startupNotifyStub) SendShutdown(_ context.Context, _ notify.ShutdownInfo) error {
	s.shutdown++
	return nil
}

func TestSendStartupNotify_Disabled(t *testing.T) {
	n := &startupNotifyStub{}
	sendStartupNotify(context.Background(), zap.NewNop(), config.SystemConfig{}, config.SymbolIndexConfig{}, nil, nil, coreDeps{}, n)
	if n.calls != 0 {
		t.Fatalf("expected no startup notify call, got %d", n.calls)
	}
}

func TestSendStartupNotify_Enabled(t *testing.T) {
	n := &startupNotifyStub{}
	sys := config.SystemConfig{Notification: config.NotificationConfig{StartupNotifyEnabled: true}}
	idx := config.SymbolIndexConfig{
		Symbols: []config.SymbolIndexEntry{{Symbol: "BTCUSDT"}},
	}
	runtimes := map[string]runtime.SymbolRuntime{
		"BTCUSDT": {Intervals: []string{"15m"}, BarInterval: 15 * time.Minute},
	}
	scheduler := &runtime.RuntimeScheduler{
		Symbols:                 runtimes,
		EnableScheduledDecision: true,
	}
	deps := coreDeps{
		execution: executionDeps{
			scheduled: true,
			freqtradeAcct: func(context.Context, string) (execution.AccountState, error) {
				return execution.AccountState{Equity: 1234.5, Currency: "USDT"}, nil
			},
		},
	}
	sendStartupNotify(context.Background(), zap.NewNop(), sys, idx, runtimes, scheduler, deps, n)
	if n.calls != 1 {
		t.Fatalf("expected startup notify call once, got %d", n.calls)
	}
	if n.info.Balance != 1234.5 || n.info.Currency != "USDT" {
		t.Fatalf("unexpected startup balance info: %+v", n.info)
	}
	if n.info.ScheduleMode != "定时调度" {
		t.Fatalf("schedule mode = %q want 定时调度", n.info.ScheduleMode)
	}
	if len(n.info.SymbolStatuses) != 1 {
		t.Fatalf("symbol statuses = %+v", n.info.SymbolStatuses)
	}
	if n.info.SymbolStatuses[0].Symbol != "BTCUSDT" || n.info.SymbolStatuses[0].NextDecision == "" {
		t.Fatalf("unexpected symbol status: %+v", n.info.SymbolStatuses[0])
	}
}

func TestSendStartupNotify_ErrorStillCalls(t *testing.T) {
	n := &startupNotifyStub{err: errors.New("send failed")}
	sys := config.SystemConfig{Notification: config.NotificationConfig{StartupNotifyEnabled: true}}
	idx := config.SymbolIndexConfig{
		Symbols: []config.SymbolIndexEntry{{Symbol: "BTCUSDT"}},
	}
	sendStartupNotify(context.Background(), zap.NewNop(), sys, idx, nil, nil, coreDeps{}, n)
	if n.calls != 1 {
		t.Fatalf("expected startup notify call once on error, got %d", n.calls)
	}
}

func TestEffectiveLLMCallTimeout(t *testing.T) {
	custom := 45
	zero := 0
	tests := []struct {
		name string
		cfg  config.LLMModelConfig
		want time.Duration
	}{
		{name: "nil timeout uses client default", cfg: config.LLMModelConfig{}, want: 30 * time.Second},
		{name: "zero timeout uses request fallback", cfg: config.LLMModelConfig{TimeoutSec: &zero}, want: 300 * time.Second},
		{name: "positive timeout uses configured value", cfg: config.LLMModelConfig{TimeoutSec: &custom}, want: 45 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveLLMCallTimeout(tt.cfg); got != tt.want {
				t.Fatalf("effectiveLLMCallTimeout()=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestMaxLLMJobTimeout(t *testing.T) {
	t.Run("empty config keeps safe minimum", func(t *testing.T) {
		if got := maxLLMJobTimeout(config.SystemConfig{}); got != 10*time.Minute {
			t.Fatalf("maxLLMJobTimeout(empty)=%v want=%v", got, 10*time.Minute)
		}
	})

	t.Run("configured model budgets retries phases and buffer", func(t *testing.T) {
		timeout := 300
		sys := config.SystemConfig{
			LLMModels: map[string]config.LLMModelConfig{
				"slow": {TimeoutSec: &timeout},
			},
		}
		want := 48*time.Minute + 30*time.Second
		if got := maxLLMJobTimeout(sys); got != want {
			t.Fatalf("maxLLMJobTimeout()=%v want=%v", got, want)
		}
	})
}
