package symbolctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddPairToFreqtradeConfigAppendsNormalizedPair(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.base.json")
	writeTestFile(t, configPath, `{"exchange":{"pair_whitelist":["ETH/USDT:USDT"],"pair_blacklist":["BNB/.*"]},"pairlists":[{"method":"StaticPairList"}]}`)

	if err := AddPairToFreqtradeConfig(configPath, "1000pepe"); err != nil {
		t.Fatalf("AddPairToFreqtradeConfig() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"1000PEPE/USDT:USDT"`) {
		t.Fatalf("config missing normalized pair:\n%s", content)
	}
}
