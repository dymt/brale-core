package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("open-fast", func() e2e.Suite { return &OpenFastSuite{} })
}

// OpenFastSuite waits until the system naturally opens a position.
type OpenFastSuite struct{}

func (s *OpenFastSuite) Name() string { return "open-fast" }

func (s *OpenFastSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol
	timeout := 60 * time.Minute

	// Check no existing position for this symbol
	pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch position status: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	for _, p := range pos.Positions {
		if p.Symbol == symbol {
			result.Status = e2e.StatusPass
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "position already exists",
				Passed:  true,
				Message: fmt.Sprintf("side=%s entry=%.4f", p.Side, p.EntryPrice),
			})
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "no existing position",
		Passed:  true,
		Message: fmt.Sprintf("positions=%d", len(pos.Positions)),
	})

	// Poll for open position
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Ctx.Done():
			result.Status = e2e.StatusFail
			result.Error = "context cancelled"
			result.Duration = time.Since(start)
			return result
		case <-deadline:
			result.Status = e2e.StatusFail
			result.Error = fmt.Sprintf("timeout waiting for open position (timeout=%s)", timeout)
			result.Duration = time.Since(start)
			return result
		case <-ticker.C:
			pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
			if err != nil {
				continue
			}
			for _, p := range pos.Positions {
				if p.Symbol == symbol {
					result.Checks = append(result.Checks, e2e.CheckResult{
						Name:    "position opened",
						Passed:  true,
						Message: fmt.Sprintf("side=%s entry=%.4f amount=%.6f", p.Side, p.EntryPrice, p.Amount),
					})
					result.Checks = append(result.Checks, e2e.CheckResult{
						Name:    "entry_price > 0",
						Passed:  p.EntryPrice > 0,
						Message: fmt.Sprintf("%.4f", p.EntryPrice),
					})
					result.Checks = append(result.Checks, e2e.CheckResult{
						Name:    "stop_loss > 0",
						Passed:  p.StopLoss > 0,
						Message: fmt.Sprintf("%.4f", p.StopLoss),
					})
					result.Checks = append(result.Checks, e2e.CheckResult{
						Name:    "take_profits non-empty",
						Passed:  len(p.TakeProfits) > 0,
						Message: fmt.Sprintf("count=%d", len(p.TakeProfits)),
					})

					result.Status = e2e.StatusPass
					result.Duration = time.Since(start)
					return result
				}
			}
		}
	}
}
