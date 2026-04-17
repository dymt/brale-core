package suites

import (
	"fmt"
	"math"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("pricing", func() e2e.Suite { return &PricingSuite{} })
}

// PricingSuite validates price and PnL calculations for an open position.
type PricingSuite struct{}

func (s *PricingSuite) Name() string { return "pricing" }

func (s *PricingSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol

	// Fetch position status
	pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch position status: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	var found bool
	for _, p := range pos.Positions {
		if p.Symbol != symbol {
			continue
		}
		found = true

		// Basic sanity checks
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "entry_price > 0",
			Passed:  p.EntryPrice > 0,
			Message: fmt.Sprintf("%.4f", p.EntryPrice),
		})
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "current_price > 0",
			Passed:  p.CurrentPrice > 0,
			Message: fmt.Sprintf("%.4f", p.CurrentPrice),
		})

		// Side-specific TP/SL ordering
		if p.Side == "long" {
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "long: stop_loss < entry_price",
				Passed:  p.StopLoss < p.EntryPrice,
				Message: fmt.Sprintf("sl=%.4f entry=%.4f", p.StopLoss, p.EntryPrice),
			})
			for i, tp := range p.TakeProfits {
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    fmt.Sprintf("long: tp[%d] > entry_price", i),
					Passed:  tp > p.EntryPrice,
					Message: fmt.Sprintf("tp=%.4f entry=%.4f", tp, p.EntryPrice),
				})
			}
		} else if p.Side == "short" {
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "short: stop_loss > entry_price",
				Passed:  p.StopLoss > p.EntryPrice,
				Message: fmt.Sprintf("sl=%.4f entry=%.4f", p.StopLoss, p.EntryPrice),
			})
			for i, tp := range p.TakeProfits {
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    fmt.Sprintf("short: tp[%d] < entry_price", i),
					Passed:  tp < p.EntryPrice,
					Message: fmt.Sprintf("tp=%.4f entry=%.4f", tp, p.EntryPrice),
				})
			}
		}

		// Profit self-consistency: total ≈ realized + unrealized
		expectedTotal := p.ProfitRealized + p.ProfitUnrealized
		tolerance := math.Max(0.5, math.Abs(p.ProfitTotal)*0.01)
		profitDrift := math.Abs(p.ProfitTotal - expectedTotal)
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "profit_total ≈ realized + unrealized",
			Passed:  profitDrift <= tolerance,
			Message: fmt.Sprintf("total=%.4f realized=%.4f unrealized=%.4f drift=%.4f tol=%.4f", p.ProfitTotal, p.ProfitRealized, p.ProfitUnrealized, profitDrift, tolerance),
		})

		// Direction-consistent unrealized PnL
		if p.Side == "long" && p.CurrentPrice > p.EntryPrice {
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "long: positive unrealized when price > entry",
				Passed:  p.ProfitUnrealized >= -tolerance,
				Message: fmt.Sprintf("unrealized=%.4f", p.ProfitUnrealized),
			})
		}
		if p.Side == "short" && p.CurrentPrice < p.EntryPrice {
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "short: positive unrealized when price < entry",
				Passed:  p.ProfitUnrealized >= -tolerance,
				Message: fmt.Sprintf("unrealized=%.4f", p.ProfitUnrealized),
			})
		}

		break
	}

	if !found {
		if ctx.Config.Bootstrap == "strict" {
			result.Status = e2e.StatusBlocked
			result.Error = "no open position for " + symbol
		} else {
			result.Status = e2e.StatusSkip
			result.Error = "no open position (use --bootstrap auto or run open-fast first)"
		}
		result.Duration = time.Since(start)
		return result
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
		result.Error = "pricing checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
