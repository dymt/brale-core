package decisionutil

import "testing"

func TestNormalizeSymbol(t *testing.T) {
	tests := map[string]string{
		" btcusdt ":     "BTCUSDT",
		"BTC/USDT:USDT": "BTCUSDT",
		"eth":           "ETHUSDT",
	}
	for input, want := range tests {
		if got := NormalizeSymbol(input); got != want {
			t.Fatalf("NormalizeSymbol(%q)=%q want %q", input, got, want)
		}
	}
}
