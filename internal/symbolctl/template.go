package symbolctl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"brale-core/internal/config"
	symbolpkg "brale-core/internal/pkg/symbol"

	"github.com/manifoldco/promptui"
)

type TemplateCandidate struct {
	Symbol        string
	ConfigPath    string
	StrategyPath  string
	FreqtradePair string
}

func LoadTemplateCandidates(repoRoot string) ([]TemplateCandidate, error) {
	indexPath := filepath.Join(repoRoot, "configs", "symbols-index.toml")
	indexCfg, err := config.LoadSymbolIndexConfig(indexPath)
	if err != nil {
		return nil, fmt.Errorf("load symbol index %s: %w", indexPath, err)
	}
	pairs, err := loadFreqtradePairWhitelist(filepath.Join(repoRoot, "configs", "freqtrade", "config.base.json"))
	if err != nil {
		return nil, err
	}
	candidates := make([]TemplateCandidate, 0, len(indexCfg.Symbols))
	for _, entry := range indexCfg.Symbols {
		symbol := symbolpkg.Normalize(entry.Symbol)
		configPath := filepath.Join(repoRoot, "configs", filepath.FromSlash(entry.Config))
		strategyPath := filepath.Join(repoRoot, "configs", filepath.FromSlash(entry.Strategy))
		if _, err := os.Stat(configPath); err != nil {
			continue
		}
		if _, err := os.Stat(strategyPath); err != nil {
			continue
		}
		if _, err := config.LoadSymbolConfig(configPath); err != nil {
			continue
		}
		if _, err := config.LoadStrategyConfigWithSymbol(strategyPath, symbol); err != nil {
			continue
		}
		pair := symbolpkg.ToFreqtradePair(symbol)
		if _, ok := pairs[pair]; !ok {
			continue
		}
		candidates = append(candidates, TemplateCandidate{
			Symbol:        symbol,
			ConfigPath:    configPath,
			StrategyPath:  strategyPath,
			FreqtradePair: pair,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Symbol < candidates[j].Symbol
	})
	return candidates, nil
}

func SelectTemplateSymbol(candidates []TemplateCandidate) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("没有可用的模板币种；请先确保 symbols-index.toml 中至少有一个完整配置并已加入 pair_whitelist")
	}
	items := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, fmt.Sprintf("%s  (pair: %s)", candidate.Symbol, candidate.FreqtradePair))
	}
	prompt := promptui.Select{
		Label: "请选择模板币种",
		Items: items,
		Size:  min(12, len(items)),
		Searcher: func(input string, index int) bool {
			item := strings.ToLower(items[index])
			query := strings.ToLower(strings.TrimSpace(input))
			return strings.Contains(item, query)
		},
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt template symbol: %w", err)
	}
	return candidates[idx].Symbol, nil
}

func loadFreqtradePairWhitelist(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read freqtrade config %s: %w", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse freqtrade config %s: %w", path, err)
	}
	exchange, _ := doc["exchange"].(map[string]any)
	rawPairs, _ := exchange["pair_whitelist"].([]any)
	out := make(map[string]struct{}, len(rawPairs))
	for _, item := range rawPairs {
		pair, _ := item.(string)
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		out[pair] = struct{}{}
	}
	return out, nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func findTemplateCandidate(repoRoot string, symbol string) (TemplateCandidate, error) {
	candidates, err := LoadTemplateCandidates(repoRoot)
	if err != nil {
		return TemplateCandidate{}, err
	}
	idx := slices.IndexFunc(candidates, func(candidate TemplateCandidate) bool {
		return candidate.Symbol == symbolpkg.Normalize(symbol)
	})
	if idx < 0 {
		return TemplateCandidate{}, fmt.Errorf("template symbol %s is not a complete local candidate", symbolpkg.Normalize(symbol))
	}
	return candidates[idx], nil
}
