package symbolctl

import (
	"fmt"
	"os"

	"brale-core/internal/onboarding"
	symbolpkg "brale-core/internal/pkg/symbol"
)

func RenderFromTemplateSymbol(repoRoot, templateSymbol, targetSymbol string) (symbolToml, strategyToml string, err error) {
	candidate, err := findTemplateCandidate(repoRoot, templateSymbol)
	if err != nil {
		return "", "", err
	}
	symbolBase, err := os.ReadFile(candidate.ConfigPath)
	if err != nil {
		return "", "", fmt.Errorf("read template symbol config %s: %w", candidate.ConfigPath, err)
	}
	strategyBase, err := os.ReadFile(candidate.StrategyPath)
	if err != nil {
		return "", "", fmt.Errorf("read template strategy config %s: %w", candidate.StrategyPath, err)
	}
	target := symbolpkg.Normalize(targetSymbol)
	symbolToml, err = onboarding.RewriteSymbolConfig(string(symbolBase), target)
	if err != nil {
		return "", "", fmt.Errorf("rewrite symbol config from template %s: %w", candidate.Symbol, err)
	}
	strategyToml, err = onboarding.RewriteStrategyConfig(string(strategyBase), target)
	if err != nil {
		return "", "", fmt.Errorf("rewrite strategy config from template %s: %w", candidate.Symbol, err)
	}
	return symbolToml, strategyToml, nil
}
