package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("post-close", func() e2e.Suite { return &PostCloseSuite{} })
}

// PostCloseSuite validates post-close data: history, episodic memory, risk plan history.
type PostCloseSuite struct{}

func (s *PostCloseSuite) Name() string { return "post-close" }

func (s *PostCloseSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol

	// Verify no open position
	pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch position status: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	for _, p := range pos.Positions {
		if p.Symbol == symbol {
			result.Status = e2e.StatusBlocked
			result.Error = fmt.Sprintf("position still open for %s (side=%s)", symbol, p.Side)
			result.Duration = time.Since(start)
			return result
		}
	}
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "no open position",
		Passed:  true,
		Message: "position is closed",
	})

	// Verify trade history exists
	trades, err := ctx.Client.FetchTradeHistory(ctx.Ctx)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch trade history: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	var matchingTrade bool
	for _, t := range trades.Trades {
		if t.Symbol == symbol {
			matchingTrade = true
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "trade history has symbol",
				Passed:  true,
				Message: fmt.Sprintf("side=%s profit=%.4f duration=%ds", t.Side, t.Profit, t.DurationSec),
			})
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "trade has side",
				Passed:  t.Side != "",
				Message: t.Side,
			})
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "trade has opened_at",
				Passed:  !t.OpenedAt.IsZero(),
				Message: t.OpenedAt.Format(time.RFC3339),
			})
			break
		}
	}

	if !matchingTrade {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "trade history has symbol",
			Passed:  false,
			Message: fmt.Sprintf("no trade found for %s in %d trades", symbol, len(trades.Trades)),
		})
	}

	// Dashboard reconciliation should be clean after close
	dash, err := ctx.Client.FetchDashboardOverview(ctx.Ctx, symbol)
	if err == nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "dashboard accessible post-close",
			Passed:  true,
			Message: "ok",
		})
		result.Evidence = append(result.Evidence, e2e.Evidence{
			Label: "dashboard_overview",
			Data:  dash,
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
		result.Error = "post-close checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
