package config

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidate_OK(t *testing.T) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader([]byte(SampleYAML())))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	cfg := Config{
		Version: 1,
		Defaults: Defaults{
			TimeoutSeconds: 10,
			Retries:        0,
			Concurrency:    1,
			UserAgent:      "x",
		},
		Trackers: []TrackerEntry{
			{Name: "x", Type: "github", Mode: "commit", Repo: "a/b"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}
