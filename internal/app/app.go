package app

import (
	"context"
	"time"

	"github.com/peeomid/update-tracker/internal/config"
	"github.com/peeomid/update-tracker/internal/execx"
	"github.com/peeomid/update-tracker/internal/httpx"
	"github.com/peeomid/update-tracker/internal/state"
	"github.com/peeomid/update-tracker/internal/trackers"
)

type Report struct {
	SchemaVersion int           `json:"schemaVersion"`
	RunAt         time.Time     `json:"runAt"`
	Summary       Summary       `json:"summary"`
	Items         []ReportItem  `json:"items"`
	RawConfigPath string        `json:"-"`
	Duration      time.Duration `json:"-"`
}

type Summary struct {
	OK     int `json:"ok"`
	Update int `json:"update"`
	Error  int `json:"error"`
}

type ReportItem struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Mode       string            `json:"mode,omitempty"`
	Label      string            `json:"label,omitempty"`
	Group      string            `json:"group,omitempty"`
	Display    string            `json:"display,omitempty"`
	Status     string            `json:"status"`
	Prev       string            `json:"prev,omitempty"`
	Current    string            `json:"current,omitempty"`
	Latest     string            `json:"latest,omitempty"`
	Local      string            `json:"local,omitempty"`
	Message    string            `json:"message"`
	Links      map[string]string `json:"links,omitempty"`
	Highlights string            `json:"highlights,omitempty"`
	Error      string            `json:"error,omitempty"`
	LocalError string            `json:"localError,omitempty"`
}

type Options struct {
	IncludeNotes bool
}

func Run(ctx context.Context, cfg config.Config, st state.State, opts Options) (Report, state.State) {
	start := time.Now()
	runAt := time.Now()

	timeout := time.Duration(cfg.Defaults.TimeoutSeconds) * time.Second
	httpClient := httpx.NewCachedFetcher(httpx.NewClient(timeout))
	execRunner := execx.NewCachedRunner(execx.OSRunner{})

	registry := trackers.Registry{
		HTTP:      httpClient,
		Exec:      execRunner,
		UserAgent: cfg.Defaults.UserAgent,
	}

	run := runner{
		Registry:    registry,
		Timeout:     timeout,
		Retries:     cfg.Defaults.Retries,
		Concurrency: cfg.Defaults.Concurrency,
		RunAt:       runAt,
		Options:     opts,
	}

	items, nextState := run.Run(ctx, cfg.Trackers, st)

	var summary Summary
	for _, it := range items {
		switch it.Status {
		case "ok":
			summary.OK++
		case "update":
			summary.Update++
		case "error":
			summary.Error++
		}
	}

	return Report{
		SchemaVersion: 1,
		RunAt:         runAt,
		Summary:       summary,
		Items:         items,
		Duration:      time.Since(start),
	}, nextState
}
