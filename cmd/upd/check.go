package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/peeomid/update-tracker/internal/app"
	"github.com/peeomid/update-tracker/internal/config"
	"github.com/peeomid/update-tracker/internal/output"
	"github.com/peeomid/update-tracker/internal/state"
)

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageCheck(os.Stdout) }
	configPath := fs.String("config", "", "config path (default: ~/.config/update-tracker/config.yaml)")
	format := fs.String("format", "text", "output format: text|json|markdown")
	notes := fs.Bool("notes", true, "include release highlights (only on update); set --notes=false to disable")
	onlyUpdates := fs.Bool("only-updates", true, "print only updates/errors (default: true); set --only-updates=false to print all")
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr)
		usageCheck(os.Stderr)
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

	statePath := config.DefaultStatePath()
	st, err := state.Load(statePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	report, newState := app.Run(rootContext(), cfg, st, app.Options{
		IncludeNotes: *notes,
	})

	outReport := report
	if *onlyUpdates {
		outReport.Items = nil
		outReport.Summary = app.Summary{}
		for _, it := range report.Items {
			if it.Status == "update" || it.Status == "error" {
				outReport.Items = append(outReport.Items, it)
			}
		}
		for _, it := range outReport.Items {
			switch it.Status {
			case "ok":
				outReport.Summary.OK++
			case "update":
				outReport.Summary.Update++
			case "error":
				outReport.Summary.Error++
			}
		}
	}

	switch *format {
	case "text":
		fmt.Print(output.Text(outReport))
	case "json":
		out, err := output.JSON(outReport)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 2
		}
		fmt.Print(out)
	case "markdown":
		fmt.Print(output.Markdown(outReport))
	default:
		fmt.Fprintln(os.Stderr, "invalid --format (use: text|json|markdown)")
		return 2
	}

	if err := state.Save(statePath, newState); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if report.Summary.Error > 0 {
		return 2
	}
	return 0
}

var _ = context.Canceled
