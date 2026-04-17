package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("lifecycle-natural", func() e2e.Suite { return &LifecycleNaturalSuite{} })
}

// LifecycleNaturalSuite waits for natural position open, then waits for TP/SL exit.
// This suite can take a very long time (hours) as it waits for real market movement.
type LifecycleNaturalSuite struct{}

func (s *LifecycleNaturalSuite) Name() string { return "lifecycle-natural" }

func (s *LifecycleNaturalSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol
	progress := newDecisionProgressTracker(ctx, symbol)

	// Step 1: Wait for position open
	ctx.Log("waiting for position to open (natural entry)...")
	openDeadline := time.Now().Add(ctx.Config.Timeout)
	var posOpen bool
	var entrySide string
	for time.Now().Before(openDeadline) {
		pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
		if err == nil {
			for _, p := range pos.Positions {
				if p.Symbol == symbol {
					posOpen = true
					entrySide = p.Side
					result.Checks = append(result.Checks, e2e.CheckResult{
						Name:    "position opened",
						Passed:  true,
						Message: fmt.Sprintf("side=%s entry=%.4f", p.Side, p.EntryPrice),
					})
					break
				}
			}
		}
		if posOpen {
			break
		}
		progress.Poll(ctx, "waiting for open")
		time.Sleep(15 * time.Second)
	}

	if !posOpen {
		result.Status = e2e.StatusFail
		result.Error = "timed out waiting for position open"
		result.Duration = time.Since(start)
		return result
	}
	ctx.Log("position opened: side=%s", entrySide)

	// Step 2: Run reconcile checks while position is open
	ctx.Log("running inline reconcile checks...")
	reconcileSuite := &ReconcileSuite{}
	reconResult := reconcileSuite.Run(ctx)
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "inline:reconcile",
		Passed:  reconResult.Status == e2e.StatusPass,
		Message: fmt.Sprintf("status=%s checks=%d", reconResult.Status, len(reconResult.Checks)),
	})

	// Step 3: Wait for natural exit (TP or SL hit)
	ctx.Log("waiting for natural exit (TP/SL)...")
	exitDeadline := time.Now().Add(ctx.Config.Timeout)
	var exited bool
	for time.Now().Before(exitDeadline) {
		pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
		if err == nil {
			found := false
			for _, p := range pos.Positions {
				if p.Symbol == symbol {
					found = true
					break
				}
			}
			if !found {
				exited = true
				break
			}
		}
		progress.Poll(ctx, "waiting for exit")
		time.Sleep(15 * time.Second)
	}
	if exited {
		ctx.Log("natural exit observed")
	}

	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "natural exit observed",
		Passed:  exited,
		Message: fmt.Sprintf("exited=%v side_was=%s", exited, entrySide),
	})

	if !exited {
		result.Status = e2e.StatusFail
		result.Error = "timed out waiting for natural exit"
		result.Duration = time.Since(start)
		return result
	}

	// Step 4: Verify trade history
	ctx.Log("verifying post-close state...")
	trades, err := ctx.Client.FetchTradeHistory(ctx.Ctx)
	if err == nil {
		var tradeFound bool
		for _, t := range trades.Trades {
			if t.Symbol == symbol {
				tradeFound = true
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    "trade recorded",
					Passed:  true,
					Message: fmt.Sprintf("side=%s profit=%.4f", t.Side, t.Profit),
				})
				break
			}
		}
		if !tradeFound {
			result.Checks = append(result.Checks, e2e.CheckResult{
				Name:    "trade recorded",
				Passed:  false,
				Message: "no trade found in history",
			})
		}
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
		result.Error = "lifecycle-natural checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
