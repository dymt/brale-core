package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"brale-core/internal/onboarding"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	if len(args) > 0 {
		switch args[0] {
		case "prepare-stack":
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
				return 1
			}
			return runPrepareStackCommand(args[1:], cwd, stdout, stderr)
		case "serve":
			return runServe(args[1:], stderr)
		case "help", "-h", "--help":
			printHelp(stdout)
			return 0
		}
	}

	return runServe(args, stderr)
}

func runPrepareStackCommand(args []string, cwd string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		printPrepareStackHelp(stdout)
		return 0
	}
	if err := onboarding.RunPrepareStack(args, cwd, stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "prepare stack: %v\n", err)
		return 1
	}
	return 0
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help":
			return true
		}
	}
	return false
}

func runServe(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	addr := fs.String("addr", "127.0.0.1:9992", "http listen addr")
	repo := fs.String("repo", ".", "repository root")
	basePath := fs.String("base", "/", "http base path")
	allowNonLoopback := fs.Bool("allow-non-loopback", false, "allow non-loopback requests for trusted containerized deployments")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	repoRoot, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repo root: %v\n", err)
		return 1
	}

	handler, err := onboarding.Server{RepoRoot: repoRoot, BasePath: *basePath, AllowNonLoopback: *allowNonLoopback}.Handler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "init onboarding server: %v\n", err)
		return 1
	}

	listenURL := "http://" + *addr
	if len(*addr) > 0 && (*addr)[0] == ':' {
		listenURL = "http://127.0.0.1" + *addr
	}
	fmt.Printf("onboarding listening on %s\n", listenURL)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "serve onboarding: %v\n", err)
		return 1
	}
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "onboarding command")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  onboarding serve [flags]")
	fmt.Fprintln(w, "  onboarding prepare-stack [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "When called without a subcommand, onboarding runs in serve mode for compatibility.")
	fmt.Fprintln(w, "Run `onboarding serve --help` or `onboarding prepare-stack --help` for details.")
}

func printPrepareStackHelp(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintln(w, "Usage of prepare-stack:")
	fmt.Fprintln(w, "  -env-file string")
	fmt.Fprintln(w, "    \tenvironment file (default \".env\")")
	fmt.Fprintln(w, "  -config-in string")
	fmt.Fprintln(w, "    \tfreqtrade base config (default \"configs/freqtrade/config.base.json\")")
	fmt.Fprintln(w, "  -config-out string")
	fmt.Fprintln(w, "    \tfreqtrade runtime config (default \"data/freqtrade/user_data/config.json\")")
	fmt.Fprintln(w, "  -proxy-env-out string")
	fmt.Fprintln(w, "    \tstack proxy env output (default \"data/freqtrade/proxy.env\")")
	fmt.Fprintln(w, "  -system-in string")
	fmt.Fprintln(w, "    \tsystem config input (default \"configs/system.toml\")")
	fmt.Fprintln(w, "  -system-out string")
	fmt.Fprintln(w, "    \toptional system config output")
	fmt.Fprintln(w, "  -exec-endpoint string")
	fmt.Fprintln(w, "    \texecution endpoint in output system config (default \"http://freqtrade:8080/api/v1\")")
	fmt.Fprintln(w, "  -check-only")
	fmt.Fprintln(w, "    \tvalidate config only")
}
