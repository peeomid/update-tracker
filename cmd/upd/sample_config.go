package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/peeomid/update-tracker/internal/config"
)

func runSampleConfig(args []string) int {
	fs := flag.NewFlagSet("sample-config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageSampleConfig(os.Stdout) }
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr)
		usageSampleConfig(os.Stderr)
		return 2
	}

	fmt.Print(config.SampleYAML())
	return 0
}
