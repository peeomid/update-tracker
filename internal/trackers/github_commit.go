package trackers

import (
	"context"
	"fmt"
	"strings"

	"github.com/peeomid/update-tracker/internal/execx"
)

type githubCommit struct {
	Exec   execx.Runner
	Repo   string
	Branch string
}

func (g githubCommit) Check(ctx context.Context, prevSeen string, opts Options) (Result, error) {
	repoURL := fmt.Sprintf("https://github.com/%s", g.Repo)
	remote := fmt.Sprintf("https://github.com/%s.git", g.Repo)
	ref := fmt.Sprintf("refs/heads/%s", g.Branch)

	_ = opts
	out, err := g.Exec.Run(ctx, "git", "ls-remote", remote, ref)
	if err != nil {
		return Result{}, fmt.Errorf("git ls-remote: %w", err)
	}
	fields := strings.Fields(out)
	if len(fields) < 1 {
		return Result{}, fmt.Errorf("git ls-remote: empty output")
	}
	sha := fields[0]

	links := map[string]string{"repo": repoURL}
	prev := strings.TrimSpace(prevSeen)
	if prev != "" && prev != sha {
		links["compare"] = fmt.Sprintf("%s/compare/%s...%s", repoURL, prev, sha)
	}

	msg := fmt.Sprintf("latest commit on %s (%s)", g.Branch, shortSHA(sha))
	if prev != "" && prev != sha {
		msg = fmt.Sprintf("new commits on %s (%s -> %s)", g.Branch, shortSHA(prev), shortSHA(sha))
	}
	return Result{
		Current: sha,
		Message: msg,
		Links:   links,
	}, nil
}
