package trackers

import (
	"html"
	"regexp"
	"strings"
)

func shortSHA(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}

func extractHighlightsFromHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	liRe := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)

	candidates := raw
	if idx := strings.Index(strings.ToLower(raw), "highlights"); idx >= 0 {
		candidates = raw[idx:]
	}

	matches := liRe.FindAllStringSubmatch(candidates, -1)
	if len(matches) == 0 && candidates != raw {
		matches = liRe.FindAllStringSubmatch(raw, -1)
	}

	var lines []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		txt := m[1]
		txt = tagRe.ReplaceAllString(txt, "")
		txt = html.UnescapeString(txt)
		txt = strings.TrimSpace(txt)
		txt = strings.Join(strings.Fields(txt), " ")
		if txt == "" {
			continue
		}
		lines = append(lines, "- "+txt)
		if len(lines) >= 6 {
			break
		}
	}

	out := strings.Join(lines, "\n")
	if len(out) > 500 {
		out = out[:500] + "..."
	}
	return out
}
