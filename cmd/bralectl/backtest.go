package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"brale-core/internal/backtest"
	"brale-core/internal/config"
	"brale-core/internal/decision/decisionutil"
	"brale-core/internal/decision/direction"
	runtimecfg "brale-core/internal/runtime"
	"brale-core/internal/store"

	"github.com/spf13/cobra"
)

func backtestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "历史规则回放工具",
	}
	cmd.AddCommand(backtestRulesCmd())
	return cmd
}

func backtestRulesCmd() *cobra.Command {
	var (
		symbol     string
		fromRaw    string
		toRaw      string
		dbPath     string
		systemPath string
		indexPath  string
		format     string
		outputPath string
	)
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "重放历史 Gate 决策并输出回测报告",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, timeRange, err := buildRuleReplay(systemPath, indexPath, dbPath, symbol, fromRaw, toRaw)
			if err != nil {
				return err
			}
			result, err := runner.Run(cmd.Context(), symbol, timeRange)
			if err != nil {
				return err
			}
			return writeReplayOutput(cmd.OutOrStdout(), result, format, outputPath)
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "", "交易对，例如 BTCUSDT")
	cmd.Flags().StringVar(&fromRaw, "from", "", "开始日期，支持 YYYY-MM-DD 或 RFC3339")
	cmd.Flags().StringVar(&toRaw, "to", "", "结束日期，支持 YYYY-MM-DD 或 RFC3339")
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite 数据库路径（留空时从 system.toml 读取）")
	cmd.Flags().StringVar(&systemPath, "system", "configs/system.toml", "system.toml 路径")
	cmd.Flags().StringVar(&indexPath, "index", "configs/symbols-index.toml", "symbols-index.toml 路径")
	cmd.Flags().StringVar(&format, "format", "text", "输出格式：text|html|json")
	cmd.Flags().StringVar(&outputPath, "output", "", "报告输出文件路径（html 格式必须指定）")
	_ = cmd.MarkFlagRequired("symbol")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func buildRuleReplay(systemPath, indexPath, dbPath, symbol, fromRaw, toRaw string) (backtest.RuleReplay, backtest.TimeRange, error) {
	systemPath = filepath.Clean(systemPath)
	indexPath = filepath.Clean(indexPath)
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("symbol is required")
	}
	fromTime, err := parseReplayTime(fromRaw, false)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("parse --from: %w", err)
	}
	toTime, err := parseReplayTime(toRaw, true)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("parse --to: %w", err)
	}
	if toTime.Before(fromTime) {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("--to must be greater than or equal to --from")
	}
	sys, err := config.LoadSystemConfig(systemPath)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("load system config: %w", err)
	}
	indexCfg, err := config.LoadSymbolIndexConfig(indexPath)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("load symbol index config: %w", err)
	}
	var (
		runner              = backtest.RuleReplay{}
		scoreThreshold      float64
		confidenceThreshold float64
		found               bool
	)
	for _, item := range indexCfg.Symbols {
		if decisionutil.NormalizeSymbol(item.Symbol) != symbol {
			continue
		}
		symbolCfg, _, bind, loadErr := runtimecfg.LoadSymbolConfigs(sys, indexPath, item)
		if loadErr != nil {
			return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("load symbol configs: %w", loadErr)
		}
		scoreThreshold = symbolCfg.Consensus.ScoreThreshold
		if scoreThreshold <= 0 {
			scoreThreshold = direction.ThresholdScore()
		}
		confidenceThreshold = symbolCfg.Consensus.ConfidenceThreshold
		if confidenceThreshold <= 0 {
			confidenceThreshold = direction.ThresholdConfidence()
		}
		runner = backtest.RuleReplay{
			Binding:             bind,
			ScoreThreshold:      scoreThreshold,
			ConfidenceThreshold: confidenceThreshold,
		}
		found = true
		break
	}
	if !found {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("symbol %s not found in %s", symbol, indexPath)
	}
	resolvedDBPath, err := resolveReplayDBPath(systemPath, dbPath, sys.DBPath)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, err
	}
	db, err := store.OpenSQLite(resolvedDBPath)
	if err != nil {
		return backtest.RuleReplay{}, backtest.TimeRange{}, fmt.Errorf("open sqlite: %w", err)
	}
	runner.Store = store.NewStore(db)
	return runner, backtest.TimeRange{
		StartUnix: fromTime.UTC().Unix(),
		EndUnix:   toTime.UTC().Unix(),
	}, nil
}

func parseReplayTime(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("value is empty")
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", time.DateOnly}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		if layout == time.DateOnly && endOfDay {
			return parsed.UTC().Add(24*time.Hour - time.Second), nil
		}
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", raw)
}

func resolveReplayDBPath(systemPath, override, configured string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return filepath.Clean(trimmed), nil
	}
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return "", fmt.Errorf("db path is empty")
	}
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured), nil
	}
	candidates := []string{
		filepath.Clean(configured),
		filepath.Join(filepath.Dir(systemPath), configured),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return filepath.Clean(configured), nil
}

func writeReplayOutput(stdout io.Writer, result *backtest.ReplayResult, format, outputPath string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "text"
	}
	if format == "html" && strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("--output is required for html format")
	}
	var buf bytes.Buffer
	switch format {
	case "text":
		if err := backtest.WriteTextReport(&buf, result); err != nil {
			return err
		}
	case "html":
		if err := backtest.WriteHTMLReport(&buf, result); err != nil {
			return err
		}
	case "json":
		if err := printJSON(&buf, result); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
	if trimmed := strings.TrimSpace(outputPath); trimmed != "" {
		return os.WriteFile(trimmed, buf.Bytes(), 0o644)
	}
	_, err := stdout.Write(buf.Bytes())
	return err
}
