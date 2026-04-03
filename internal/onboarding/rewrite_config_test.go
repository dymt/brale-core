package onboarding

import (
	"strings"
	"testing"
)

func TestRewriteSymbolConfigReplacesOnlySymbolField(t *testing.T) {
	base := strings.Join([]string{
		`# symbol comment`,
		`symbol = "ETHUSDT"`,
		`intervals = ["15m", "1h"]`,
		``,
		`[agent]`,
		`indicator = true`,
	}, "\n")

	got, err := RewriteSymbolConfig(base, "XAGUSDT")
	if err != nil {
		t.Fatalf("RewriteSymbolConfig() error = %v", err)
	}
	if !strings.Contains(got, `symbol = "XAGUSDT"`) {
		t.Fatalf("rewritten symbol config missing target symbol:\n%s", got)
	}
	if strings.Contains(got, `symbol = "ETHUSDT"`) {
		t.Fatalf("rewritten symbol config still contains template symbol:\n%s", got)
	}
	if !strings.Contains(got, `intervals = ["15m", "1h"]`) {
		t.Fatalf("rewritten symbol config should preserve unrelated fields:\n%s", got)
	}
	if !strings.Contains(got, `# symbol comment`) {
		t.Fatalf("rewritten symbol config should preserve comments:\n%s", got)
	}
}

func TestRewriteStrategyConfigRewritesSymbolAndDefaultID(t *testing.T) {
	base := strings.Join([]string{
		`symbol = "ETHUSDT"`,
		`id = "eth-breakout-1"`,
		`rule_chain = "configs/rules/default.json"`,
		``,
		`[risk_management]`,
		`entry_mode = "orderbook"`,
	}, "\n")

	got, err := RewriteStrategyConfig(base, "XAGUSDT")
	if err != nil {
		t.Fatalf("RewriteStrategyConfig() error = %v", err)
	}
	if !strings.Contains(got, `symbol = "XAGUSDT"`) {
		t.Fatalf("rewritten strategy config missing target symbol:\n%s", got)
	}
	if !strings.Contains(got, `id = "default-xagusdt"`) {
		t.Fatalf("rewritten strategy config missing default target id:\n%s", got)
	}
	if strings.Contains(got, `id = "eth-breakout-1"`) {
		t.Fatalf("rewritten strategy config should replace template id:\n%s", got)
	}
	if !strings.Contains(got, `entry_mode = "orderbook"`) {
		t.Fatalf("rewritten strategy config should preserve unrelated fields:\n%s", got)
	}
}
