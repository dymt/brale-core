package main

import (
	"os"

	symbolpkg "brale-core/internal/pkg/symbol"
	"brale-core/internal/symbolctl"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func addSymbolCmd() *cobra.Command {
	var repoRoot string
	var templateSymbol string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "add-symbol <SYMBOL>",
		Short: "添加一个新的交易币种",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return symbolctl.AddSymbol(symbolctl.AddSymbolOptions{
				RepoRoot:       repoRoot,
				TargetSymbol:   symbolpkg.Normalize(args[0]),
				TemplateSymbol: symbolpkg.Normalize(templateSymbol),
				Interactive:    term.IsTerminal(int(os.Stdin.Fd())),
				DryRun:         dryRun,
			})
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo", ".", "项目根目录路径")
	cmd.Flags().StringVar(&templateSymbol, "template-symbol", "", "模板币种；为空时进入交互选择")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只预览生成内容，不写文件")
	return cmd
}
