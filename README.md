# upd (update-tracker)

`upd` is a small CLI **update checker** and **release tracker**.
Think: “GitHub release monitor + npm update checker + brew outdated checker”, with a clean output for cron and Discord.

It helps you monitor:
- GitHub **releases** (no API key needed, uses GitHub Atom feed)
- GitHub **commits** (uses `git ls-remote`)
- GitHub **pull requests** (PR status + checks)
- **npm** package versions
- **brew** formula versions

It can also check your **local** installed version / local git clone, compare with the latest, and print a clear report (good for cron update notifications and Discord release notifications).

## Why use this? (problem it solves)

If you have many “check update” scripts:
- hard to maintain
- different outputs
- different state files

`upd` gives you:
- one config file
- one state file
- one output format (text / json / markdown)

So you can replace many “dependency update monitor” scripts with one tool.

## Install

Requirements:
- Go 1.22+

Install from source (local clone):
```bash
go install ./cmd/upd
```

Install from GitHub:
```bash
go install github.com/peeomid/update-tracker/cmd/upd@latest
```

## Quickstart

1) Create config:
```bash
mkdir -p ~/.config/update-tracker
upd sample-config > ~/.config/update-tracker/config.yaml
```

2) Edit config:
```bash
$EDITOR ~/.config/update-tracker/config.yaml
```

3) Validate config:
```bash
upd validate-config
```

4) Run check:
```bash
upd check --format markdown --only-updates=false
```

Notes:
- Default is **quiet**: `--only-updates=true` (prints only updates/errors).
- Use `--only-updates=false` for “always show status” (good for daily Discord message).

## Example output (Markdown for Discord)

Up-to-date:
```md
NPM Package: ✅ 2026.1.24 (up-to-date)

Local Clone: ✅ 1006798 (up-to-date)
```

Update available (example):
```md
🔄 **Clawdbot Update Available!**

Current: 2026.1.24-3
Latest:  2026.2.2

(several versions behind)
```

## Output formats

`upd check --format ...` supports:
- `text`: one line per tracker
- `json`: stable schema for automation (Lobster, scripts)
- `markdown`: Discord-ready message when you use `label/group/display` in config

## Config example (GitHub release + local version, GitHub commit + local clone, npm latest + global install)

See `examples/config.yaml`.

Key ideas:
- `type: github` + `mode: release|commit|pr`
- `type: github` + `mode: pr` + `pr: 123` (PR status)
- `local:` tells `upd` how to read your local version:
  - `command`: run a command and extract version
  - `git`: read local repo HEAD
  - `npm`: read global installed package version
- `label/group/display` controls nicer Markdown output.

## Quick tracker management (no YAML editing)

Add/remove trackers:
```bash
upd track ls
upd track add --url https://github.com/openclaw/lobster/pull/123
upd track rm lobster-pr-123
```

## Lobster workflow example (Discord)

See `examples/openclaw/workflows/upd-outside-updates.yaml`.

That workflow:
- runs `upd check --format markdown --only-updates=false`
- prints the message to stdout
- OpenClaw/Clawdbot cron can send stdout to Discord

## OpenClaw setup (cron -> lobster -> discord)

Step-by-step guide:
- `docs/openclaw.md`

## Release notes (highlights)

For GitHub `mode: release`, `upd` can extract short highlights from GitHub `releases.atom`.

Flags:
- `--notes=true` (default): include highlights when `status=update`
- `--notes=false`: disable highlights

## What people use this for

- GitHub release watcher / notifier (no API key)
- Cron job that posts “up-to-date” or “update available” to Discord
- Track npm updates and compare with your global install
- Track a local git clone vs latest remote commit
- Brew outdated checks (and a single report output)

## Limitations

- No GitHub API token support (by design, simple + public endpoints).
- Highlights parsing is best-effort (HTML from Atom feed).
- No “ignore pre-release” rule yet.
