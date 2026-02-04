package config

func SampleYAML() string {
	return `version: 1
defaults:
  timeoutSeconds: 20
  retries: 1
  concurrency: 6
  userAgent: update-tracker/0.1

trackers:
  - name: clawdbot
    label: Clawdbot
    display: clawdbot
    type: github
    mode: release
    repo: anthropics/clawdbot
    local:
      type: command
      command: clawdbot --version
      # optional: regex to extract version from command output
      # regex: '[0-9]+(\\.[0-9]+){2}(-[0-9A-Za-z.-]+)?'

  - name: lobster
    label: Local Clone
    display: compare
    type: github
    mode: commit
    repo: openclaw/lobster
    branch: main
    local:
      type: git
      path: /path/to/your/lobster

  - name: ffmpeg
    type: brew
    formula: ffmpeg

  - name: npm-example
    label: NPM Package
    display: compare
    type: npm
    package: typescript
    local:
      type: npm
`
}
