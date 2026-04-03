package symbolctl

import (
	"fmt"
	"os"
	"strings"

	"brale-core/internal/config"
	symbolpkg "brale-core/internal/pkg/symbol"
)

func AppendSymbolToIndex(indexPath, symbol string) error {
	targetSymbol := symbolpkg.Normalize(symbol)
	indexCfg, err := config.LoadSymbolIndexConfig(indexPath)
	if err != nil {
		return fmt.Errorf("load symbol index %s: %w", indexPath, err)
	}
	for _, entry := range indexCfg.Symbols {
		if symbolpkg.Normalize(entry.Symbol) == targetSymbol {
			return fmt.Errorf("symbol already exists in index: %s", targetSymbol)
		}
	}
	existing, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read symbol index %s: %w", indexPath, err)
	}
	content := string(existing)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += fmt.Sprintf(`
[[symbols]]
symbol = "%s"
config = "symbols/%s.toml"
strategy = "strategies/%s.toml"
`, targetSymbol, targetSymbol, targetSymbol)
	if err := writeAtomic(indexPath, content); err != nil {
		return fmt.Errorf("write symbol index %s: %w", indexPath, err)
	}
	return nil
}
