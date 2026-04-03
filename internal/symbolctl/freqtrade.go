package symbolctl

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	symbolpkg "brale-core/internal/pkg/symbol"
)

func AddPairToFreqtradeConfig(configPath, symbol string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read freqtrade config %s: %w", configPath, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse freqtrade config %s: %w", configPath, err)
	}
	exchange, ok := doc["exchange"].(map[string]any)
	if !ok {
		return fmt.Errorf("parse freqtrade config %s: missing exchange object", configPath)
	}
	pair := symbolpkg.ToFreqtradePair(symbolpkg.Normalize(symbol))
	rawPairs, _ := exchange["pair_whitelist"].([]any)
	pairs := make([]string, 0, len(rawPairs))
	for _, item := range rawPairs {
		s, _ := item.(string)
		s = strings.TrimSpace(s)
		if s != "" {
			pairs = append(pairs, s)
		}
	}
	if !slices.Contains(pairs, pair) {
		pairs = append(pairs, pair)
	}
	exchange["pair_whitelist"] = pairs
	doc["exchange"] = exchange
	rendered, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("render freqtrade config %s: %w", configPath, err)
	}
	if err := writeAtomic(configPath, string(rendered)+"\n"); err != nil {
		return fmt.Errorf("write freqtrade config %s: %w", configPath, err)
	}
	return nil
}
