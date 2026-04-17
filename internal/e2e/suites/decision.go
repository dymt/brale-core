package suites

import (
	"fmt"
	"time"

	"brale-core/internal/e2e"
)

func init() {
	e2e.Register("decision", func() e2e.Suite { return &DecisionSuite{} })
}

// DecisionSuite waits for a new LLM decision round to complete.
type DecisionSuite struct{}

func (s *DecisionSuite) Name() string { return "decision" }

func (s *DecisionSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	symbol := ctx.Config.Symbol
	timeout := 25 * time.Minute

	// Record baseline snapshot_id
	baseline, err := ctx.Client.FetchDecisionLatest(ctx.Ctx, symbol)
	if err != nil {
		result.Status = e2e.StatusFail
		result.Error = fmt.Sprintf("fetch baseline decision: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	baselineID := baseline.SnapshotID
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name:    "baseline snapshot_id",
		Passed:  true,
		Message: fmt.Sprintf("id=%d", baselineID),
	})

	// Poll for a new decision
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	deadline := time.After(timeout)

	var newDecision bool
	var latestID uint
	for {
		select {
		case <-ctx.Ctx.Done():
			result.Status = e2e.StatusFail
			result.Error = "context cancelled"
			result.Duration = time.Since(start)
			return result
		case <-deadline:
			result.Status = e2e.StatusFail
			result.Error = fmt.Sprintf("timeout waiting for new decision (baseline=%d, timeout=%s)", baselineID, timeout)
			result.Duration = time.Since(start)
			return result
		case <-ticker.C:
			latest, err := ctx.Client.FetchDecisionLatest(ctx.Ctx, symbol)
			if err != nil {
				continue
			}
			if latest.SnapshotID > baselineID {
				latestID = latest.SnapshotID
				newDecision = true

				// Validate decision structure
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    "new snapshot_id",
					Passed:  true,
					Message: fmt.Sprintf("id=%d (was %d)", latestID, baselineID),
				})
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    "decision has report",
					Passed:  latest.Report != "",
					Message: fmt.Sprintf("len=%d", len(latest.Report)),
				})
				result.Checks = append(result.Checks, e2e.CheckResult{
					Name:    "decision has summary",
					Passed:  latest.Summary != "",
					Message: fmt.Sprintf("len=%d", len(latest.Summary)),
				})
			}
			if newDecision {
				goto done
			}
		}
	}

done:
	// Verify LLM rounds
	rounds, err := ctx.LLMRounds(symbol, 5)
	if err == nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "llm/rounds accessible",
			Passed:  true,
			Message: "ok",
		})
		result.Evidence = append(result.Evidence, e2e.Evidence{
			Label: "llm_rounds",
			Data:  rounds,
		})
	} else {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "llm/rounds accessible",
			Passed:  false,
			Message: err.Error(),
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
		result.Error = "some decision checks failed"
	}
	result.Duration = time.Since(start)
	return result
}
