package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/peeomid/update-tracker/internal/config"
	"github.com/peeomid/update-tracker/internal/state"
	"github.com/peeomid/update-tracker/internal/trackers"
)

type runner struct {
	Registry    trackers.Registry
	Timeout     time.Duration
	Retries     int
	Concurrency int
	RunAt       time.Time
	Options     Options
}

func (r runner) Run(ctx context.Context, trackerCfgs []config.TrackerEntry, st state.State) ([]ReportItem, state.State) {
	if st.Items == nil {
		st.Items = map[string]state.Item{}
	}

	type job struct {
		Idx int
		Cfg config.TrackerEntry
	}

	results := make([]ReportItem, len(trackerCfgs))
	prevItems := make(map[string]state.Item, len(st.Items))
	for k, v := range st.Items {
		prevItems[k] = v
	}
	nextState := state.State{Items: make(map[string]state.Item, len(prevItems))}
	for k, v := range prevItems {
		nextState.Items[k] = v
	}

	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex

	workerCount := r.Concurrency
	if workerCount > len(trackerCfgs) {
		workerCount = len(trackerCfgs)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				res, stItem := r.runOne(ctx, j.Cfg, prevItems[j.Cfg.Name])
				results[j.Idx] = res
				mu.Lock()
				nextState.Items[j.Cfg.Name] = stItem
				mu.Unlock()
			}
		}()
	}

	for idx, cfg := range trackerCfgs {
		jobs <- job{Idx: idx, Cfg: cfg}
	}
	close(jobs)
	wg.Wait()

	return results, nextState
}

func (r runner) runOne(ctx context.Context, cfg config.TrackerEntry, prev state.Item) (ReportItem, state.Item) {
	tr, err := r.Registry.Build(cfg)
	if err != nil {
		return errorItem(cfg, err.Error()), state.Item{
			LastCheckedAt: r.RunAt,
			LastSeen:      prev.LastSeen,
			LastStatus:    "error",
			LastError:     err.Error(),
		}
	}

	var (
		current    string
		latest     string
		local      string
		message    string
		links      map[string]string
		highlights string
		localErr   string
	)

	var lastErr error
	for attempt := 0; attempt <= r.Retries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		res, err := tr.Check(attemptCtx, prev.LastSeen, trackers.Options{IncludeNotes: r.Options.IncludeNotes})
		current, message, links, highlights = res.Current, res.Message, res.Links, res.Highlights
		latest = normalizeLatest(cfg, current)
		lastErr = err
		cancel()
		if lastErr == nil {
			break
		}
		if !isRetryable(lastErr) {
			break
		}
	}

	if lastErr != nil {
		res := ReportItem{
			Name:    cfg.Name,
			Type:    cfg.Type,
			Mode:    cfg.Mode,
			Label:   cfg.Label,
			Group:   cfg.Group,
			Display: cfg.Display,
			Status:  "error",
			Message: "ERROR",
			Links:   links,
			Error:   lastErr.Error(),
		}
		return res, state.Item{
			LastCheckedAt: r.RunAt,
			LastSeen:      prev.LastSeen,
			LastStatus:    "error",
			LastError:     lastErr.Error(),
		}
	}

	prevSeen := strings.TrimSpace(prev.LastSeen)
	currSeen := strings.TrimSpace(current)
	local, localErr = runLocalCheck(ctx, r, cfg)
	local = strings.TrimSpace(local)

	status := "ok"
	remoteChanged := prevSeen != "" && currSeen != "" && prevSeen != currSeen
	localChanged := localUpdateAvailable(cfg, local, latest)
	if remoteChanged || localChanged {
		status = "update"
	}
	if strings.TrimSpace(localErr) != "" && strings.TrimSpace(cfg.Local.Type) != "" {
		// local check failed, but remote might still be ok. Keep the run "ok", but surface localError for output.
	}

	// If local is behind (update), but this is not a "new remote release" for state,
	// we still want highlights for the latest release (same style as your old script).
	if status == "update" &&
		!remoteChanged &&
		cfg.Type == "github" &&
		cfg.Mode == "release" &&
		strings.TrimSpace(highlights) == "" &&
		r.Options.IncludeNotes {
		attemptCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		res2, err2 := tr.Check(attemptCtx, "force-notes", trackers.Options{IncludeNotes: true})
		cancel()
		if err2 == nil && strings.TrimSpace(res2.Highlights) != "" {
			highlights = res2.Highlights
		}
	}

	res := ReportItem{
		Name:       cfg.Name,
		Type:       cfg.Type,
		Mode:       cfg.Mode,
		Label:      cfg.Label,
		Group:      cfg.Group,
		Display:    cfg.Display,
		Status:     status,
		Prev:       prevSeen,
		Current:    currSeen,
		Latest:     strings.TrimSpace(latest),
		Local:      local,
		Message:    message,
		Links:      links,
		Highlights: highlights,
		LocalError: strings.TrimSpace(localErr),
	}
	return res, state.Item{
		LastCheckedAt: r.RunAt,
		LastSeen:      currSeen,
		LastStatus:    status,
		LastError:     "",
	}
}

func errorItem(cfg config.TrackerEntry, msg string) ReportItem {
	return ReportItem{
		Name:    cfg.Name,
		Type:    cfg.Type,
		Mode:    cfg.Mode,
		Label:   cfg.Label,
		Group:   cfg.Group,
		Display: cfg.Display,
		Status:  "error",
		Message: "ERROR",
		Error:   msg,
	}
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "tls") ||
		strings.Contains(msg, "temporary") {
		return true
	}
	return false
}

var _ = fmt.Sprintf

var versionRe = regexp.MustCompile(`[0-9]+(\.[0-9]+){2}(-[0-9A-Za-z.-]+)?`)

func normalizeLatest(cfg config.TrackerEntry, current string) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return ""
	}
	if cfg.Type == "github" && cfg.Mode == "release" {
		if m := versionRe.FindString(current); m != "" {
			return m
		}
	}
	return current
}

func runLocalCheck(ctx context.Context, r runner, cfg config.TrackerEntry) (string, string) {
	switch strings.TrimSpace(cfg.Local.Type) {
	case "":
		return "", ""
	case "command":
		// Use zsh -lc so the user's shell init is loaded (PATH, nvm, etc).
		attemptCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		out, err := r.Registry.Exec.Run(attemptCtx, "zsh", "-lc", cfg.Local.Command)
		cancel()
		if err != nil {
			return "", err.Error()
		}
		out = strings.TrimSpace(out)
		if out == "" {
			return "unknown", ""
		}
		if strings.TrimSpace(cfg.Local.Regex) == "" {
			if m := versionRe.FindString(out); m != "" {
				return m, ""
			}
			return "unknown", ""
		}
		re, err := regexp.Compile(cfg.Local.Regex)
		if err != nil {
			return "", "invalid local.regex: " + err.Error()
		}
		if m := re.FindString(out); m != "" {
			return m, ""
		}
		return "unknown", ""
	case "git":
		attemptCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		out, err := r.Registry.Exec.Run(attemptCtx, "git", "-C", cfg.Local.Path, "rev-parse", "HEAD")
		cancel()
		if err != nil {
			return "", err.Error()
		}
		out = strings.TrimSpace(out)
		if out == "" {
			return "unknown", ""
		}
		return out, ""
	case "npm":
		pkg := cfg.NpmPackage
		if strings.TrimSpace(cfg.Local.Package) != "" {
			pkg = cfg.Local.Package
		}
		if strings.TrimSpace(pkg) == "" {
			return "", "missing npm package"
		}
		attemptCtx, cancel := context.WithTimeout(ctx, r.Timeout)
		out, err := r.Registry.Exec.Run(attemptCtx, "npm", "list", pkg, "--depth=0", "-g", "--json")
		cancel()

		// npm returns non-zero for missing packages, but still prints JSON. Parse stdout first.
		if v := parseNpmListJSON(out, pkg); v != "" {
			return v, ""
		}
		if err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "missing") || strings.Contains(msg, "not installed") || strings.Contains(msg, "empty") {
				return "not-installed", ""
			}
			return "", err.Error()
		}
		return "unknown", ""
	default:
		return "", "unknown local.type: " + cfg.Local.Type
	}
}

func localUpdateAvailable(cfg config.TrackerEntry, local string, latest string) bool {
	local = strings.TrimSpace(local)
	latest = strings.TrimSpace(latest)
	if strings.TrimSpace(cfg.Local.Type) == "" {
		return false
	}
	if local == "" || local == "unknown" {
		return false
	}
	if latest == "" {
		return false
	}
	switch cfg.Type {
	case "github":
		if cfg.Mode == "commit" {
			// Allow comparing short to full SHA.
			if strings.HasPrefix(latest, local) || strings.HasPrefix(local, latest) {
				return false
			}
			return local != latest
		}
		return local != latest
	case "npm":
		if local == "not-installed" {
			return true
		}
		return local != latest
	default:
		return local != latest
	}
}

func parseNpmListJSON(out string, pkg string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return ""
	}
	type npmDep struct {
		Version string `json:"version"`
	}
	type npmList struct {
		Dependencies map[string]npmDep `json:"dependencies"`
	}

	var parsed npmList
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return ""
	}
	if parsed.Dependencies == nil {
		return ""
	}
	dep, ok := parsed.Dependencies[pkg]
	if !ok {
		return ""
	}
	return strings.TrimSpace(dep.Version)
}
