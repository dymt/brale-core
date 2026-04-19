package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"brale-core/internal/config"
	"brale-core/internal/jobs"
	braleOtel "brale-core/internal/otel"
	"brale-core/internal/runtime"
	"brale-core/internal/transport"
	"brale-core/internal/transport/notify"
	"brale-core/internal/transport/runtimeapi"
	dashboard "brale-core/webui/dashboard"
	decisionview "brale-core/webui/decision-view"

	"go.uber.org/zap"
)

type startupNotifier interface {
	SendStartup(ctx context.Context, info notify.StartupInfo) error
	SendShutdown(ctx context.Context, info notify.ShutdownInfo) error
}

type Options struct {
	SystemPath      string
	SymbolIndexPath string
}

type appRuntime struct {
	startedAt      time.Time
	env            appEnv
	deps           coreDeps
	asyncNotifier  *notify.AsyncManager
	runtimes       map[string]runtime.SymbolRuntime
	viewerHandler  http.Handler
	dashboardHandler http.Handler
	resolver       *runtimeapi.RuntimeSymbolResolver
	symbolConfigs  map[string]runtimeapi.ConfigBundle
	cleanup        cleanupStack
}

func Run(baseCtx context.Context, opts Options) error {
	state, err := prepareApp(baseCtx, time.Now(), opts)
	if err != nil {
		return err
	}
	if err := prepareRuntime(&state); err != nil {
		state.cleanup.Run(context.Background())
		return err
	}
	if err := startBackgroundServices(&state); err != nil {
		state.cleanup.Run(context.Background())
		return err
	}
	return waitForShutdown(&state)
}

func prepareApp(baseCtx context.Context, startedAt time.Time, opts Options) (appRuntime, error) {
	systemPath, symbolIndexPath := resolveRunPaths(opts)
	env, err := bootstrapAppEnv(baseCtx, systemPath, symbolIndexPath)
	if err != nil {
		return appRuntime{}, err
	}
	deps, err := buildCoreDeps(env.ctx, env.logger, env)
	if err != nil {
		return appRuntime{}, err
	}
	state := appRuntime{
		startedAt: startedAt,
		env:       env,
		deps:      deps,
	}
	if deps.closeDB != nil {
		state.cleanup.Add(func(context.Context) { deps.closeDB() })
	}
	state.asyncNotifier = notify.NewAsyncManager(nil, env.notifier, env.logger.Named("notify-async"))
	wireAsyncNotifier(&state)
	otelShutdown, otelErr := braleOtel.Init(env.ctx, env.sys.Telemetry, env.logger)
	if otelErr != nil {
		env.logger.Error("otel init failed, continuing without telemetry", zap.Error(otelErr))
	} else {
		state.cleanup.Add(func(context.Context) {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				env.logger.Error("otel shutdown error", zap.Error(err))
			}
		})
	}
	return state, nil
}

func prepareRuntime(state *appRuntime) error {
	runScheduledWarmup(state.env.ctx, state.env.logger, state.deps)
	state.viewerHandler = decisionview.StartDecisionViewer(state.env.logger, state.env.sys, state.env.symbolIndexPath, state.env.index, state.deps.persistence.store)
	state.dashboardHandler = dashboard.Start()
	state.runtimes = buildRuntimeMap(state.env.ctx, state.env.logger, state.env.sys, state.env.symbolIndexPath, state.env.index, state.deps)
	state.resolver = buildRuntimeResolver(state.env.ctx, state.env.logger, state.env.sys, state.env.symbolIndexPath, state.env.index, state.deps, state.runtimes)
	state.symbolConfigs = loadSymbolConfigs(state.env.logger, state.env.sys, state.env.symbolIndexPath, state.env.index)
	return nil
}

func startBackgroundServices(state *appRuntime) error {
	runFreqtradeBalanceCheck(state.env.ctx, state.env.logger, state.deps)
	scheduler, err := startScheduler(state.env.ctx, state.env.logger, state.env.sys, state.deps, state.runtimes)
	if err != nil {
		return err
	}
	state.cleanup.Add(func(context.Context) { scheduler.Stop() })
	if migrateErr := jobs.RunMigrations(state.env.ctx, state.deps.persistence.pool); migrateErr != nil {
		return fmt.Errorf("river migration failed: %w", migrateErr)
	}
	riverWorkers := jobs.RegisterWorkers(
		func(ctx context.Context, symbol string) error { return runtime.RunObserveOnce(ctx, scheduler, symbol) },
		func(ctx context.Context, symbol string) error { return runtime.RunDecideOnce(ctx, scheduler, symbol) },
		func(ctx context.Context, symbol string) error { return runtime.RunReconcileOnce(ctx, scheduler, symbol) },
		func(ctx context.Context, symbol string) error { return runtime.RunRiskMonitorOnce(ctx, scheduler, symbol) },
		state.asyncNotifier.Render,
		state.asyncNotifier.EnqueueRendered,
		state.asyncNotifier.Deliver,
	)
	periodicJobs := jobs.BuildPeriodicJobs(buildRiverPeriodicSchedules(state.env.sys, state.runtimes))
	riverClient, err := jobs.NewClient(state.env.ctx, state.deps.persistence.pool, riverWorkers, periodicJobs, maxLLMJobTimeout(state.env.sys), state.env.logger)
	if err != nil {
		return fmt.Errorf("river client init failed: %w", err)
	}
	state.asyncNotifier.SetClient(riverClient.Inner())
	if startErr := riverClient.Start(state.env.ctx); startErr != nil {
		return fmt.Errorf("river client start failed: %w", startErr)
	}
	state.cleanup.Add(func(context.Context) {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		riverClient.Stop(stopCtx)
	})
	runtimeHandler, err := buildRuntimeHandler(state.env.sys, state.deps, scheduler, state.resolver, state.symbolConfigs)
	if err != nil {
		return fmt.Errorf("runtime api init failed: %w", err)
	}
	topMux := buildTopMux(state.viewerHandler, state.dashboardHandler, runtimeHandler)
	attachWebhookRoutes(state.env.ctx, state.env.logger, state.env.sys, state.deps, scheduler, topMux)
	addr := strings.TrimSpace(state.env.sys.Webhook.Addr)
	if addr == "" {
		return fmt.Errorf("http addr missing")
	}
	startFeishuBot(state.env.ctx, state.env.logger, state.env.sys, addr, topMux)
	if _, err := transport.StartHTTPServer(state.env.ctx, addr, topMux, state.env.logger); err != nil {
		return fmt.Errorf("start http server: %w", err)
	}
	startTelegramBot(state.env.ctx, state.env.logger, state.env.sys, addr)
	sendStartupNotify(state.env.ctx, state.env.logger, state.env.sys, state.env.index, state.runtimes, scheduler, state.deps, state.env.notifier)
	return nil
}

func waitForShutdown(state *appRuntime) error {
	<-state.env.ctx.Done()
	state.env.logger.Info("shutdown signal received")
	sendShutdownNotify(state.env.logger, state.env.notifier, state.startedAt)
	state.cleanup.Run(context.Background())
	state.env.logger.Info("shutdown flow completed")
	return nil
}

func wireAsyncNotifier(state *appRuntime) {
	state.deps.execution.notifier = state.asyncNotifier
	if state.deps.position.positioner != nil {
		state.deps.position.positioner.Notifier = state.asyncNotifier
	}
	if state.deps.reconcile.reconciler != nil {
		state.deps.reconcile.reconciler.Notifier = state.asyncNotifier
	}
	if state.deps.reconcile.recovery != nil {
		state.deps.reconcile.recovery.Notifier = state.asyncNotifier
	}
}

func resolveRunPaths(opts Options) (string, string) {
	systemPath := strings.TrimSpace(opts.SystemPath)
	if systemPath == "" {
		systemPath = "configs/system.toml"
	}
	symbolIndexPath := strings.TrimSpace(opts.SymbolIndexPath)
	if symbolIndexPath == "" {
		symbolIndexPath = "configs/symbols-index.toml"
	}
	return systemPath, symbolIndexPath
}

func buildRiverPeriodicSchedules(sys config.SystemConfig, runtimes map[string]runtime.SymbolRuntime) []jobs.PeriodicSchedule {
	if !strings.EqualFold(sys.Scheduler.Backend, "river") {
		return nil
	}
	schedules := make([]jobs.PeriodicSchedule, 0, len(runtimes))
	for symbol, rt := range runtimes {
		if rt.BarInterval <= 0 {
			continue
		}
		schedules = append(schedules, jobs.PeriodicSchedule{
			Symbol:              symbol,
			ObserveInterval:     rt.BarInterval,
			DecideInterval:      rt.BarInterval,
			ReconcileInterval:   time.Duration(sys.Webhook.FallbackReconcileSec) * time.Second,
			RiskMonitorInterval: time.Second,
		})
	}
	return schedules
}

const (
	defaultLLMClientTimeout   = 30 * time.Second
	fallbackLLMRequestTimeout = 300 * time.Second
	llmRequestMaxAttempts     = 3
	llmParseRetryBudget       = 30 * time.Second
	riverDecisionLLMPhases    = 3
	riverJobTimeoutBuffer     = 2 * time.Minute
	minRiverJobTimeout        = 10 * time.Minute
)

func maxLLMJobTimeout(sys config.SystemConfig) time.Duration {
	maxTimeout := defaultLLMClientTimeout
	for _, cfg := range sys.LLMModels {
		timeout := effectiveLLMCallTimeout(cfg)
		if timeout > maxTimeout {
			maxTimeout = timeout
		}
	}
	// A single decide job runs up to three serial LLM-backed phases (agent, provider, risk).
	// Stages within agent/provider run in parallel, so the phase budget is the slowest stage:
	// 3 network attempts plus one parse-retry window, then a small non-LLM buffer.
	timeout := (maxTimeout * llmRequestMaxAttempts * riverDecisionLLMPhases) +
		(llmParseRetryBudget * riverDecisionLLMPhases) +
		riverJobTimeoutBuffer
	if timeout < minRiverJobTimeout {
		return minRiverJobTimeout
	}
	return timeout
}

func effectiveLLMCallTimeout(cfg config.LLMModelConfig) time.Duration {
	if cfg.TimeoutSec == nil {
		return defaultLLMClientTimeout
	}
	if *cfg.TimeoutSec <= 0 {
		return fallbackLLMRequestTimeout
	}
	return time.Duration(*cfg.TimeoutSec) * time.Second
}

func sendStartupNotify(ctx context.Context, logger *zap.Logger, sys config.SystemConfig, index config.SymbolIndexConfig, runtimes map[string]runtime.SymbolRuntime, scheduler *runtime.RuntimeScheduler, deps coreDeps, notifier startupNotifier) {
	if !sys.Notification.StartupNotifyEnabled {
		return
	}
	info := buildStartupInfo(ctx, logger, index, runtimes, scheduler, deps)
	if err := notifier.SendStartup(ctx, info); err != nil {
		logger.Error("startup notify failed", zap.Error(err))
		return
	}
	logger.Info("startup notify sent")
}

func buildStartupInfo(ctx context.Context, logger *zap.Logger, index config.SymbolIndexConfig, runtimes map[string]runtime.SymbolRuntime, scheduler *runtime.RuntimeScheduler, deps coreDeps) notify.StartupInfo {
	symbols := make([]string, 0, len(index.Symbols))
	for _, entry := range index.Symbols {
		symbols = append(symbols, strings.TrimSpace(entry.Symbol))
	}
	intervals := collectIntervals(runtimes)
	barInterval := collectBarInterval(runtimes)
	info := notify.StartupInfo{
		Symbols:        symbols,
		Intervals:      intervals,
		BarInterval:    barInterval,
		ScheduleMode:   resolveStartupScheduleMode(deps.execution.scheduled),
		SymbolStatuses: collectStartupSymbolStatuses(index, runtimes, scheduler),
	}
	if deps.execution.freqtradeAcct != nil {
		balanceCtx := ctx
		if balanceCtx == nil {
			balanceCtx = context.Background()
		}
		accountCtx, cancel := context.WithTimeout(balanceCtx, 5*time.Second)
		defer cancel()
		symbolForAccount := ""
		if len(symbols) > 0 {
			symbolForAccount = symbols[0]
		}
		acct, err := deps.execution.freqtradeAcct(accountCtx, symbolForAccount)
		if err != nil {
			logger.Warn("startup balance fetch failed", zap.Error(err))
		} else {
			info.Balance = acct.Equity
			info.Currency = strings.TrimSpace(acct.Currency)
		}
	}
	return info
}

func resolveStartupScheduleMode(scheduled bool) string {
	if scheduled {
		return "定时调度"
	}
	return "手动/观察模式"
}

func collectStartupSymbolStatuses(index config.SymbolIndexConfig, runtimes map[string]runtime.SymbolRuntime, scheduler *runtime.RuntimeScheduler) []notify.StartupSymbolStatus {
	nextRunBySymbol := make(map[string]runtime.SymbolNextRun)
	if scheduler != nil {
		for _, item := range scheduler.GetScheduleStatus().NextRuns {
			nextRunBySymbol[strings.TrimSpace(item.Symbol)] = item
		}
	}
	statuses := make([]notify.StartupSymbolStatus, 0, len(index.Symbols))
	for _, entry := range index.Symbols {
		symbol := strings.TrimSpace(entry.Symbol)
		if symbol == "" {
			continue
		}
		rt, ok := runtimes[symbol]
		if !ok {
			continue
		}
		nextRun := nextRunBySymbol[symbol]
		statuses = append(statuses, notify.StartupSymbolStatus{
			Symbol:       symbol,
			Intervals:    slices.Clone(rt.Intervals),
			NextDecision: strings.TrimSpace(nextRun.NextExecution),
			Mode:         strings.TrimSpace(nextRun.Mode),
		})
	}
	return statuses
}

func sendShutdownNotify(logger *zap.Logger, notifier startupNotifier, startedAt time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info := notify.ShutdownInfo{
		Reason: "收到停止信号",
		Uptime: time.Since(startedAt),
	}
	if err := notifier.SendShutdown(ctx, info); err != nil {
		logger.Error("shutdown notify failed", zap.Error(err))
		return
	}
	logger.Info("shutdown notify sent")
}

func collectIntervals(runtimes map[string]runtime.SymbolRuntime) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, rt := range runtimes {
		for _, iv := range rt.Intervals {
			iv = strings.TrimSpace(iv)
			if iv == "" {
				continue
			}
			if _, ok := seen[iv]; ok {
				continue
			}
			seen[iv] = struct{}{}
			result = append(result, iv)
		}
	}
	return result
}

func collectBarInterval(runtimes map[string]runtime.SymbolRuntime) string {
	for _, rt := range runtimes {
		if rt.BarInterval > 0 {
			return rt.BarInterval.String()
		}
	}
	return ""
}
