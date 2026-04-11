package features

import (
	"math"
)

// --- Snapshot types ---

type bbSnapshot struct {
	Upper    float64 `json:"upper"`
	Middle   float64 `json:"middle"`
	Lower    float64 `json:"lower"`
	Width    float64 `json:"width"`
	PercentB float64 `json:"percent_b"`
}

type chopSnapshot struct {
	Value float64 `json:"value"`
}

type stochRSISnapshot struct {
	Value float64 `json:"value"`
}

type aroonSnapshot struct {
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
}

type tdSequentialSnapshot struct {
	BuySetup  int `json:"buy_setup"`
	SellSetup int `json:"sell_setup"`
}

// --- Computation functions ---

// computeBollingerBands calculates Bollinger Bands for the given closes.
// Returns upper, middle (SMA), lower series of len(closes).
// NaN values are emitted for indices where the window is incomplete.
func computeBollingerBands(closes []float64, period int, multiplier float64) (upper, middle, lower []float64) {
	n := len(closes)
	upper = make([]float64, n)
	middle = make([]float64, n)
	lower = make([]float64, n)
	for i := range closes {
		upper[i] = math.NaN()
		middle[i] = math.NaN()
		lower[i] = math.NaN()
	}
	if period <= 0 || n < period {
		return
	}
	for i := period - 1; i < n; i++ {
		start := i + 1 - period
		sum := 0.0
		for j := start; j <= i; j++ {
			sum += closes[j]
		}
		sma := sum / float64(period)
		variance := 0.0
		for j := start; j <= i; j++ {
			d := closes[j] - sma
			variance += d * d
		}
		stddev := math.Sqrt(variance / float64(period))
		middle[i] = sma
		upper[i] = sma + multiplier*stddev
		lower[i] = sma - multiplier*stddev
	}
	return
}

// buildBBSnapshot creates a Bollinger Bands snapshot from the last values of the computed series.
func buildBBSnapshot(upper, middle, lower []float64, price float64) *bbSnapshot {
	if len(upper) == 0 || len(middle) == 0 || len(lower) == 0 {
		return nil
	}
	u := upper[len(upper)-1]
	m := middle[len(middle)-1]
	l := lower[len(lower)-1]
	if math.IsNaN(u) || math.IsNaN(m) || math.IsNaN(l) {
		return nil
	}
	width := 0.0
	if m > 0 {
		width = (u - l) / m * 100
	}
	percentB := 0.0
	band := u - l
	if band > 1e-12 {
		percentB = (price - l) / band
	}
	return &bbSnapshot{
		Upper:    roundFloat(u, 4),
		Middle:   roundFloat(m, 4),
		Lower:    roundFloat(l, 4),
		Width:    roundFloat(width, 4),
		PercentB: roundFloat(percentB, 4),
	}
}

// computeCHOP calculates the Choppiness Index series.
// CHOP = 100 * log10(SUM(ATR(1), n) / (Highest(High, n) - Lowest(Low, n))) / log10(n)
func computeCHOP(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	if period <= 1 || n < period+1 {
		return out
	}
	// Compute True Range series (ATR with period=1 is just TR)
	tr := make([]float64, n)
	for i := range closes {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
		} else {
			hl := highs[i] - lows[i]
			hc := math.Abs(highs[i] - closes[i-1])
			lc := math.Abs(lows[i] - closes[i-1])
			tr[i] = math.Max(hl, math.Max(hc, lc))
		}
	}
	log10Period := math.Log10(float64(period))
	if log10Period <= 1e-12 {
		return out
	}
	for i := period; i < n; i++ {
		sumTR := 0.0
		highest := lows[i-period+1]
		lowest := highs[i-period+1]
		for j := i - period + 1; j <= i; j++ {
			sumTR += tr[j]
			if highs[j] > highest {
				highest = highs[j]
			}
			if lows[j] < lowest {
				lowest = lows[j]
			}
		}
		rng := highest - lowest
		if rng <= 1e-12 {
			continue
		}
		ratio := sumTR / rng
		if ratio <= 0 {
			continue
		}
		out[i] = 100 * math.Log10(ratio) / log10Period
	}
	return out
}

// buildCHOPSnapshot creates a CHOP snapshot from the last value.
func buildCHOPSnapshot(series []float64) *chopSnapshot {
	if len(series) == 0 {
		return nil
	}
	v := series[len(series)-1]
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &chopSnapshot{Value: roundFloat(v, 4)}
}

// computeStochRSI calculates Stochastic RSI: (RSI - MinRSI) / (MaxRSI - MinRSI).
// rsiSeries should be the output of talib.Rsi or equivalent.
func computeStochRSI(rsiSeries []float64, period int) []float64 {
	n := len(rsiSeries)
	out := make([]float64, n)
	for i := range out {
		out[i] = math.NaN()
	}
	if period <= 0 || n < period {
		return out
	}
	for i := period - 1; i < n; i++ {
		start := i + 1 - period
		lo := rsiSeries[start]
		hi := rsiSeries[start]
		allValid := true
		for j := start; j <= i; j++ {
			v := rsiSeries[j]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				allValid = false
				break
			}
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
		if !allValid {
			continue
		}
		rng := hi - lo
		if rng <= 1e-12 {
			out[i] = 0.5
			continue
		}
		current := rsiSeries[i]
		out[i] = (current - lo) / rng
	}
	return out
}

// buildStochRSISnapshot creates a snapshot from the last StochRSI value.
func buildStochRSISnapshot(series []float64) *stochRSISnapshot {
	if len(series) == 0 {
		return nil
	}
	v := series[len(series)-1]
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &stochRSISnapshot{Value: roundFloat(v, 4)}
}

// computeAroon calculates the Aroon Up and Down indicator series.
// AroonUp = (period - bars_since_highest_high) / period * 100
// AroonDown = (period - bars_since_lowest_low) / period * 100
func computeAroon(highs, lows []float64, period int) (aroonUp, aroonDown []float64) {
	n := len(highs)
	aroonUp = make([]float64, n)
	aroonDown = make([]float64, n)
	for i := range aroonUp {
		aroonUp[i] = math.NaN()
		aroonDown[i] = math.NaN()
	}
	if period <= 0 || n < period+1 {
		return
	}
	for i := period; i < n; i++ {
		start := i - period
		highIdx := start
		lowIdx := start
		for j := start + 1; j <= i; j++ {
			if highs[j] >= highs[highIdx] {
				highIdx = j
			}
			if lows[j] <= lows[lowIdx] {
				lowIdx = j
			}
		}
		barsSinceHigh := float64(i - highIdx)
		barsSinceLow := float64(i - lowIdx)
		p := float64(period)
		aroonUp[i] = (p - barsSinceHigh) / p * 100
		aroonDown[i] = (p - barsSinceLow) / p * 100
	}
	return
}

// buildAroonSnapshot creates an Aroon snapshot from the last values.
func buildAroonSnapshot(aroonUp, aroonDown []float64) *aroonSnapshot {
	if len(aroonUp) == 0 || len(aroonDown) == 0 {
		return nil
	}
	u := aroonUp[len(aroonUp)-1]
	d := aroonDown[len(aroonDown)-1]
	if math.IsNaN(u) || math.IsNaN(d) {
		return nil
	}
	return &aroonSnapshot{
		Up:   roundFloat(u, 4),
		Down: roundFloat(d, 4),
	}
}

// computeTDSequential returns buySetup and sellSetup counts for each bar.
// Buy setup: close < close[i-4], count consecutive occurrences.
// Sell setup: close > close[i-4], count consecutive occurrences.
func computeTDSequential(closes []float64) (buySetup, sellSetup []int) {
	n := len(closes)
	buySetup = make([]int, n)
	sellSetup = make([]int, n)
	if n <= 4 {
		return
	}
	for i := 4; i < n; i++ {
		if closes[i] < closes[i-4] {
			buySetup[i] = buySetup[i-1] + 1
			sellSetup[i] = 0
		} else if closes[i] > closes[i-4] {
			sellSetup[i] = sellSetup[i-1] + 1
			buySetup[i] = 0
		} else {
			buySetup[i] = 0
			sellSetup[i] = 0
		}
	}
	return
}

// buildTDSequentialSnapshot creates a snapshot from the last TD Sequential values.
func buildTDSequentialSnapshot(buySetup, sellSetup []int) *tdSequentialSnapshot {
	if len(buySetup) == 0 || len(sellSetup) == 0 {
		return nil
	}
	b := buySetup[len(buySetup)-1]
	s := sellSetup[len(sellSetup)-1]
	if b == 0 && s == 0 {
		return nil
	}
	return &tdSequentialSnapshot{BuySetup: b, SellSetup: s}
}
