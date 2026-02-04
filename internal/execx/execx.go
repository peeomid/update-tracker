package execx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type OSRunner struct{}

func (r OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	if err != nil {
		if errStr == "" && outStr != "" {
			errStr = outStr
		}
		if errStr == "" {
			return outStr, err
		}
		return outStr, fmt.Errorf("%w: %s", err, errStr)
	}
	return outStr, nil
}

type CachedRunner struct {
	Inner Runner

	mu    sync.Mutex
	cache map[string]string
}

func NewCachedRunner(inner Runner) *CachedRunner {
	return &CachedRunner{
		Inner: inner,
		cache: map[string]string{},
	}
}

func (c *CachedRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	key := name + "\x00" + strings.Join(args, "\x00")
	c.mu.Lock()
	if v, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()

	out, err := c.Inner.Run(ctx, name, args...)
	if err != nil {
		return out, err
	}

	c.mu.Lock()
	c.cache[key] = out
	c.mu.Unlock()
	return out, nil
}
