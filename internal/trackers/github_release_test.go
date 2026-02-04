package trackers

import (
	"context"
	"testing"

	"github.com/peeomid/update-tracker/internal/execx"
	"github.com/peeomid/update-tracker/internal/httpx"
)

type fakeFetcher struct {
	Body []byte
	Err  error
}

func (f fakeFetcher) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return f.Body, f.Err
}

type fakeRunner struct {
	Out string
	Err error
}

func (f fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return f.Out, f.Err
}

func TestGitHubAtomParsing(t *testing.T) {
	atom := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <id>tag:github.com,2008:Repository/1/v1.2.3</id>
    <title>v1.2.3</title>
    <content type="html"><![CDATA[<h3>Highlights</h3><ul><li>Fix A</li><li>Add B</li></ul>]]></content>
    <link rel="alternate" href="https://github.com/a/b/releases/tag/v1.2.3"/>
  </entry>
</feed>`

	tr := githubReleaseOrCommit{
		HTTP:      fakeFetcher{Body: []byte(atom)},
		Exec:      execx.OSRunner{},
		UserAgent: "x",
		Repo:      "a/b",
		Fallback: githubCommit{
			Exec:   fakeRunner{Out: "abc123\trefs/heads/main"},
			Repo:   "a/b",
			Branch: "main",
		},
	}

	res, err := tr.Check(context.Background(), "v1.0.0", Options{IncludeNotes: true})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if res.Current != "v1.2.3" {
		t.Fatalf("current=%q", res.Current)
	}
	if res.Message == "" {
		t.Fatalf("msg empty")
	}
	if res.Links["release"] == "" {
		t.Fatalf("missing release link")
	}
	if res.Highlights == "" {
		t.Fatalf("missing highlights")
	}
}

var _ httpx.Fetcher = fakeFetcher{}
