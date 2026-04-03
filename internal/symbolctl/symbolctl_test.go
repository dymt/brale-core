package symbolctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyRepoFixture(t *testing.T, repoRoot string, dstRoot string, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	dst := filepath.Join(dstRoot, rel)
	writeTestFile(t, dst, string(data))
	return dst
}

func writeDotEnv(t *testing.T, repoRoot string) {
	t.Helper()
	writeTestFile(t, filepath.Join(repoRoot, ".env"), strings.Join([]string{
		"LLM_MODEL_INDICATOR=test-indicator",
		"LLM_MODEL_STRUCTURE=test-structure",
		"LLM_MODEL_MECHANICS=test-mechanics",
	}, "\n")+"\n")
}

func writeFreqtradeConfig(t *testing.T, repoRoot string, pairs []string) {
	t.Helper()
	doc := map[string]any{
		"exchange": map[string]any{
			"name":           "binance",
			"pair_whitelist": pairs,
			"pair_blacklist": []string{"BNB/.*"},
		},
		"pairlists": []map[string]any{
			{"method": "StaticPairList"},
		},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal freqtrade config: %v", err)
	}
	writeTestFile(t, filepath.Join(repoRoot, "configs/freqtrade/config.base.json"), string(data)+"\n")
}

func writeIndex(t *testing.T, repoRoot string, content string) string {
	t.Helper()
	path := filepath.Join(repoRoot, "configs/symbols-index.toml")
	writeTestFile(t, path, content)
	return path
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repoRoot := repoRootFromTest(t)
	writeDotEnv(t, root)
	copyRepoFixture(t, repoRoot, root, "configs/symbols/ETHUSDT.toml")
	copyRepoFixture(t, repoRoot, root, "configs/strategies/ETHUSDT.toml")
	return root
}
