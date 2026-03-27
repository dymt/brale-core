package config

import (
	"path/filepath"
	"strings"
)

func ResolveLogPath(sys SystemConfig) string {
	logPath := strings.TrimSpace(sys.LogPath)
	if logPath != "" {
		return logPath
	}
	dir := filepath.Dir(sys.DBPath)
	if dir == "." || dir == "" {
		return "brale-core.log"
	}
	return filepath.Join(dir, "brale-core.log")
}

func SymbolsFromIndex(index SymbolIndexConfig) []string {
	symbols := make([]string, 0, len(index.Symbols))
	for _, item := range index.Symbols {
		normalized := NormalizeSymbol(item.Symbol)
		if normalized == "" {
			continue
		}
		symbols = append(symbols, normalized)
	}
	return symbols
}
