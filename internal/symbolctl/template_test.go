package symbolctl

import (
	"path/filepath"
	"testing"
)

func TestLoadTemplateCandidatesUsesOnlyIndexedCompleteSymbols(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
# header
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"

[[symbols]]
symbol = "BADUSDT"
config = "symbols/BADUSDT.toml"
strategy = "strategies/BADUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})
	writeTestFile(t, filepath.Join(repoRoot, "configs/symbols/LOOSEUSDT.toml"), `symbol = "LOOSEUSDT"`)
	writeTestFile(t, filepath.Join(repoRoot, "configs/strategies/LOOSEUSDT.toml"), `symbol = "LOOSEUSDT"`)

	got, err := LoadTemplateCandidates(repoRoot)
	if err != nil {
		t.Fatalf("LoadTemplateCandidates() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(got))
	}
	if got[0].Symbol != "ETHUSDT" {
		t.Fatalf("candidate symbol = %q, want ETHUSDT", got[0].Symbol)
	}
	if got[0].FreqtradePair != "ETH/USDT:USDT" {
		t.Fatalf("candidate pair = %q, want ETH/USDT:USDT", got[0].FreqtradePair)
	}
}

func TestLoadTemplateCandidatesRequiresPairWhitelist(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, nil)

	got, err := LoadTemplateCandidates(repoRoot)
	if err != nil {
		t.Fatalf("LoadTemplateCandidates() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("candidate count = %d, want 0", len(got))
	}
}
