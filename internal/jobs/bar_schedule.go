package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

const barCloseDelay = 10 * time.Second

type alignedBarSchedule struct {
	interval time.Duration
}

// AlignedBarCloseSchedule returns a River periodic schedule aligned to the next
// bar close boundary for the given interval, plus a small delay so data sources
// have time to finalize the just-closed candle.
func AlignedBarCloseSchedule(interval time.Duration) river.PeriodicSchedule {
	return &alignedBarSchedule{interval: interval}
}

func (s *alignedBarSchedule) Next(current time.Time) time.Time {
	if s == nil || s.interval <= 0 {
		return current
	}
	next := current.Truncate(s.interval).Add(s.interval)
	return next.Add(barCloseDelay)
}
