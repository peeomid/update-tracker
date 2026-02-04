# OpenClaw / Clawdbot setup (cron -> lobster -> discord)

Goal: run `upd` on a schedule and send the message to Discord.

## 1) Install upd

On the machine that runs the cron job:
```bash
go install github.com/peeomid/update-tracker/cmd/upd@latest
```

Make sure `upd` is on PATH (common path: `~/go/bin`).

## 2) Create upd config

Config file:
- `~/.config/update-tracker/config.yaml`

Start from:
- `upd sample-config`
- or copy `examples/config.yaml`

Validate:
```bash
upd validate-config
```

## 3) Add Lobster workflow

Copy this file somewhere in your OpenClaw repo:
- `examples/openclaw/workflows/upd-outside-updates.yaml`

Test locally:
```bash
lobster run path/to/upd-outside-updates.yaml
```

## 4) Create Clawdbot cron job

Create a cron job that calls the Lobster tool (workflow runner) and delivers stdout to Discord.

Notes:
- Use a `lobsterPath` that points to the *executable*, not a broken symlink.
- Output should be ONLY the content from lobster’s output array.

