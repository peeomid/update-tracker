package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/peeomid/update-tracker/internal/config"
)

func runValidateConfig(args []string) int {
	fs := flag.NewFlagSet("validate-config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageValidateConfig(os.Stdout) }
	configPath := fs.String("config", "", "config path (default: ~/.config/update-tracker/config.yaml)")
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr)
		usageValidateConfig(os.Stderr)
		return 2
	}

	cfg, err := config.Load(config.ResolvePath(*configPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	fmt.Println("OK")
	return 0
}
