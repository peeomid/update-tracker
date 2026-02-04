package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Fetcher interface {
	Get(ctx context.Context, url string, headers map[string]string) ([]byte, error)
}

type Client struct {
	Client *http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		Client: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return body, nil
}

type CachedFetcher struct {
	Inner Fetcher

	mu    sync.Mutex
	cache map[string][]byte
}

func NewCachedFetcher(inner Fetcher) *CachedFetcher {
	return &CachedFetcher{
		Inner: inner,
		cache: map[string][]byte{},
	}
}

func (c *CachedFetcher) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	c.mu.Lock()
	if v, ok := c.cache[url]; ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()

	body, err := c.Inner.Get(ctx, url, headers)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[url] = body
	c.mu.Unlock()
	return body, nil
}
