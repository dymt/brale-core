package symbolctl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"brale-core/internal/config"
	symbolpkg "brale-core/internal/pkg/symbol"
)

type AddSymbolOptions struct {
	RepoRoot       string
	TargetSymbol   string
	TemplateSymbol string
	Interactive    bool
	DryRun         bool
}

var (
	stdout               io.Writer = os.Stdout
	selectTemplateSymbol           = SelectTemplateSymbol
)

func AddSymbol(opts AddSymbolOptions) error {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	targetSymbol := symbolpkg.Normalize(opts.TargetSymbol)
	if targetSymbol == "" {
		return fmt.Errorf("normalize target symbol: empty symbol")
	}
	if !strings.HasSuffix(targetSymbol, "USDT") {
		return fmt.Errorf("target symbol must normalize to *USDT, got %q", targetSymbol)
	}

	indexPath := filepath.Join(repoRoot, "configs", "symbols-index.toml")
	indexCfg, err := config.LoadSymbolIndexConfig(indexPath)
	if err != nil {
		return fmt.Errorf("load symbol index %s: %w", indexPath, err)
	}
	for _, entry := range indexCfg.Symbols {
		if symbolpkg.Normalize(entry.Symbol) == targetSymbol {
			return fmt.Errorf("target symbol already exists in index: %s", targetSymbol)
		}
	}

	candidates, err := LoadTemplateCandidates(repoRoot)
	if err != nil {
		return err
	}
	templateSymbol := symbolpkg.Normalize(opts.TemplateSymbol)
	switch {
	case templateSymbol != "":
		idx := slices.IndexFunc(candidates, func(candidate TemplateCandidate) bool {
			return candidate.Symbol == templateSymbol
		})
		if idx < 0 {
			return fmt.Errorf("template symbol %s is not a complete local candidate", templateSymbol)
		}
	case !opts.Interactive:
		return fmt.Errorf("template symbol is required in non-interactive mode; pass --template-symbol")
	default:
		templateSymbol, err = selectTemplateSymbol(candidates)
		if err != nil {
			return fmt.Errorf("select template symbol: %w", err)
		}
	}

	symbolToml, strategyToml, err := RenderFromTemplateSymbol(repoRoot, templateSymbol, targetSymbol)
	if err != nil {
		return err
	}

	symbolPath := filepath.Join(repoRoot, "configs", "symbols", targetSymbol+".toml")
	strategyPath := filepath.Join(repoRoot, "configs", "strategies", targetSymbol+".toml")
	freqtradePath := filepath.Join(repoRoot, "configs", "freqtrade", "config.base.json")

	if opts.DryRun {
		if _, err := fmt.Fprintf(stdout,
			"dry-run: template=%s target=%s\nwould write %s\nwould write %s\n",
			templateSymbol, targetSymbol, symbolPath, strategyPath,
		); err != nil {
			return fmt.Errorf("write dry-run output: %w", err)
		}
		return nil
	}

	if err := writeAtomic(symbolPath, symbolToml); err != nil {
		return fmt.Errorf("write symbol config %s: %w", symbolPath, err)
	}
	if err := writeAtomic(strategyPath, strategyToml); err != nil {
		return fmt.Errorf("write strategy config %s: %w", strategyPath, err)
	}
	if err := AppendSymbolToIndex(indexPath, targetSymbol); err != nil {
		return err
	}
	if err := AddPairToFreqtradeConfig(freqtradePath, targetSymbol); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "added symbol %s from template %s\n", targetSymbol, templateSymbol); err != nil {
		return fmt.Errorf("write result output: %w", err)
	}
	return nil
}

func writeAtomic(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
