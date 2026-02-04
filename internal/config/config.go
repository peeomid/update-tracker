package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  int            `yaml:"version"`
	Defaults Defaults       `yaml:"defaults"`
	Trackers []TrackerEntry `yaml:"trackers"`
}

type Defaults struct {
	TimeoutSeconds int    `yaml:"timeoutSeconds"`
	Retries        int    `yaml:"retries"`
	Concurrency    int    `yaml:"concurrency"`
	UserAgent      string `yaml:"userAgent"`
}

type TrackerEntry struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`

	// output (optional)
	Label   string `yaml:"label"`
	Group   string `yaml:"group"`
	Display string `yaml:"display"`

	// github
	Mode   string `yaml:"mode"`
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`

	// brew
	Formula string `yaml:"formula"`

	// npm
	NpmPackage string `yaml:"package"`

	// local checks (optional)
	Local LocalEntry `yaml:"local"`
}

type LocalEntry struct {
	// one of: command|git|npm
	Type string `yaml:"type"`

	// command
	Command string `yaml:"command"`
	Regex   string `yaml:"regex"`

	// git
	Path string `yaml:"path"`

	// npm
	Package string `yaml:"package"`
}

func ResolvePath(p string) string {
	if strings.TrimSpace(p) != "" {
		return p
	}
	return DefaultConfigPath()
}

func DefaultConfigPath() string {
	dir := DefaultConfigDir()
	return filepath.Join(dir, "config.yaml")
}

func DefaultStatePath() string {
	dir := DefaultConfigDir()
	return filepath.Join(dir, "state.json")
}

func DefaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "update-tracker")
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Defaults.TimeoutSeconds == 0 {
		c.Defaults.TimeoutSeconds = 20
	}
	if c.Defaults.Retries == 0 {
		c.Defaults.Retries = 1
	}
	if c.Defaults.Concurrency == 0 {
		c.Defaults.Concurrency = 6
	}
	if strings.TrimSpace(c.Defaults.UserAgent) == "" {
		c.Defaults.UserAgent = "update-tracker/0.1"
	}
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("config: version must be 1")
	}
	if len(c.Trackers) == 0 {
		return fmt.Errorf("config: trackers must not be empty")
	}
	if c.Defaults.TimeoutSeconds <= 0 {
		return fmt.Errorf("config: defaults.timeoutSeconds must be > 0")
	}
	if c.Defaults.Retries < 0 {
		return fmt.Errorf("config: defaults.retries must be >= 0")
	}
	if c.Defaults.Concurrency <= 0 {
		return fmt.Errorf("config: defaults.concurrency must be > 0")
	}

	seenNames := map[string]bool{}
	for i, t := range c.Trackers {
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("config: trackers[%d].name is required", i)
		}
		if seenNames[t.Name] {
			return fmt.Errorf("config: duplicate tracker name: %s", t.Name)
		}
		seenNames[t.Name] = true

		if strings.TrimSpace(t.Display) != "" {
			switch t.Display {
			case "clawdbot", "compare":
			default:
				return fmt.Errorf("config: trackers[%d].display must be clawdbot|compare (or empty)", i)
			}
		}

		switch t.Type {
		case "github":
			if strings.TrimSpace(t.Repo) == "" {
				return fmt.Errorf("config: trackers[%d].repo is required (github)", i)
			}
			if t.Mode != "release" && t.Mode != "commit" {
				return fmt.Errorf("config: trackers[%d].mode must be release|commit (github)", i)
			}
				if t.Mode == "commit" && strings.TrimSpace(t.Branch) == "" {
					return fmt.Errorf("config: trackers[%d].branch is required (github commit)", i)
				}
				if strings.TrimSpace(t.Formula) != "" || strings.TrimSpace(t.NpmPackage) != "" {
					return fmt.Errorf("config: trackers[%d] has fields not allowed for github", i)
				}

				if strings.TrimSpace(t.Local.Type) != "" {
					switch t.Mode {
					case "release":
						if t.Local.Type != "command" {
							return fmt.Errorf("config: trackers[%d].local.type must be command (github release)", i)
						}
						if strings.TrimSpace(t.Local.Command) == "" {
							return fmt.Errorf("config: trackers[%d].local.command is required (github release)", i)
						}
					case "commit":
						if t.Local.Type != "git" {
							return fmt.Errorf("config: trackers[%d].local.type must be git (github commit)", i)
						}
						if strings.TrimSpace(t.Local.Path) == "" {
							return fmt.Errorf("config: trackers[%d].local.path is required (github commit)", i)
						}
					}
				}
			case "brew":
				if strings.TrimSpace(t.Formula) == "" {
					return fmt.Errorf("config: trackers[%d].formula is required (brew)", i)
				}
			if strings.TrimSpace(t.Mode) != "" {
				return fmt.Errorf("config: trackers[%d].mode not allowed for type brew", i)
			}
				if strings.TrimSpace(t.Repo) != "" || strings.TrimSpace(t.Branch) != "" || strings.TrimSpace(t.NpmPackage) != "" {
					return fmt.Errorf("config: trackers[%d] has fields not allowed for brew", i)
				}
				if strings.TrimSpace(t.Local.Type) != "" {
					return fmt.Errorf("config: trackers[%d].local not supported for brew", i)
				}
			case "npm":
				if strings.TrimSpace(t.NpmPackage) == "" {
					return fmt.Errorf("config: trackers[%d].package is required (npm)", i)
				}
			if strings.TrimSpace(t.Mode) != "" {
				return fmt.Errorf("config: trackers[%d].mode not allowed for type npm", i)
			}
				if strings.TrimSpace(t.Repo) != "" || strings.TrimSpace(t.Branch) != "" || strings.TrimSpace(t.Formula) != "" {
					return fmt.Errorf("config: trackers[%d] has fields not allowed for npm", i)
				}
				if strings.TrimSpace(t.Local.Type) != "" {
					if t.Local.Type != "npm" {
						return fmt.Errorf("config: trackers[%d].local.type must be npm (npm)", i)
					}
					if strings.TrimSpace(t.Local.Package) != "" && strings.TrimSpace(t.Local.Package) != strings.TrimSpace(t.NpmPackage) {
						// allowed, but must be explicit and non-empty; keep it validated (no extra rule)
					}
				}
			default:
				return fmt.Errorf("config: trackers[%d].type must be github|brew|npm", i)
			}

		// validate local fields (no extra keys)
		if strings.TrimSpace(t.Local.Type) != "" {
			switch t.Local.Type {
			case "command":
				if strings.TrimSpace(t.Local.Command) == "" {
					return fmt.Errorf("config: trackers[%d].local.command is required (command)", i)
				}
				if strings.TrimSpace(t.Local.Path) != "" || strings.TrimSpace(t.Local.Package) != "" {
					return fmt.Errorf("config: trackers[%d].local has fields not allowed for command", i)
				}
			case "git":
				if strings.TrimSpace(t.Local.Path) == "" {
					return fmt.Errorf("config: trackers[%d].local.path is required (git)", i)
				}
				if strings.TrimSpace(t.Local.Command) != "" || strings.TrimSpace(t.Local.Regex) != "" || strings.TrimSpace(t.Local.Package) != "" {
					return fmt.Errorf("config: trackers[%d].local has fields not allowed for git", i)
				}
			case "npm":
				if strings.TrimSpace(t.Local.Command) != "" || strings.TrimSpace(t.Local.Regex) != "" || strings.TrimSpace(t.Local.Path) != "" {
					return fmt.Errorf("config: trackers[%d].local has fields not allowed for npm", i)
				}
			default:
				return fmt.Errorf("config: trackers[%d].local.type must be command|git|npm", i)
			}
		}
	}

	return nil
}
