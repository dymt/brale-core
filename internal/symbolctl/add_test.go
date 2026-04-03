package symbolctl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddSymbolWithExplicitTemplateWritesFilesAndConfig(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})

	var out bytes.Buffer
	prevStdout := stdout
	stdout = &out
	t.Cleanup(func() { stdout = prevStdout })

	err := AddSymbol(AddSymbolOptions{
		RepoRoot:       repoRoot,
		TargetSymbol:   "xag",
		TemplateSymbol: "ethusdt",
	})
	if err != nil {
		t.Fatalf("AddSymbol() error = %v", err)
	}

	symbolData, err := os.ReadFile(filepath.Join(repoRoot, "configs/symbols/XAGUSDT.toml"))
	if err != nil {
		t.Fatalf("read symbol file: %v", err)
	}
	if !strings.Contains(string(symbolData), `symbol = "XAGUSDT"`) {
		t.Fatalf("symbol file missing target symbol:\n%s", string(symbolData))
	}

	strategyData, err := os.ReadFile(filepath.Join(repoRoot, "configs/strategies/XAGUSDT.toml"))
	if err != nil {
		t.Fatalf("read strategy file: %v", err)
	}
	if !strings.Contains(string(strategyData), `id = "default-xagusdt"`) {
		t.Fatalf("strategy file missing rewritten id:\n%s", string(strategyData))
	}

	indexData, err := os.ReadFile(filepath.Join(repoRoot, "configs/symbols-index.toml"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	if !strings.Contains(string(indexData), `symbol = "XAGUSDT"`) {
		t.Fatalf("index missing target symbol:\n%s", string(indexData))
	}

	freqtradeData, err := os.ReadFile(filepath.Join(repoRoot, "configs/freqtrade/config.base.json"))
	if err != nil {
		t.Fatalf("read freqtrade config: %v", err)
	}
	if !strings.Contains(string(freqtradeData), `"XAG/USDT:USDT"`) {
		t.Fatalf("freqtrade config missing target pair:\n%s", string(freqtradeData))
	}
	if !strings.Contains(out.String(), "XAGUSDT") {
		t.Fatalf("stdout should mention target symbol:\n%s", out.String())
	}
}

func TestAddSymbolInteractiveTemplateUsesSelector(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})

	called := false
	prevSelector := selectTemplateSymbol
	selectTemplateSymbol = func(candidates []TemplateCandidate) (string, error) {
		called = true
		return "ETHUSDT", nil
	}
	t.Cleanup(func() { selectTemplateSymbol = prevSelector })

	if err := AddSymbol(AddSymbolOptions{
		RepoRoot:     repoRoot,
		TargetSymbol: "XAGUSDT",
		Interactive:  true,
	}); err != nil {
		t.Fatalf("AddSymbol() error = %v", err)
	}

	if !called {
		t.Fatal("interactive template selector was not called")
	}
}

func TestAddSymbolOverwritesLooseFiles(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})
	writeTestFile(t, filepath.Join(repoRoot, "configs/symbols/XAGUSDT.toml"), `symbol = "STALEUSDT"`)
	writeTestFile(t, filepath.Join(repoRoot, "configs/strategies/XAGUSDT.toml"), `id = "stale"`)

	if err := AddSymbol(AddSymbolOptions{
		RepoRoot:       repoRoot,
		TargetSymbol:   "XAGUSDT",
		TemplateSymbol: "ETHUSDT",
	}); err != nil {
		t.Fatalf("AddSymbol() error = %v", err)
	}

	symbolData, err := os.ReadFile(filepath.Join(repoRoot, "configs/symbols/XAGUSDT.toml"))
	if err != nil {
		t.Fatalf("read symbol file: %v", err)
	}
	if strings.Contains(string(symbolData), "STALEUSDT") {
		t.Fatalf("loose symbol file was not overwritten:\n%s", string(symbolData))
	}

	strategyData, err := os.ReadFile(filepath.Join(repoRoot, "configs/strategies/XAGUSDT.toml"))
	if err != nil {
		t.Fatalf("read strategy file: %v", err)
	}
	if strings.Contains(string(strategyData), `id = "stale"`) {
		t.Fatalf("loose strategy file was not overwritten:\n%s", string(strategyData))
	}
}

func TestAddSymbolDryRunDoesNotWriteFiles(t *testing.T) {
	repoRoot := newTestRepo(t)
	writeIndex(t, repoRoot, `
[[symbols]]
symbol = "ETHUSDT"
config = "symbols/ETHUSDT.toml"
strategy = "strategies/ETHUSDT.toml"
`)
	writeFreqtradeConfig(t, repoRoot, []string{"ETH/USDT:USDT"})

	indexBefore, err := os.ReadFile(filepath.Join(repoRoot, "configs/symbols-index.toml"))
	if err != nil {
		t.Fatalf("read index before: %v", err)
	}

	var out bytes.Buffer
	prevStdout := stdout
	stdout = &out
	t.Cleanup(func() { stdout = prevStdout })

	if err := AddSymbol(AddSymbolOptions{
		RepoRoot:       repoRoot,
		TargetSymbol:   "XAGUSDT",
		TemplateSymbol: "ETHUSDT",
		DryRun:         true,
	}); err != nil {
		t.Fatalf("AddSymbol() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "configs/symbols/XAGUSDT.toml")); !os.IsNotExist(err) {
		t.Fatalf("dry run should not write symbol file, stat err = %v", err)
	}

	indexAfter, err := os.ReadFile(filepath.Join(repoRoot, "configs/symbols-index.toml"))
	if err != nil {
		t.Fatalf("read index after: %v", err)
	}
	if string(indexAfter) != string(indexBefore) {
		t.Fatalf("dry run should not change index:\n%s", string(indexAfter))
	}
	if !strings.Contains(out.String(), "dry-run") && !strings.Contains(out.String(), "dry run") {
		t.Fatalf("dry run output should mention preview:\n%s", out.String())
	}
}
