package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("reconcile", func() e2e.Suite { return &ReconcileSuite{} })
}

// ReconcileSuite checks consistency between Brale runtime, dashboard, and database views.
type ReconcileSuite struct{}

func (s *ReconcileSuite) Name() string { return "reconcile" }

func (s *ReconcileSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol

	// Layer 1: Brale runtime
	pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch position status: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "position/status accessible",
		Passed:  true,
		Message: fmt.Sprintf("positions=%d", len(pos.Positions)),
	})

	// Layer 2: Dashboard overview
	dash, err := ctx.Client.FetchDashboardOverview(ctx.Ctx, symbol)
	if err != nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard/overview accessible",
			Passed:  false,
			Message: err.Error(),
		})
	} else {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard/overview accessible",
			Passed:  true,
			Message: "ok",
		})

		// Check reconciliation status per symbol
		for _, sym := range dash.Symbols {
			if sym.Symbol == symbol {
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    "reconciliation.status != error",
					Passed:  sym.Reconciliation.Status != "error",
					Message: fmt.Sprintf("status=%s drift_abs=%.6f", sym.Reconciliation.Status, sym.Reconciliation.DriftAbs),
				})
				break
			}
		}
	}

	// Layer 3: Account summary
	acct, err := ctx.Client.FetchDashboardAccountSummary(ctx.Ctx)
	if err != nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard/account_summary accessible",
			Passed:  false,
			Message: err.Error(),
		})
	} else {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard/account_summary accessible",
			Passed:  true,
			Message: "ok",
		})
		result.Evidence = append(result.Evidence, e2e.Evidence{
			Label: "account_summary",
			Data:  acct,
		})
	}

	// Layer 4: Trade history
	trades, err := ctx.Client.FetchTradeHistory(ctx.Ctx)
	if err != nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "position/history accessible",
			Passed:  false,
			Message: err.Error(),
		})
	} else {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "position/history accessible",
			Passed:  true,
			Message: fmt.Sprintf("trades=%d", len(trades.Trades)),
		})
	}

	// Cross-check: if position exists in Brale, dashboard should reflect it
	var braleHasPosition bool
	for _, p := range pos.Positions {
		if p.Symbol == symbol {
			braleHasPosition = true
			break
		}
	}

	if braleHasPosition {
		var dashHasPosition bool
		for _, sym := range dash.Symbols {
			if sym.Symbol == symbol && sym.Position.Side != "" {
				dashHasPosition = true
				break
			}
		}
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard reflects open position",
			Passed:  dashHasPosition,
			Message: fmt.Sprintf("brale=true dashboard=%v", dashHasPosition),
		})
	}

	allPassed := true
	for _, c := range result.Checks {
		if !c.Passed {
			allPassed = false
			break
		}
	}
	if allPassed {
		result.Status = e2e.StatusPass
	} else {
		result.Status = e2e.StatusFail
		result.Error = "reconcile checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
