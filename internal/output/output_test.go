package output

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/peeomid/update-tracker/internal/app"
)

func TestJSONSchemaShape(t *testing.T) {
	r := app.Report{
		SchemaVersion: 1,
		RunAt:         time.Date(2026, 2, 3, 1, 2, 3, 0, time.UTC),
		Summary:       app.Summary{OK: 1, Update: 2, Error: 3},
		Items: []app.ReportItem{
			{
				Name:    "x",
				Type:    "github",
				Mode:    "commit",
				Status:  "ok",
				Prev:    "a",
				Current: "b",
				Message: "m",
				Links:   map[string]string{"repo": "https://example.com"},
			},
		},
	}

	s, err := JSON(r)
	if err != nil {
		t.Fatalf("json: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["schemaVersion"]; !ok {
		t.Fatalf("missing schemaVersion")
	}
	if _, ok := m["runAt"]; !ok {
		t.Fatalf("missing runAt")
	}
	if _, ok := m["summary"]; !ok {
		t.Fatalf("missing summary")
	}
	if _, ok := m["items"]; !ok {
		t.Fatalf("missing items")
	}
}

func TestMarkdown_DiscordStyle_CompareAndClawdbot(t *testing.T) {
	r := app.Report{
		SchemaVersion: 1,
		RunAt:         time.Date(2026, 2, 4, 1, 2, 3, 0, time.UTC),
		Summary:       app.Summary{OK: 0, Update: 1, Error: 0},
		Items: []app.ReportItem{
			{
				Name:       "clawdbot-release",
				Label:      "Clawdbot",
				Group:      "clawdbot",
				Display:    "clawdbot",
				Type:       "github",
				Mode:       "release",
				Status:     "update",
				Latest:     "2026.2.2",
				Local:      "2026.1.24-3",
				Highlights: "- A\n- B",
				Links:      map[string]string{"release": "https://example.com/release"},
				Message:    "ignored",
			},
			{
				Name:    "lobster-npm",
				Label:   "NPM Package",
				Group:   "lobster",
				Display: "compare",
				Type:    "npm",
				Status:  "ok",
				Local:   "2026.1.24",
				Latest:  "2026.1.24",
				Message: "ignored",
			},
			{
				Name:    "lobster",
				Label:   "Local Clone",
				Group:   "lobster",
				Display: "compare",
				Type:    "github",
				Mode:    "commit",
				Status:  "ok",
				Local:   "1006798459a17e11903137ce09198e64686a4dbb",
				Latest:  "1006798459a17e11903137ce09198e64686a4dbb",
				Message: "ignored",
			},
		},
	}

	got := Markdown(r)
	want := "🔄 **Clawdbot Update Available!**\n\nCurrent: 2026.1.24-3\nLatest:  2026.2.2\n\n(several versions behind)\n\nHighlights:\n- A\n- B\n\n🔗 https://example.com/release\n\nNPM Package: ✅ 2026.1.24 (up-to-date)\n\nLocal Clone: ✅ 1006798 (up-to-date)\n"
	if got != want {
		t.Fatalf("markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestMarkdown_DiscordStyle_LobsterHeaderOnUpdate(t *testing.T) {
	r := app.Report{
		SchemaVersion: 1,
		RunAt:         time.Date(2026, 2, 4, 1, 2, 3, 0, time.UTC),
		Summary:       app.Summary{OK: 0, Update: 1, Error: 0},
		Items: []app.ReportItem{
			{
				Name:    "lobster",
				Label:   "Local Clone",
				Group:   "lobster",
				Display: "compare",
				Type:    "github",
				Mode:    "commit",
				Status:  "update",
				Local:   "aaaaaaa1111111",
				Latest:  "bbbbbbb2222222",
				Message: "ignored",
			},
		},
	}
	got := Markdown(r)
	if len(got) < len("🔄 **Lobster Update Available!**") || got[:len("🔄 **Lobster Update Available!**")] != "🔄 **Lobster Update Available!**" {
		t.Fatalf("expected lobster header, got: %q", got)
	}
}
