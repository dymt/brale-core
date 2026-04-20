package risk

import "strings"

func BreakevenStop(side string, entry float64, feePct float64) float64 {
	if entry <= 0 {
		return 0
	}
	if feePct < 0 {
		feePct = 0
	}
	fee := entry * feePct
	if strings.EqualFold(strings.TrimSpace(side), "short") {
		return entry - fee
	}
	return entry + fee
}
