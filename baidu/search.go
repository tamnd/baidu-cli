package baidu

import (
	"context"
	"fmt"
)

// Search runs a Baidu web search for query across the given number of result
// pages (page 1..pages) and returns the merged result list in rank order.
//
// The web SERP is the only walled Baidu surface. When Baidu serves a CAPTCHA or
// block page instead of results, Search returns ErrBlocked so the caller can map
// it to a rate-limited exit and surface the --baiduid hint.
func (c *Client) Search(ctx context.Context, query string, pages int) ([]SearchResult, error) {
	if pages < 1 {
		pages = 1
	}
	var all []SearchResult
	for page := 1; page <= pages; page++ {
		body, ok, err := c.FetchBaiduSearch(ctx, query, page)
		if err != nil {
			return all, fmt.Errorf("search %q page %d: %w", query, page, err)
		}
		if !ok {
			// First page blocked means we have nothing; later pages blocked means
			// we keep what we already gathered.
			if page == 1 {
				return nil, ErrBlocked
			}
			return all, ErrBlocked
		}
		results, err := ParseSERP(body, query, page)
		if err != nil {
			return all, fmt.Errorf("search %q page %d: %w", query, page, err)
		}
		all = append(all, results...)
	}
	return all, nil
}
