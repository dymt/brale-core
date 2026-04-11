package decisionutil

import symbolpkg "brale-core/internal/pkg/symbol"

// NormalizeSymbol converts a raw symbol or freqtrade pair into the canonical
// internal symbol form used by decision/runtime pipelines.
func NormalizeSymbol(symbol string) string {
	return symbolpkg.Normalize(symbol)
}
