package symbolctl

import (
	"strings"
	"testing"
)

func TestRenderFromTemplateSymbolRewritesTargetFields(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})

	symbolToml, strategyToml, err := RenderFromTemplateSymbol(repoRoot, "ETHUSDT", "XAGUSDT")
	if err != nil {
		t.Fatalf("RenderFromTemplateSymbol() error = %v", err)
	}
	if !strings.Contains(symbolToml, `symbol = "XAGUSDT"`) {
		t.Fatalf("symbol TOML missing target symbol:\n%s", symbolToml)
	}
	if strings.Contains(symbolToml, `symbol = "ETHUSDT"`) {
		t.Fatalf("symbol TOML still contains template symbol:\n%s", symbolToml)
	}
	if !strings.Contains(symbolToml, `# Candlestick intervals to collect`) {
		t.Fatalf("symbol TOML should preserve template comments:\n%s", symbolToml)
	}
	if !strings.Contains(strategyToml, `symbol = "XAGUSDT"`) {
		t.Fatalf("strategy TOML missing target symbol:\n%s", strategyToml)
	}
	if !strings.Contains(strategyToml, `id = "default-xagusdt"`) {
		t.Fatalf("strategy TOML missing rewritten default id:\n%s", strategyToml)
	}
	if !strings.Contains(strategyToml, `# B. 入场模式（Entry Mode）`) {
		t.Fatalf("strategy TOML should preserve template comments:\n%s", strategyToml)
	}
}
