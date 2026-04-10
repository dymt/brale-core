package features

import "math"

func computeSuperTrendSeries(highs, lows, closes []float64, period int, multiplier float64) []float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) || len(closes) < 2 || period <= 0 || multiplier <= 0 {
		return nil
	}

	tr := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		high := highs[i]
		low := lows[i]
		prevClose := closes[i-1]
		tr = append(tr, math.Max(high-low, math.Max(high-prevClose, prevClose-low)))
	}

	atr := hmaSeries(tr, period)
	if len(atr) == 0 {
		return nil
	}

	atrIdle := superTrendATRIdlePeriod(period)
	medians := make([]float64, 0, len(closes)-atrIdle)
	closings := make([]float64, 0, len(closes)-atrIdle)
	for i := atrIdle; i < len(closes); i++ {
		medians = append(medians, (highs[i]+lows[i])/2)
		closings = append(closings, closes[i])
	}
	if len(medians) != len(atr) || len(closings) != len(atr) {
		return nil
	}

	first := true
	upTrend := false
	previousClosing := 0.0
	finalUpperBand := 0.0
	finalLowerBand := 0.0
	superTrend := make([]float64, len(atr))

	for i := range atr {
		median := medians[i]
		atrMultiple := atr[i] * multiplier
		closing := closings[i]
		basicUpperBand := median + atrMultiple
		basicLowerBand := median - atrMultiple

		if first {
			first = false
			finalUpperBand = basicUpperBand
			finalLowerBand = basicLowerBand
			superTrend[i] = finalLowerBand
		} else {
			if basicUpperBand < finalUpperBand || previousClosing > finalUpperBand {
				finalUpperBand = basicUpperBand
			}
			if basicLowerBand > finalLowerBand || previousClosing < finalLowerBand {
				finalLowerBand = basicLowerBand
			}
			if upTrend {
				if closing <= finalUpperBand {
					superTrend[i] = finalUpperBand
				} else {
					superTrend[i] = finalLowerBand
					upTrend = false
				}
			} else {
				if closing >= finalLowerBand {
					superTrend[i] = finalLowerBand
				} else {
					superTrend[i] = finalUpperBand
					upTrend = true
				}
			}
		}

		previousClosing = closing
	}

	return superTrend
}

func hmaSeries(values []float64, period int) []float64 {
	if len(values) == 0 || period <= 0 {
		return nil
	}
	halfPeriod := int(math.Round(float64(period) / 2))
	sqrtPeriod := int(math.Round(math.Sqrt(float64(period))))
	if halfPeriod <= 0 || sqrtPeriod <= 0 {
		return nil
	}

	wma1 := wmaSeries(values, halfPeriod)
	wma2 := wmaSeries(values, period)
	if len(wma2) == 0 {
		return nil
	}
	skip := period - halfPeriod
	if skip < 0 || skip > len(wma1) {
		return nil
	}
	wma1 = wma1[skip:]
	if len(wma1) != len(wma2) {
		return nil
	}

	diff := make([]float64, len(wma2))
	for i := range wma2 {
		diff[i] = 2*wma1[i] - wma2[i]
	}
	return wmaSeries(diff, sqrtPeriod)
}

func wmaSeries(values []float64, period int) []float64 {
	if len(values) < period || period <= 0 {
		return nil
	}
	divisor := float64(period*(period+1)) / 2
	out := make([]float64, 0, len(values)-period+1)
	for end := period - 1; end < len(values); end++ {
		sum := 0.0
		weight := float64(period)
		for idx := end - period + 1; idx <= end; idx++ {
			sum += values[idx] * weight
			weight--
		}
		out = append(out, sum/divisor)
	}
	return out
}

func superTrendATRIdlePeriod(period int) int {
	if period <= 0 {
		return 0
	}
	sqrtPeriod := int(math.Round(math.Sqrt(float64(period))))
	return period + sqrtPeriod - 1
}
