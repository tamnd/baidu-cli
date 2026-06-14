// Package baidu is the library behind the baidu command: the HTTP client,
// request shaping, and the typed data models for Baidu.
//
// Two APIs used: top.baidu.com for hot search boards (no key required) and
// suggestion.baidu.com for query suggestions (JSONP, no key required).
package baidu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	defaultBaseURL        = "https://top.baidu.com"
	defaultSuggestBaseURL = "https://suggestion.baidu.com"
	// DefaultUserAgent identifies the client to Baidu APIs.
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

// Config holds constructor parameters.
type Config struct {
	BaseURL        string
	SuggestBaseURL string
	UserAgent      string
	Rate           time.Duration
	Timeout        time.Duration
	Retries        int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:        defaultBaseURL,
		SuggestBaseURL: defaultSuggestBaseURL,
		UserAgent:      DefaultUserAgent,
		Rate:           200 * time.Millisecond,
		Timeout:        15 * time.Second,
		Retries:        3,
	}
}

// Client talks to the Baidu hot search and suggest APIs.
type Client struct {
	httpClient     *http.Client
	userAgent      string
	rate           time.Duration
	retries        int
	baseURL        string
	suggestBaseURL string
	mu             sync.Mutex
	last           time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.SuggestBaseURL == "" {
		cfg.SuggestBaseURL = defaultSuggestBaseURL
	}
	return &Client{
		httpClient:     &http.Client{Timeout: cfg.Timeout},
		userAgent:      cfg.UserAgent,
		rate:           cfg.Rate,
		retries:        cfg.Retries,
		baseURL:        cfg.BaseURL,
		suggestBaseURL: cfg.SuggestBaseURL,
	}
}

// Hot fetches the hot search board for the given tab and returns up to limit items.
// tab is one of: realtime, novel, movie, teleplay, car.
func (c *Client) Hot(ctx context.Context, tab string, limit int) ([]HotItem, error) {
	rawURL := c.baseURL + "/api/board?platform=wise&tab=" + url.QueryEscape(tab)
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("hot %s: %w", tab, err)
	}
	var board wireBoard
	if err := json.Unmarshal(body, &board); err != nil {
		return nil, fmt.Errorf("decode hot %s: %w", tab, err)
	}
	var items []HotItem
	for _, card := range board.Data.Cards {
		for _, w := range card.Content {
			items = append(items, wireToHotItem(w))
			if limit > 0 && len(items) >= limit {
				return items, nil
			}
		}
	}
	return items, nil
}

// Suggest fetches query suggestions for the given search term.
// Returns up to 10 items from the Baidu suggest JSONP API.
func (c *Client) Suggest(ctx context.Context, query string) ([]Suggestion, error) {
	rawURL := c.suggestBaseURL + "/su?wd=" + url.QueryEscape(query) + "&cb=cb&sid=3986&logver=1"
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("suggest %q: %w", query, err)
	}
	// JSONP: cb({"q":"word","s":["item1","item2",...]});
	body = bytes.TrimPrefix(body, []byte("cb("))
	body = bytes.TrimSuffix(bytes.TrimRight(body, "\n"), []byte(");"))
	var wire struct {
		Q string   `json:"q"`
		S []string `json:"s"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode suggest %q: %w", query, err)
	}
	out := make([]Suggestion, 0, len(wire.S))
	for i, s := range wire.S {
		out = append(out, Suggestion{Rank: i + 1, Word: s})
	}
	return out, nil
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	return min(d, 5*time.Second)
}
