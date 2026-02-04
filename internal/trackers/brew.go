package trackers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/peeomid/update-tracker/internal/execx"
)

type brewFormula struct {
	Exec    execx.Runner
	Formula string
}

type brewInfoV2 struct {
	Formulae []struct {
		Name     string `json:"name"`
		Homepage string `json:"homepage"`
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	} `json:"formulae"`
}

func (b brewFormula) Check(ctx context.Context, prevSeen string, opts Options) (Result, error) {
	_ = opts
	out, err := b.Exec.Run(ctx, "brew", "info", "--json=v2", b.Formula)
	if err != nil {
		return Result{}, fmt.Errorf("brew info: %w", err)
	}

	var info brewInfoV2
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return Result{}, fmt.Errorf("brew info json: %w", err)
	}
	if len(info.Formulae) == 0 {
		return Result{}, fmt.Errorf("brew info json: missing formulae")
	}
	f := info.Formulae[0]
	version := strings.TrimSpace(f.Versions.Stable)
	if version == "" {
		return Result{}, fmt.Errorf("brew info: missing stable version")
	}

	links := map[string]string{}
	if strings.TrimSpace(f.Homepage) != "" {
		links["homepage"] = f.Homepage
	}

	msg := fmt.Sprintf("latest stable %s", version)
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
