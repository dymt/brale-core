package features

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"brale-core/internal/config"
)

func TestBuildIndicatorCompressedInputIncludesSTC(t *testing.T) {
	required := config.STCRequiredBars(23, 50)
	candles := trendTestCandles(required + 20)

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", candles, DefaultIndicatorCompressOptions())
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.STC == nil {
		t.Fatalf("STC = nil")
	}
	if got.Data.STC.Current < 0 || got.Data.STC.Current > 100 {
		t.Fatalf("STC.Current=%v want [0,100]", got.Data.STC.Current)
	}
	if got.Data.STC.State != "rising" && got.Data.STC.State != "falling" && got.Data.STC.State != "flat" {
		t.Fatalf("STC.State=%q", got.Data.STC.State)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(raw), `"macd"`) {
		t.Fatalf("payload still contains macd: %s", raw)
	}
}

func TestBuildIndicatorCompressedInputIncludesEMAFastAtThreshold(t *testing.T) {
	required := config.EMARequiredBars(21)
	opts := DefaultIndicatorCompressOptions()
	opts.SkipRSI = true
	opts.SkipSTC = true

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.EMAFast == nil {
		t.Fatalf("EMAFast = nil at threshold=%d", required)
	}

	got, err = BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required-1), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.EMAFast != nil {
		t.Fatalf("EMAFast=%+v want nil below threshold", got.Data.EMAFast)
	}
}

func TestBuildIndicatorCompressedInputIncludesRSIAtThreshold(t *testing.T) {
	required := config.RSIRequiredBars(14)
	opts := DefaultIndicatorCompressOptions()
	opts.SkipEMA = true
	opts.SkipSTC = true

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.RSI == nil {
		t.Fatalf("RSI = nil at threshold=%d", required)
	}

	got, err = BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required-1), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.RSI != nil {
		t.Fatalf("RSI=%+v want nil below threshold", got.Data.RSI)
	}
}

func TestBuildIndicatorCompressedInputIncludesATRAtThreshold(t *testing.T) {
	required := config.ATRRequiredBars(14)
	opts := DefaultIndicatorCompressOptions()
	opts.SkipEMA = true
	opts.SkipRSI = true
	opts.SkipSTC = true

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.ATR == nil {
		t.Fatalf("ATR = nil at threshold=%d", required)
	}

	got, err = BuildIndicatorCompressedInput("BTCUSDT", "1h", trendTestCandles(required-1), opts)
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.ATR != nil {
		t.Fatalf("ATR=%+v want nil below threshold", got.Data.ATR)
	}
}

func TestBuildIndicatorCompressedInputOmitsSTCWhenBarsInsufficient(t *testing.T) {
	required := config.STCRequiredBars(23, 50)
	candles := trendTestCandles(required - 1)

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", candles, DefaultIndicatorCompressOptions())
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.STC != nil {
		t.Fatalf("STC=%+v want nil", got.Data.STC)
	}
}

func TestBuildIndicatorCompressedInputIncludesSTCAtThreshold(t *testing.T) {
	required := config.STCRequiredBars(23, 50)
	candles := trendTestCandles(required)

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", candles, DefaultIndicatorCompressOptions())
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.STC == nil {
		t.Fatalf("STC = nil at threshold=%d", required)
	}
}

func TestSTCRequiredBarsChangeWithParams(t *testing.T) {
	base := config.STCRequiredBars(23, 50)
	alt := config.STCRequiredBars(30, 60)
	if alt <= base {
		t.Fatalf("alt=%d want > base=%d", alt, base)
	}
}

func TestBuildIndicatorCompressedInputSTCNoNaNOnSideways(t *testing.T) {
	required := config.STCRequiredBars(23, 50)
	candles := sidewaysTrendTestCandles(required + 20)

	got, err := BuildIndicatorCompressedInput("BTCUSDT", "1h", candles, DefaultIndicatorCompressOptions())
	if err != nil {
		t.Fatalf("BuildIndicatorCompressedInput() error = %v", err)
	}
	if got.Data.STC == nil {
		t.Fatalf("STC = nil")
	}
	if math.IsNaN(got.Data.STC.Current) || math.IsInf(got.Data.STC.Current, 0) {
		t.Fatalf("STC.Current=%v", got.Data.STC.Current)
	}
	for _, v := range got.Data.STC.LastN {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("STC.LastN contains invalid value: %v", got.Data.STC.LastN)
		}
	}
}

func TestComputeSTCSeriesGoldenTail(t *testing.T) {
	closes := make([]float64, 90)
	for i := range closes {
		base := 100 + math.Sin(float64(i)/11.0)*1.1
		wave := math.Sin(float64(i)/3.2)*2.4 + math.Cos(float64(i)/5.7)*1.6
		closes[i] = base + wave
	}
	series := sanitizeSeries(computeSTCSeries(closes, 23, 50, config.DefaultSTCKPeriod, config.DefaultSTCDPeriod))
	got := roundSeriesTail(series, 5)
	want := []float64{100, 100, 100, 0, 0}
	if len(got) != len(want) {
		t.Fatalf("tail len=%d want %d tail=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tail[%d]=%v want %v full=%v", i, got[i], want[i], got)
		}
	}
}
