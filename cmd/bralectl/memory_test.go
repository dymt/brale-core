package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"brale-core/internal/memory"
	"brale-core/internal/store"
)

func TestMemoryListCommandFiltersJSONBySource(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "brale.db")
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := store.Migrate(db, store.MigrateOptions{Full: true}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	sem := memory.NewSemanticMemory(store.NewStore(db), 100)
	if err := sem.SaveRule(memory.Rule{
		Symbol:     "BTCUSDT",
		RuleText:   "user rule",
		Source:     "user",
		Confidence: 0.8,
		Active:     true,
	}); err != nil {
		t.Fatalf("save user rule: %v", err)
	}
	if err := sem.SaveRule(memory.Rule{
		Symbol:     "BTCUSDT",
		RuleText:   "reflector rule",
		Source:     "reflector",
		Confidence: 0.9,
		Active:     true,
	}); err != nil {
		t.Fatalf("save reflector rule: %v", err)
	}

	out, errOut, err := executeRootCommand(t, "--json", "memory", "list", "--db", dbPath, "--source", "reflector")
	if err != nil {
		t.Fatalf("execute command: %v\nstderr=%s", err, errOut)
	}

	var rules []memory.Rule
	if err := json.Unmarshal([]byte(out), &rules); err != nil {
		t.Fatalf("unmarshal rules: %v\nout=%s", err, out)
	}
	if len(rules) != 1 {
		t.Fatalf("rules=%v want 1 reflector rule", rules)
	}
	if rules[0].Source != "reflector" || rules[0].RuleText != "reflector rule" {
		t.Fatalf("rules=%v", rules)
	}
}
