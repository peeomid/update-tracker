package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peeomid/update-tracker/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usageRoot(os.Stderr)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(runCheck(os.Args[2:]))
	case "validate-config":
		os.Exit(runValidateConfig(os.Args[2:]))
	case "sample-config":
		os.Exit(runSampleConfig(os.Args[2:]))
	case "help":
		os.Exit(runHelp(os.Args[2:]))
	case "-h", "--help":
		usageRoot(os.Stdout)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usageRoot(os.Stderr)
		os.Exit(2)
	}
}

func usageRoot(w *os.File) {
	fmt.Fprintln(w, "upd - update tracker")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Checks for updates (GitHub release/commit, brew, npm) and can compare with local installs/clones.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Default files:")
	fmt.Fprintf(w, "  config: %s\n", config.DefaultConfigPath())
	fmt.Fprintf(w, "  state:  %s\n", config.DefaultStatePath())
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  upd check [--config PATH] [--format text|json|markdown] [--only-updates=true|false]")
	fmt.Fprintln(w, "  upd validate-config [--config PATH]")
	fmt.Fprintln(w, "  upd sample-config")
	fmt.Fprintln(w, "  upd help [command]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Exit codes:")
	fmt.Fprintln(w, "  0 = ran all trackers (OK or UPDATE)")
	fmt.Fprintln(w, "  2 = at least 1 tracker had ERROR")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  upd sample-config > ~/.config/update-tracker/config.yaml")
	fmt.Fprintln(w, "  upd validate-config")
	fmt.Fprintln(w, "  upd check --format markdown --only-updates=false")
	fmt.Fprintln(w, "  upd check --format json --notes=false")
}

func rootContext() context.Context {
	return context.Background()
}

func runHelp(args []string) int {
	if len(args) == 0 {
		usageRoot(os.Stdout)
		return 0
	}
	switch args[0] {
	case "check":
		usageCheck(os.Stdout)
		return 0
	case "validate-config":
		usageValidateConfig(os.Stdout)
		return 0
	case "sample-config":
		usageSampleConfig(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command for help: %s\n\n", args[0])
		usageRoot(os.Stderr)
		return 2
	}
}

func usageCheck(w *os.File) {
	fmt.Fprintln(w, "upd check")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Runs all trackers from config and prints 1 line per tracker.")
	fmt.Fprintln(w, "If 1 tracker fails: continues others, and marks that line as ERROR.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --config PATH     Config path (default shown below)")
	fmt.Fprintln(w, "  --format FORMAT   text|json|markdown (default: text)")
	fmt.Fprintln(w, "  --notes BOOL      GitHub release highlights (default: true)")
	fmt.Fprintln(w, "                   Only included when status=update.")
	fmt.Fprintln(w, "  --only-updates BOOL  Print only updates/errors (default: true)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Default paths:")
	fmt.Fprintf(w, "  config: %s\n", config.DefaultConfigPath())
	fmt.Fprintf(w, "  state:  %s\n", config.DefaultStatePath())
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  upd check")
	fmt.Fprintln(w, "  upd check --format json")
	fmt.Fprintln(w, "  upd check --format markdown --notes=false")
	fmt.Fprintln(w, "  upd check --format markdown --only-updates=false")
	fmt.Fprintln(w, "  upd check --config ./my-config.yaml --format text")
}

func usageValidateConfig(w *os.File) {
	fmt.Fprintln(w, "upd validate-config")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Loads config YAML and validates required fields.")
	fmt.Fprintln(w, "Also checks unknown keys (typos).")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --config PATH   Config path (default shown below)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Default path:")
	fmt.Fprintf(w, "  %s\n", config.DefaultConfigPath())
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  upd validate-config")
	fmt.Fprintln(w, "  upd validate-config --config ./config.yaml")
}

func usageSampleConfig(w *os.File) {
	fmt.Fprintln(w, "upd sample-config")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Prints a sample config YAML to stdout.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  upd sample-config")
	fmt.Fprintln(w, "  upd sample-config > ~/.config/update-tracker/config.yaml")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Tip: then validate:")
	fmt.Fprintln(w, "  upd validate-config")
}

func helpRequested(err error) bool {
	return err == flag.ErrHelp || strings.Contains(err.Error(), "help requested")
}
