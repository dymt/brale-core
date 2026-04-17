package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"brale-core/internal/e2e"
	_ "brale-core/internal/e2e/suites" // register all suites

	"github.com/spf13/cobra"
)

func testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "E2E 测试工具 / E2E test runner",
	}
	cmd.AddCommand(testRunCmd())
	cmd.AddCommand(testListCmd())
	return cmd
}

func testRunCmd() *cobra.Command {
	var (
		profile        string
		suites         string
		symbol         string
		ftEndpoint     string
		pgPort         string
		mcpEndpoint    string
		timeout        time.Duration
		forceCloseWait time.Duration
		reportDir      string
		bootstrap      string
		telegramToken  string
		telegramChatID int64
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "执行 E2E 测试套件 / Run E2E test suites",
		Long: `Run one or more E2E test suites against a running brale-core stack.

Examples:
  # Quick regression (default suites)
  bralectl test run --endpoint http://127.0.0.1:19991

  # Run specific suites
  bralectl test run --suites ctl,decision,pricing --symbol SOLUSDT

  # Full lifecycle with force close
  bralectl test run --suites lifecycle-force-close --timeout 30m

  # Run against E2E stack with custom ports
  bralectl test run --endpoint http://127.0.0.1:19991 --ft-endpoint http://127.0.0.1:18080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var suiteList []string
			if suites != "" {
				suiteList = strings.Split(suites, ",")
			}
			telegramToken = strings.TrimSpace(telegramToken)
			if (telegramToken == "" && telegramChatID != 0) || (telegramToken != "" && telegramChatID == 0) {
				return fmt.Errorf("telegram report requires both --telegram-token and --telegram-chat-id")
			}

			cfg := e2e.RunConfig{
				Profile:         profile,
				Suites:          suiteList,
				Symbol:          symbol,
				Endpoint:        flagEndpoint,
				FTEndpoint:      ftEndpoint,
				PGPort:          pgPort,
				MCPEndpoint:     mcpEndpoint,
				Timeout:         timeout,
				ForceCloseAfter: forceCloseWait,
				ReportDir:       reportDir,
				Bootstrap:       bootstrap,
				TelegramToken:   telegramToken,
				TelegramChatID:  telegramChatID,
			}

			runner := e2e.NewRunner(cfg)
			report, err := runner.Run()
			if err != nil {
				return fmt.Errorf("runner error: %w", err)
			}

			if flagJSON {
				return printJSON(cmd.OutOrStdout(), report)
			}

			printTestReport(cmd, report)

			if !report.Passed {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "quick-lifecycle", "E2E 配置 profile 名称")
	cmd.Flags().StringVar(&suites, "suites", "", "要执行的 suite 列表，逗号分隔 (默认: ctl,decision,open-fast,pricing,reconcile)")
	cmd.Flags().StringVar(&symbol, "symbol", "SOLUSDT", "测试用交易对")
	cmd.Flags().StringVar(&ftEndpoint, "ft-endpoint", "http://127.0.0.1:18080", "Freqtrade API 地址")
	cmd.Flags().StringVar(&pgPort, "pg-port", "15432", "PostgreSQL 端口")
	cmd.Flags().StringVar(&mcpEndpoint, "mcp-endpoint", "http://127.0.0.1:18765", "MCP HTTP 地址")
	cmd.Flags().DurationVar(&timeout, "timeout", 15*time.Minute, "全局超时时间")
	cmd.Flags().DurationVar(&forceCloseWait, "force-close-after", 5*time.Minute, "lifecycle-force-close 等待自然平仓的时间")
	cmd.Flags().StringVar(&reportDir, "report-dir", "_output/e2e", "报告输出目录")
	cmd.Flags().StringVar(&bootstrap, "bootstrap", "auto", "引导模式: auto | strict")
	cmd.Flags().StringVar(&telegramToken, "telegram-token", envString("BRALE_E2E_TELEGRAM_TOKEN", "NOTIFICATION_TELEGRAM_TOKEN"), "E2E 报告 Telegram Bot token")
	cmd.Flags().Int64Var(&telegramChatID, "telegram-chat-id", envInt64("BRALE_E2E_TELEGRAM_CHAT_ID", "NOTIFICATION_TELEGRAM_CHAT_ID"), "E2E 报告 Telegram chat_id")
	return cmd
}

func testListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有可用的 E2E 测试套件 / List available test suites",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := e2e.SuiteNames()
			if flagJSON {
				return printJSON(cmd.OutOrStdout(), names)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Available E2E suites (%d):\n", len(names))
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", n)
			}
			return nil
		},
	}
}

func printTestReport(cmd *cobra.Command, report e2e.RunReport) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "\n╔══════════════════════════════════════════════╗\n")
	fmt.Fprintf(w, "║           E2E Test Report                    ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Profile:  %-35s║\n", report.Profile)
	fmt.Fprintf(w, "║ Symbol:   %-35s║\n", report.Symbol)
	fmt.Fprintf(w, "║ Duration: %-35s║\n", report.Duration.Round(time.Millisecond))
	if report.Passed {
		fmt.Fprintf(w, "║ Result:   ✅ PASSED                           ║\n")
	} else {
		fmt.Fprintf(w, "║ Result:   ❌ FAILED                           ║\n")
	}
	fmt.Fprintf(w, "╠══════════════════════════════════════════════╣\n")

	for _, s := range report.Suites {
		icon := "✅"
		switch s.Status {
		case e2e.StatusFail:
			icon = "❌"
		case e2e.StatusSkip:
			icon = "⏭️"
		case e2e.StatusBlocked:
			icon = "🚫"
		}
		line := fmt.Sprintf("%s %s (%s)", icon, s.Name, s.Duration.Round(time.Millisecond))
		fmt.Fprintf(w, "║ %-45s║\n", line)
		if s.Error != "" {
			fmt.Fprintf(w, "║   → %-41s║\n", s.Error)
		}
		for _, c := range s.Checks {
			ck := "✓"
			if !c.Passed {
				ck = "✗"
			}
			fmt.Fprintf(w, "║   %s %-42s║\n", ck, c.Name)
		}
	}

	fmt.Fprintf(w, "╚══════════════════════════════════════════════╝\n\n")

	if report.RunID != "" {
		fmt.Fprintf(w, "Report saved: %s\n", report.RunID)
	}
}

func envString(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envInt64(keys ...string) int64 {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
