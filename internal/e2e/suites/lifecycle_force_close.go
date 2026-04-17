package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("lifecycle-force-close", func() e2e.Suite { return &LifecycleForceCloseSuite{} })
}

// LifecycleForceCloseSuite runs the full lifecycle: wait for open → verify → force close → verify close.
type LifecycleForceCloseSuite struct{}

func (s *LifecycleForceCloseSuite) Name() string { return "lifecycle-force-close" }

func (s *LifecycleForceCloseSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol

	// Step 1: Wait for position to open (reuse open-fast logic)
	ctx.Log("waiting for position to open...")
	deadline := time.Now().Add(ctx.Config.Timeout)
	var posOpen bool
	for time.Now().Before(deadline) {
		pos, err := ctx.Client.FetchPositionStatus(ctx.Ctx)
		if err == nil {
			for _, p := range pos.Positions {
				if p.Symbol == symbol {
					posOpen = true
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
		time.Sleep(10 * time.Second)
	}

	if !posOpen {
		result.Status = e2e.StatusFail
		result.Error = "timed out waiting for position open"
		result.Duration = time.Since(start)
		return result
	}

	// Step 2: Run pricing checks
	ctx.Log("running inline pricing checks...")
	pricingSuite := &PricingSuite{}
	pricingResult := pricingSuite.Run(ctx)
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "inline:pricing",
		Passed:  pricingResult.Status == e2e.StatusPass,
		Message: fmt.Sprintf("status=%s checks=%d", pricingResult.Status, len(pricingResult.Checks)),
	})

	// Step 3: Force close via Freqtrade forceexit
	ctx.Log("sending force close via freqtrade...")
	err := ctx.ForceClosePosition(symbol)
	if err != nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "force_close API call",
			Passed:  false,
			Message: err.Error(),
		})
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("force close failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "force_close API call",
		Passed:  true,
		Message: "accepted",
	})

	// Step 4: Wait for position to close
	ctx.Log("waiting for position to close...")
	closeDeadline := time.Now().Add(2 * time.Minute)
	var closed bool
	for time.Now().Before(closeDeadline) {
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
				closed = true
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "position closed",
		Passed:  closed,
		Message: fmt.Sprintf("closed=%v", closed),
	})

	if !closed {
		result.Status = e2e.StatusFail
		result.Error = "position did not close after force_close"
		result.Duration = time.Since(start)
		return result
	}

	// Step 5: Run post-close checks
	ctx.Log("running inline post-close checks...")
	postCloseSuite := &PostCloseSuite{}
	postCloseResult := postCloseSuite.Run(ctx)
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "inline:post-close",
		Passed:  postCloseResult.Status == e2e.StatusPass,
		Message: fmt.Sprintf("status=%s checks=%d", postCloseResult.Status, len(postCloseResult.Checks)),
	})

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
		result.Error = "lifecycle-force-close checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
