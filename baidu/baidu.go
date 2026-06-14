// Package baidu is the library behind the baidu command: the HTTP clients,
// request shaping, anti-bot handling, parsers, and typed data models for Baidu.
//
// Baidu is several products behind one brand, and each is served by its own
// host: top.baidu.com for the hot search board, www.baidu.com for the typeahead
// (the open sugrec endpoint) and web search, and baike.baidu.com for the
// encyclopedia. The hot board and suggest serve open JSON from any IP. The web
// SERP is walled (CAPTCHA) and best effort. The encyclopedia (card API, article
// HTML, category API) is anti-bot/geo-walled: it answers from China IPs but
// returns 403 / {"errno":2} / empty lists from at least non-China and anonymous
// IPs, which the client surfaces as a clean block rather than a hard failure.
// No API key is needed anywhere.
package baidu

import (
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
	defaultBaseURL = "https://top.baidu.com"
	// The open sugrec suggest endpoint lives on www.baidu.com and returns clean
	// UTF-8 JSON; the legacy suggestion.baidu.com/su path serves GBK JSONP.
	defaultSuggestBaseURL = "https://www.baidu.com"
	// DefaultUserAgent identifies the client to Baidu when the operator does
	// not pin one with --user-agent. The search and baike fetchers rotate the
	// pool instead, but this stays the baseline for the hot board and suggest.
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

	// DefaultRate is the spacing for the open hot board and suggest surfaces.
	DefaultRate = 200 * time.Millisecond
	// DefaultTimeout is the per-request HTTP timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultRetries is how many times a transient failure is retried.
	DefaultRetries = 3
)

// Config holds constructor parameters. When BaseURL or one of the per-surface
// base URLs is set, the client resolves every matching host to it, which is the
// injectable-base pattern the tests use to point all four surfaces at one
// httptest mux. An empty value means the real Baidu host.
type Config struct {
	BaseURL        string // top.baidu.com override
	SuggestBaseURL string // suggestion.baidu.com override
	SearchBaseURL  string // www.baidu.com override
	BaikeBaseURL   string // baike.baidu.com override
	UserAgent      string // "" rotates the pool for search/baike
	BaiduID        string // "" synthesizes a BAIDUID per SERP request
	Rate           time.Duration
	Timeout        time.Duration
	Retries        int
}

// DefaultConfig returns sensible defaults that hit the real Baidu hosts.
func DefaultConfig() Config {
	return Config{
		BaseURL:        defaultBaseURL,
		SuggestBaseURL: defaultSuggestBaseURL,
		SearchBaseURL:  SearchBaseURL,
		BaikeBaseURL:   BaikeBaseURL,
		UserAgent:      DefaultUserAgent,
		Rate:           DefaultRate,
		Timeout:        DefaultTimeout,
		Retries:        DefaultRetries,
	}
}

// Client talks to all four Baidu surfaces. It holds two HTTP clients: one
// follows redirects (the hot board, suggest, and Baike, whose article pages
// legitimately 302 to a canonical URL) and one blocks them (the SERP, where a
// 302 is a CAPTCHA bounce). A single mutex paces requests across both.
type Client struct {
	httpClient     *http.Client // follows redirects
	httpSearch     *http.Client // blocks redirects for CAPTCHA detection
	cfg            Config
	userAgent      string
	rate           time.Duration
	retries        int
	baiduID        string
	baseURL        string
	suggestBaseURL string
	searchBaseURL  string
	baikeBaseURL   string
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
	if cfg.SearchBaseURL == "" {
		cfg.SearchBaseURL = SearchBaseURL
	}
	if cfg.BaikeBaseURL == "" {
		cfg.BaikeBaseURL = BaikeBaseURL
	}
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     8,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second,
		DisableKeepAlives:   true,
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout, Transport: transport},
		httpSearch: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			// Do not follow redirects on the SERP: a 301/302 is a CAPTCHA bounce.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cfg:            cfg,
		userAgent:      cfg.UserAgent,
		rate:           cfg.Rate,
		retries:        cfg.Retries,
		baiduID:        cfg.BaiduID,
		baseURL:        cfg.BaseURL,
		suggestBaseURL: cfg.SuggestBaseURL,
		searchBaseURL:  cfg.SearchBaseURL,
		baikeBaseURL:   cfg.BaikeBaseURL,
	}
}

// Hot fetches the hot search board for the given tab and returns up to limit
// items. tab is one of: realtime, novel, movie, teleplay, car.
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
		for _, group := range card.Content {
			// Realtime nests items one level deeper (content[].content[]); the
			// other boards carry the item fields on the group itself. Emit the
			// nested items when present, else the group as a single item.
			itemsIn := group.Content
			if len(itemsIn) == 0 {
				itemsIn = []wireItem{group.wireItem}
			}
			for _, w := range itemsIn {
				if w.Word == "" {
					continue
				}
				items = append(items, wireToHotItem(w))
				if limit > 0 && len(items) >= limit {
					return items, nil
				}
			}
		}
	}
	return items, nil
}

// Suggest fetches query suggestions for the given search term. It uses the open
// sugrec endpoint, which returns clean UTF-8 JSON ({"q":..,"g":[{"q":..}]}) from
// any IP — unlike the legacy /su path, which serves GBK-encoded, unquoted-key
// JSONP that a JSON decoder cannot parse.
func (c *Client) Suggest(ctx context.Context, query string) ([]Suggestion, error) {
	rawURL := c.suggestBaseURL + "/sugrec?prod=pc&ie=utf-8&wd=" + url.QueryEscape(query)
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, fmt.Errorf("suggest %q: %w", query, err)
	}
	var wire struct {
		Q string `json:"q"`
		G []struct {
			Q string `json:"q"`
		} `json:"g"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("decode suggest %q: %w", query, err)
	}
	out := make([]Suggestion, 0, len(wire.G))
	for _, g := range wire.G {
		if g.Q == "" {
			continue
		}
		out = append(out, Suggestion{Rank: len(out) + 1, Word: g.Q})
	}
	return out, nil
}

// get fetches a URL with pacing and retries, following redirects. It is used
// for the open JSON/JSONP surfaces (hot board, suggest).
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
	req.Header.Set("User-Agent", c.ua())
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

// ua returns the configured User-Agent, or a rotated one from the pool when the
// operator did not pin a fixed value.
func (c *Client) ua() string {
	if c.userAgent != "" {
		return c.userAgent
	}
	return randomUserAgent()
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
