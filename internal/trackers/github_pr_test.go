package trackers

import (
	"context"
	"testing"
)

type mapFetcher struct {
	ByURL map[string][]byte
	Err   error
}

func (m mapFetcher) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if b, ok := m.ByURL[url]; ok {
		return b, nil
	}
	return nil, nil
}

func TestGitHubPRCurrentSeenIgnoresHeadSHA(t *testing.T) {
	prJSON := `{
  "number": 123,
  "state": "open",
  "draft": false,
  "merged": false,
  "html_url": "https://github.com/a/b/pull/123",
  "head": { "sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }
}`

	checkRuns := `{
  "total_count": 1,
  "check_runs": [
    { "status": "completed", "conclusion": "success" }
  ]
}`

	f := mapFetcher{
		ByURL: map[string][]byte{
			"https://api.github.com/repos/a/b/pulls/123":                                                                []byte(prJSON),
			"https://api.github.com/repos/a/b/commits/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/check-runs?per_page=100": []byte(checkRuns),
		},
	}

	tr := githubPR{
		HTTP:      f,
		UserAgent: "x",
		Repo:      "a/b",
		PR:        123,
	}

	res, err := tr.Check(context.Background(), "anything", Options{})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if res.Current != "open|draft=false|checks=success" {
		t.Fatalf("current=%q", res.Current)
	}
	if res.Links["pr"] == "" {
		t.Fatalf("missing pr link")
	}
}
