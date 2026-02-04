package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/peeomid/update-tracker/internal/app"
)

func Text(r app.Report) string {
	var b strings.Builder
	for _, it := range r.Items {
		msg := it.Message
		if it.Status == "error" && strings.TrimSpace(it.Error) != "" {
			msg = it.Error
		}
		b.WriteString(fmt.Sprintf("[%s] %s - %s", it.Name, strings.ToUpper(it.Status), msg))
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Summary: ok=%d update=%d error=%d\n", r.Summary.OK, r.Summary.Update, r.Summary.Error))
	return b.String()
}

func JSON(r app.Report) (string, error) {
	type report struct {
		SchemaVersion int              `json:"schemaVersion"`
		RunAt         string           `json:"runAt"`
		Summary       app.Summary      `json:"summary"`
		Items         []app.ReportItem `json:"items"`
	}
	out := report{
		SchemaVersion: r.SchemaVersion,
		RunAt:         r.RunAt.Format(time.RFC3339),
		Summary:       r.Summary,
		Items:         r.Items,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(append(data, '\n')), nil
}

func Markdown(r app.Report) string {
	// If config is using display/local fields, prefer a clean Discord-ready message.
	for _, it := range r.Items {
		if strings.TrimSpace(it.Display) != "" || strings.TrimSpace(it.Label) != "" || strings.TrimSpace(it.Group) != "" || strings.TrimSpace(it.Local) != "" {
			return discordMarkdown(r)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Run: `%s`\n\n", r.RunAt.Format(time.RFC3339)))
	for _, it := range r.Items {
		hasHighlights := strings.TrimSpace(it.Highlights) != ""
		msg := it.Message
		if it.Status == "error" && strings.TrimSpace(it.Error) != "" {
			msg = it.Error
		}
		b.WriteString("- `")
		b.WriteString(it.Name)
		b.WriteString("` **")
		b.WriteString(strings.ToUpper(it.Status))
		b.WriteString("** - ")
		b.WriteString(msg)
		if hasHighlights {
			b.WriteString("\n")
			b.WriteString("  - Highlights:\n")
			for _, line := range strings.Split(it.Highlights, "\n") {
				line = strings.TrimRight(line, " \t")
				if line == "" {
					continue
				}
				b.WriteString("    ")
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
		if len(it.Links) > 0 {
			if hasHighlights {
				b.WriteString("  - ")
			} else {
				b.WriteString(" - ")
			}
			keys := make([]string, 0, len(it.Links))
			for k := range it.Links {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for i, k := range keys {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(fmt.Sprintf("[%s](%s)", k, it.Links[k]))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\nSummary: ok=%d update=%d error=%d\n", r.Summary.OK, r.Summary.Update, r.Summary.Error))
	return b.String()
}

func discordMarkdown(r app.Report) string {
	type group struct {
		Name  string
		Items []app.ReportItem
	}

	// Preserve order of groups as they appear.
	var groups []group
	groupIdx := map[string]int{}
	add := func(g string, it app.ReportItem) {
		key := strings.TrimSpace(g)
		if key == "" {
			key = "__default__"
		}
		if idx, ok := groupIdx[key]; ok {
			groups[idx].Items = append(groups[idx].Items, it)
			return
		}
		groupIdx[key] = len(groups)
		groups = append(groups, group{Name: key, Items: []app.ReportItem{it}})
	}
	for _, it := range r.Items {
		add(it.Group, it)
	}

	var b strings.Builder
	for gi, g := range groups {
		if gi > 0 {
			b.WriteString("\n\n")
		}

		groupHasUpdate := false
		for _, it := range g.Items {
			if it.Status == "update" {
				groupHasUpdate = true
				break
			}
		}

		// Special header for Lobster group (matches your old scripts).
		if strings.EqualFold(g.Name, "lobster") && groupHasUpdate {
			b.WriteString("🔄 **Lobster Update Available!**\n\n")
		}

		for ii, it := range g.Items {
			if ii > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(renderDiscordItem(it))
		}
	}
	return strings.TrimRight(b.String(), " \n\t") + "\n"
}

func renderDiscordItem(it app.ReportItem) string {
	label := strings.TrimSpace(it.Label)
	if label == "" {
		label = it.Name
	}

	switch strings.TrimSpace(it.Display) {
	case "clawdbot":
		return renderClawdbot(it, label)
	case "compare":
		return renderCompare(it, label)
	default:
		// fallback: one line
		msg := it.Message
		if it.Status == "error" && strings.TrimSpace(it.Error) != "" {
			msg = it.Error
		}
		return fmt.Sprintf("%s: %s", label, msg)
	}
}

func renderClawdbot(it app.ReportItem, label string) string {
	local := strings.TrimSpace(it.Local)
	if local == "" {
		local = "unknown"
	}
	latest := strings.TrimSpace(it.Latest)
	if latest == "" {
		latest = "unknown"
	}

	if it.Status == "update" {
		var b strings.Builder
		b.WriteString("🔄 **Clawdbot Update Available!**\n\n")
		b.WriteString(fmt.Sprintf("Current: %s\n", local))
		b.WriteString(fmt.Sprintf("Latest:  %s\n\n", latest))
		b.WriteString("(several versions behind)")
		if strings.TrimSpace(it.Highlights) != "" {
			b.WriteString("\n\nHighlights:\n")
			b.WriteString(strings.TrimSpace(it.Highlights))
		}
		if it.Links != nil && strings.TrimSpace(it.Links["release"]) != "" {
			b.WriteString("\n\n🔗 ")
			b.WriteString(strings.TrimSpace(it.Links["release"]))
		}
		return b.String()
	}

	if local == "unknown" || latest == "unknown" {
		return fmt.Sprintf("%s: ⚠️ %s (latest: %s)", label, local, latest)
	}
	return fmt.Sprintf("%s: ✅ %s (up-to-date)", label, local)
}

func renderCompare(it app.ReportItem, label string) string {
	local := strings.TrimSpace(it.Local)
	latest := strings.TrimSpace(it.Latest)

	if it.Status == "error" {
		if strings.TrimSpace(it.Error) != "" {
			return fmt.Sprintf("%s: ❌ %s", label, strings.TrimSpace(it.Error))
		}
		return fmt.Sprintf("%s: ❌ error", label)
	}
	if strings.TrimSpace(it.LocalError) != "" && local == "" {
		return fmt.Sprintf("%s: ⚠️ local check failed", label)
	}

	switch it.Type {
	case "npm":
		if local == "not-installed" {
			return fmt.Sprintf("%s: ❌ not installed", label)
		}
		if local != "" && latest != "" && local == latest {
			return fmt.Sprintf("%s: ✅ %s (up-to-date)", label, local)
		}
		if local != "" && latest != "" {
			return fmt.Sprintf("%s: 🔄 %s → %s", label, local, latest)
		}
		if local != "" {
			return fmt.Sprintf("%s: ⚠️ %s (latest: %s)", label, local, latest)
		}
		return fmt.Sprintf("%s: ⚠️ unknown", label)
	case "github":
		if it.Mode == "commit" {
			ls := short7(local)
			rs := short7(latest)
			if ls != "" && rs != "" && (strings.HasPrefix(latest, local) || strings.HasPrefix(local, latest) || local == latest) {
				return fmt.Sprintf("%s: ✅ %s (up-to-date)", label, ls)
			}
			if ls != "" && rs != "" {
				return fmt.Sprintf("%s: 🔄 %s → %s", label, ls, rs)
			}
			if ls != "" {
				return fmt.Sprintf("%s: ⚠️ %s (remote: %s)", label, ls, rs)
			}
			return fmt.Sprintf("%s: ⚠️ unknown", label)
		}
	}

	if local != "" && latest != "" && local == latest {
		return fmt.Sprintf("%s: ✅ %s (up-to-date)", label, local)
	}
	if local != "" && latest != "" {
		return fmt.Sprintf("%s: 🔄 %s → %s", label, local, latest)
	}
	if local != "" {
		return fmt.Sprintf("%s: ⚠️ %s (latest: %s)", label, local, latest)
	}
	return fmt.Sprintf("%s: ⚠️ unknown", label)
}

func short7(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}
