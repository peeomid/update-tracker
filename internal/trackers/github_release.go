package trackers

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/peeomid/update-tracker/internal/execx"
	"github.com/peeomid/update-tracker/internal/httpx"
)

type githubReleaseOrCommit struct {
	HTTP      httpx.Fetcher
	Exec      execx.Runner
	UserAgent string
	Repo      string
	Fallback  githubCommit
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID      string `xml:"id"`
	Title   string `xml:"title"`
	Content struct {
		Type string `xml:"type,attr"`
		Body string `xml:",chardata"`
	} `xml:"content"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
}

func (g githubReleaseOrCommit) Check(ctx context.Context, prevSeen string, opts Options) (Result, error) {
	repoURL := fmt.Sprintf("https://github.com/%s", g.Repo)
	feedURL := fmt.Sprintf("%s/releases.atom", repoURL)

	body, err := g.HTTP.Get(ctx, feedURL, map[string]string{
		"User-Agent": g.UserAgent,
	})
	if err != nil {
		return Result{}, fmt.Errorf("fetch atom: %w", err)
	}

	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return Result{}, fmt.Errorf("parse atom: %w", err)
	}

	if len(feed.Entries) == 0 {
		fb, err := g.Fallback.Check(ctx, prevSeen, opts)
		if err != nil {
			return Result{}, fmt.Errorf("no releases; fallback commit failed: %w", err)
		}
		if fb.Links == nil {
			fb.Links = map[string]string{}
		}
		fb.Links["feed"] = feedURL
		fb.Message = "no releases; " + fb.Message
		return fb, nil
	}

	entry := feed.Entries[0]
	title := strings.TrimSpace(entry.Title)
	if title == "" {
		title = strings.TrimSpace(entry.ID)
	}
	if title == "" {
		return Result{}, fmt.Errorf("atom: missing entry title/id")
	}

	releaseLink := ""
	for _, l := range entry.Links {
		if l.Rel == "" || l.Rel == "alternate" {
			releaseLink = l.Href
			break
		}
	}
	if releaseLink == "" && len(entry.Links) > 0 {
		releaseLink = entry.Links[0].Href
	}

	links := map[string]string{"repo": repoURL, "feed": feedURL}
	if releaseLink != "" {
		links["release"] = releaseLink
	}

	msg := fmt.Sprintf("latest release %s", title)
	prev := strings.TrimSpace(prevSeen)
	highlights := ""
	if prev != "" && prev != title {
		msg = fmt.Sprintf("new release %s", title)
		if opts.IncludeNotes {
			highlights = extractHighlightsFromHTML(entry.Content.Body)
		}
	}
	return Result{
		Current:    title,
		Message:    msg,
		Links:      links,
		Highlights: highlights,
	}, nil
}
