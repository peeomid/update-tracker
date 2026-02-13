package trackers

import (
	"context"
	"fmt"

	"github.com/peeomid/update-tracker/internal/config"
	"github.com/peeomid/update-tracker/internal/execx"
	"github.com/peeomid/update-tracker/internal/httpx"
)

type Registry struct {
	HTTP      httpx.Fetcher
	Exec      execx.Runner
	UserAgent string
}

type Options struct {
	IncludeNotes bool
}

type Result struct {
	Current    string
	Message    string
	Links      map[string]string
	Highlights string
}

type Tracker interface {
	Check(ctx context.Context, prevSeen string, opts Options) (Result, error)
}

func (r Registry) Build(cfg config.TrackerEntry) (Tracker, error) {
	switch cfg.Type {
	case "github":
		switch cfg.Mode {
		case "commit":
			return githubCommit{
				Exec:   r.Exec,
				Repo:   cfg.Repo,
				Branch: cfg.Branch,
			}, nil
		case "release":
			branch := cfg.Branch
			if branch == "" {
				branch = "main"
			}
			return githubReleaseOrCommit{
				HTTP:      r.HTTP,
				Exec:      r.Exec,
				UserAgent: r.UserAgent,
				Repo:      cfg.Repo,
				Fallback: githubCommit{
					Exec:   r.Exec,
					Repo:   cfg.Repo,
					Branch: branch,
				},
			}, nil
		case "pr":
			return githubPR{
				HTTP:      r.HTTP,
				UserAgent: r.UserAgent,
				Repo:      cfg.Repo,
				PR:        cfg.PR,
			}, nil
		default:
			return nil, fmt.Errorf("tracker %s: github mode must be release|commit|pr", cfg.Name)
		}
	case "brew":
		return brewFormula{
			Exec:    r.Exec,
			Formula: cfg.Formula,
		}, nil
	case "npm":
		return npmPackage{
			Exec:    r.Exec,
			Package: cfg.NpmPackage,
		}, nil
	default:
		return nil, fmt.Errorf("tracker %s: unknown type: %s", cfg.Name, cfg.Type)
	}
}
