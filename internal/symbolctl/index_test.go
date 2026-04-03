package symbolctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendSymbolToIndexAppendsWithoutRewritingHeader(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "symbols-index.toml")
	writeTestFile(t, indexPath, strings.Join([]string{
		`# header`,
		`[[symbols]]`,
		`symbol = "ETHUSDT"`,
		`config = "symbols/ETHUSDT.toml"`,
		`strategy = "strategies/ETHUSDT.toml"`,
		``,
	}, "\n"))

	if err := AppendSymbolToIndex(indexPath, "XAGUSDT"); err != nil {
		t.Fatalf("AppendSymbolToIndex() error = %v", err)
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "# header\n[[symbols]]") {
		t.Fatalf("index header should be preserved:\n%s", content)
	}
	if !strings.Contains(content, `symbol = "XAGUSDT"`) {
		t.Fatalf("index missing appended symbol:\n%s", content)
	}
}
