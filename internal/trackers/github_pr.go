package trackers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/peeomid/update-tracker/internal/httpx"
)

type githubPR struct {
	HTTP      httpx.Fetcher
	UserAgent string
	Repo      string
	PR        int
}

type githubPRResp struct {
	Number  int    `json:"number"`
	State   string `json:"state"` // open|closed
	Draft   bool   `json:"draft"`
	Merged  bool   `json:"merged"`
	HTMLURL string `json:"html_url"`
	Head    struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

type githubCommitStatusResp struct {
	State string `json:"state"` // error|failure|pending|success
}

type githubCheckRunsResp struct {
	TotalCount int `json:"total_count"`
	CheckRuns  []struct {
		Status     string  `json:"status"`     // queued|in_progress|completed
		Conclusion *string `json:"conclusion"` // success|failure|neutral|cancelled|timed_out|skipped|action_required|...
	} `json:"check_runs"`
}

func (g githubPR) Check(ctx context.Context, prevSeen string, opts Options) (Result, error) {
	_ = opts

	prURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", g.Repo, g.PR)
	body, err := g.HTTP.Get(ctx, prURL, map[string]string{
		"User-Agent": g.UserAgent,
		"Accept":     "application/vnd.github+json",
	})
	if err != nil {
		return Result{}, fmt.Errorf("fetch pr: %w", err)
	}

	var pr githubPRResp
	if err := json.Unmarshal(body, &pr); err != nil {
		return Result{}, fmt.Errorf("parse pr json: %w", err)
	}
	if pr.Number == 0 {
		pr.Number = g.PR
	}
	if strings.TrimSpace(pr.Head.SHA) == "" {
		return Result{}, fmt.Errorf("pr: missing head sha")
	}

	state := strings.ToLower(strings.TrimSpace(pr.State))
	if pr.Merged {
		state = "merged"
	}
	if state == "" {
		state = "unknown"
	}

	checks := g.checksSummary(ctx, pr.Head.SHA)
	currentSeen := fmt.Sprintf("%s|draft=%t|checks=%s", state, pr.Draft, checks)

	repoWebURL := fmt.Sprintf("https://github.com/%s", g.Repo)
	prWebURL := pr.HTMLURL
	if strings.TrimSpace(prWebURL) == "" {
		prWebURL = repoWebURL + "/pull/" + strconv.Itoa(pr.Number)
	}

	msg := fmt.Sprintf("PR #%d %s, checks=%s", pr.Number, state, checks)
	if pr.Draft {
		msg = fmt.Sprintf("PR #%d %s (draft), checks=%s", pr.Number, state, checks)
	}

	return Result{
		Current: currentSeen,
		Message: msg,
		Links: map[string]string{
			"repo": repoWebURL,
			"pr":   prWebURL,
		},
	}, nil
}

func (g githubPR) checksSummary(ctx context.Context, sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "none"
	}

	// Prefer check-runs (covers GitHub Actions). If it fails, fallback to combined status.
	checkURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/check-runs?per_page=100", g.Repo, sha)
	body, err := g.HTTP.Get(ctx, checkURL, map[string]string{
		"User-Agent": g.UserAgent,
		"Accept":     "application/vnd.github+json",
	})
	if err == nil {
		var cr githubCheckRunsResp
		if err := json.Unmarshal(body, &cr); err == nil {
			if v := summarizeCheckRuns(cr); v != "" {
				return v
			}
		}
	}

	statusURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/status", g.Repo, sha)
	body, err = g.HTTP.Get(ctx, statusURL, map[string]string{
		"User-Agent": g.UserAgent,
		"Accept":     "application/vnd.github+json",
	})
	if err != nil {
		// Don't fail the whole tracker because checks endpoint failed.
		return "unknown"
	}
	var st githubCommitStatusResp
	if err := json.Unmarshal(body, &st); err != nil {
		return "unknown"
	}

	switch strings.ToLower(strings.TrimSpace(st.State)) {
	case "success":
		return "success"
	case "failure", "error":
		return "failure"
	case "pending":
		return "pending"
	default:
		return "none"
	}
}

func summarizeCheckRuns(cr githubCheckRunsResp) string {
	if cr.TotalCount <= 0 || len(cr.CheckRuns) == 0 {
		return "none"
	}

	anyPending := false
	anyFailure := false
	anySuccess := false

	for _, r := range cr.CheckRuns {
		if strings.ToLower(strings.TrimSpace(r.Status)) != "completed" {
			anyPending = true
			continue
		}
		if r.Conclusion == nil {
			anyPending = true
			continue
		}
		c := strings.ToLower(strings.TrimSpace(*r.Conclusion))
		switch c {
		case "success", "neutral", "skipped":
			anySuccess = true
		case "failure", "cancelled", "timed_out", "action_required", "startup_failure", "stale":
			anyFailure = true
		default:
			// Treat unknown conclusion as pending so we don't spam failure.
			anyPending = true
		}
	}

	if anyPending {
		return "pending"
	}
	if anyFailure {
		return "failure"
	}
	if anySuccess {
		return "success"
	}
	return "none"
}
