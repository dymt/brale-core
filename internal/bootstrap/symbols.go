package bootstrap

import (
	"brale-core/internal/config"
	"brale-core/internal/pkg/symbol"
)

func canonicalSymbol(raw string) string {
	return symbol.Normalize(raw)
}

func canonicalSymbolFromIndexEntry(entry config.SymbolIndexEntry) string {
	return canonicalSymbol(entry.Symbol)
}
