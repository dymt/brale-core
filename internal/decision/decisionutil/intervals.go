package decisionutil

import (
	"time"

	"brale-core/internal/interval"
)

// SelectDecisionInterval returns the shortest valid interval. If none of the
// provided values parse, it falls back to the first raw entry.
func SelectDecisionInterval(intervals []string) string {
	shortest := ""
	var shortestDur time.Duration
	for _, candidate := range intervals {
		dur, err := interval.ParseInterval(candidate)
		if err != nil {
			continue
		}
		if shortest == "" || dur < shortestDur {
			shortest = candidate
			shortestDur = dur
		}
	}
	if shortest != "" {
		return shortest
	}
	if len(intervals) == 0 {
		return ""
	}
	return intervals[0]
}
