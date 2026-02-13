package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/peeomid/update-tracker/internal/config"
)

func runTrack(args []string) int {
	if len(args) == 0 {
		usageTrack(os.Stderr)
		return 2
	}
	switch args[0] {
	case "ls":
		return runTrackLS(args[1:])
	case "add":
		return runTrackAdd(args[1:])
	case "rm":
		return runTrackRM(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown track command: %s\n\n", args[0])
		usageTrack(os.Stderr)
		return 2
	}
}

func usageTrack(w *os.File) {
	fmt.Fprintln(w, "upd track")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Quickly manage trackers in your config file.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  upd track ls [--config PATH]")
	fmt.Fprintln(w, "  upd track add --url URL [--config PATH] [--name NAME] [--label LABEL] [--group GROUP] [--display DISPLAY]")
	fmt.Fprintln(w, "  upd track rm NAME [--config PATH]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "URL examples:")
	fmt.Fprintln(w, "  https://github.com/OWNER/REPO")
	fmt.Fprintln(w, "  https://github.com/OWNER/REPO/pull/123")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Tip: validate after changes:")
	fmt.Fprintln(w, "  upd validate-config")
}

func runTrackLS(args []string) int {
	fs := flag.NewFlagSet("track ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageTrack(os.Stdout) }
	configPath := fs.String("config", "", "config path (default: ~/.config/update-tracker/config.yaml)")
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	path := config.ResolvePath(*configPath)
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	for _, t := range cfg.Trackers {
		desc := t.Type
		if t.Type == "github" && strings.TrimSpace(t.Mode) != "" {
			desc = desc + ":" + t.Mode
		}
		if t.Type == "github" && t.Mode == "pr" {
			desc = desc + " #" + strconv.Itoa(t.PR)
		}
		fmt.Printf("%s\t%s\n", t.Name, desc)
	}
	return 0
}

func runTrackAdd(args []string) int {
	fs := flag.NewFlagSet("track add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageTrack(os.Stdout) }
	configPath := fs.String("config", "", "config path (default: ~/.config/update-tracker/config.yaml)")
	rawURL := fs.String("url", "", "github url (repo or pull request)")
	name := fs.String("name", "", "tracker name (optional)")
	label := fs.String("label", "", "output label (optional)")
	group := fs.String("group", "", "output group (optional)")
	display := fs.String("display", "", "display mode (optional)")
	mode := fs.String("mode", "", "github mode for repo url: release|commit (optional; default: release)")
	branch := fs.String("branch", "", "github branch (only for mode=commit; optional)")
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if strings.TrimSpace(*rawURL) == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 2
	}

	kind, repo, prNum, err := parseGitHubURL(*rawURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	path := config.ResolvePath(*configPath)
	cfg, err := loadOrNewConfig(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	entry, err := buildTrackerFromURL(kind, repo, prNum, *mode, *branch)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if strings.TrimSpace(*name) != "" {
		entry.Name = strings.TrimSpace(*name)
	} else {
		entry.Name = uniqueName(cfg, entry.Name)
	}
	entry.Label = strings.TrimSpace(*label)
	entry.Group = strings.TrimSpace(*group)
	entry.Display = strings.TrimSpace(*display)

	cfg.Trackers = append(cfg.Trackers, entry)
	if err := cfg.Validate(); err != nil {
		// Don't write anything if invalid.
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if err := backupFile(path); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	return 0
}

func runTrackRM(args []string) int {
	fs := flag.NewFlagSet("track rm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { usageTrack(os.Stdout) }
	configPath := fs.String("config", "", "config path (default: ~/.config/update-tracker/config.yaml)")
	if err := fs.Parse(args); err != nil {
		if helpRequested(err) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing NAME")
		return 2
	}
	target := strings.TrimSpace(fs.Arg(0))
	if target == "" {
		fmt.Fprintln(os.Stderr, "missing NAME")
		return 2
	}

	path := config.ResolvePath(*configPath)
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	kept := cfg.Trackers[:0]
	found := false
	for _, t := range cfg.Trackers {
		if t.Name == target {
			found = true
			continue
		}
		kept = append(kept, t)
	}
	cfg.Trackers = kept
	if !found {
		fmt.Fprintln(os.Stderr, "tracker not found:", target)
		return 2
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if err := backupFile(path); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	return 0
}

func loadOrNewConfig(path string) (config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return config.Config{
			Version: 1,
			Defaults: config.Defaults{
				TimeoutSeconds: 20,
				Retries:        1,
				Concurrency:    6,
				UserAgent:      "update-tracker/0.1",
			},
			Trackers: nil,
		}, nil
	}
	return config.Config{}, err
}

func parseGitHubURL(raw string) (kind string, repo string, pr int, err error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid url: %w", err)
	}
	if !strings.EqualFold(u.Host, "github.com") {
		return "", "", 0, fmt.Errorf("only github.com supported (got host=%s)", u.Host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", 0, fmt.Errorf("invalid github url path")
	}
	repo = parts[0] + "/" + parts[1]
	if len(parts) >= 4 && parts[2] == "pull" {
		n, err := strconv.Atoi(parts[3])
		if err != nil || n <= 0 {
			return "", "", 0, fmt.Errorf("invalid pull request number")
		}
		return "pr", repo, n, nil
	}
	return "repo", repo, 0, nil
}

func buildTrackerFromURL(kind string, repo string, prNum int, mode string, branch string) (config.TrackerEntry, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return config.TrackerEntry{}, fmt.Errorf("missing repo")
	}

	switch kind {
	case "pr":
		if strings.TrimSpace(mode) != "" && strings.TrimSpace(mode) != "pr" {
			return config.TrackerEntry{}, fmt.Errorf("--mode is not allowed for pull request url")
		}
		name := strings.ReplaceAll(repo, "/", "-") + "-pr-" + strconv.Itoa(prNum)
		return config.TrackerEntry{
			Name: name,
			Type: "github",
			Mode: "pr",
			Repo: repo,
			PR:   prNum,
		}, nil
	case "repo":
		m := strings.TrimSpace(mode)
		if m == "" {
			m = "release"
		}
		if m != "release" && m != "commit" {
			return config.TrackerEntry{}, fmt.Errorf("--mode must be release|commit")
		}
		name := strings.ReplaceAll(repo, "/", "-") + "-" + m
		e := config.TrackerEntry{
			Name: name,
			Type: "github",
			Mode: m,
			Repo: repo,
		}
		if m == "commit" {
			b := strings.TrimSpace(branch)
			if b == "" {
				b = "main"
			}
			e.Branch = b
		}
		return e, nil
	default:
		return config.TrackerEntry{}, fmt.Errorf("unsupported url kind")
	}
}

func uniqueName(cfg config.Config, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "tracker"
	}
	seen := map[string]bool{}
	for _, t := range cfg.Trackers {
		seen[t.Name] = true
	}
	if !seen[base] {
		return base
	}
	for i := 2; i < 1000; i++ {
		n := base + "-" + strconv.Itoa(i)
		if !seen[n] {
			return n
		}
	}
	return base + "-" + strconv.FormatInt(time.Now().Unix(), 10)
}

func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	ts := time.Now().Format("20060102-150405")
	backup := path + ".bak-" + ts

	if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
		return err
	}
	return os.WriteFile(backup, data, 0o644)
}

var _ = filepath.Separator
