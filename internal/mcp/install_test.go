package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallMergesMCPConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	systemPath := filepath.Join(dir, "system.toml")
	indexPath := filepath.Join(dir, "symbols-index.toml")
	auditPath := filepath.Join(dir, "audit.jsonl")
	commandPath := filepath.Join(dir, "bralectl")
	if err := os.WriteFile(systemPath, []byte("db_path = \"db.sqlite\"\n"), 0o644); err != nil {
		t.Fatalf("write system: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("[[symbols]]\nsymbol = \"BTCUSDT\"\nconfig = \"symbols/BTCUSDT.toml\"\nstrategy = \"strategies/BTCUSDT.toml\"\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write command: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"mcpServers":{"existing":{"command":"existing"}}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := Install(InstallOptions{
		Name:       "brale-core",
		Command:    commandPath,
		ConfigPath: configPath,
		Endpoint:   "http://127.0.0.1:9991",
		SystemPath: systemPath,
		IndexPath:  indexPath,
		AuditPath:  auditPath,
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.ConfigPath != configPath {
		t.Fatalf("ConfigPath=%s want %s", result.ConfigPath, configPath)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	servers := doc["mcpServers"].(map[string]any)
	if _, ok := servers["existing"]; !ok {
		t.Fatalf("existing server missing: %v", servers)
	}
	brale := servers["brale-core"].(map[string]any)
	if brale["command"] != commandPath {
		t.Fatalf("command=%v want %s", brale["command"], commandPath)
	}
	args := toStringSlice(t, brale["args"])
	for _, want := range []string{
		"--endpoint",
		"http://127.0.0.1:9991",
		"mcp",
		"serve",
		"--mode",
		"stdio",
		"--system",
		systemPath,
		"--index",
		indexPath,
		"--audit-log",
		auditPath,
	} {
		if !containsString(args, want) {
			t.Fatalf("args=%v missing %q", args, want)
		}
	}
}

func TestInstallRejectsMissingCommand(t *testing.T) {
	dir := t.TempDir()
	systemPath := filepath.Join(dir, "system.toml")
	indexPath := filepath.Join(dir, "symbols-index.toml")
	if err := os.WriteFile(systemPath, []byte("db_path = \"db.sqlite\"\n"), 0o644); err != nil {
		t.Fatalf("write system: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("[[symbols]]\nsymbol = \"BTCUSDT\"\nconfig = \"symbols/BTCUSDT.toml\"\nstrategy = \"strategies/BTCUSDT.toml\"\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	_, err := Install(InstallOptions{
		Command:    filepath.Join(dir, "missing-bralectl"),
		ConfigPath: filepath.Join(dir, "mcp.json"),
		SystemPath: systemPath,
		IndexPath:  indexPath,
	})
	if err == nil || !strings.Contains(err.Error(), "command path") {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallRejectsDirectoryForSystemPath(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "symbols-index.toml")
	commandPath := filepath.Join(dir, "bralectl")
	if err := os.WriteFile(indexPath, []byte("[[symbols]]\nsymbol = \"BTCUSDT\"\nconfig = \"symbols/BTCUSDT.toml\"\nstrategy = \"strategies/BTCUSDT.toml\"\n"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write command: %v", err)
	}
	_, err := Install(InstallOptions{
		Command:    commandPath,
		ConfigPath: filepath.Join(dir, "mcp.json"),
		SystemPath: dir,
		IndexPath:  indexPath,
	})
	if err == nil || !strings.Contains(err.Error(), "file path") {
		t.Fatalf("err=%v", err)
	}
}

func toStringSlice(t *testing.T, v any) []string {
	t.Helper()
	raw := v.([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, item.(string))
	}
	return out
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
