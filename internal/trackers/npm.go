package trackers

import (
	"context"
	"fmt"
	"strings"

	"github.com/peeomid/update-tracker/internal/execx"
)

type npmPackage struct {
	Exec    execx.Runner
	Package string
}

func (n npmPackage) Check(ctx context.Context, prevSeen string, opts Options) (Result, error) {
	_ = opts
	out, err := n.Exec.Run(ctx, "npm", "view", n.Package, "version")
	if err != nil {
		return Result{}, fmt.Errorf("npm view: %w", err)
	}
	version := strings.TrimSpace(out)
	if version == "" {
		return Result{}, fmt.Errorf("npm view: empty version")
	}
	links := map[string]string{"npm": fmt.Sprintf("https://www.npmjs.com/package/%s", n.Package)}
	msg := fmt.Sprintf("latest %s", version)
	prev := strings.TrimSpace(prevSeen)
	if prev != "" && prev != version {
		msg = fmt.Sprintf("new version %s", version)
	}
	return Result{
		Current: version,
		Message: msg,
		Links:   links,
	}, nil
}
