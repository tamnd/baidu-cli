package baidu

import (
	"context"
	"fmt"
)

// Categories walks Baidu Baike category tags and returns the lemma stubs found
// under them. With a tag, only that tag is walked. With an empty tag, all 16
// built-in top-level tags are walked in turn.
//
// limit caps the total number of lemmas returned across all walked tags; a
// non-positive limit means no cap. The wikitag API is open and paginated by an
// opaque "after" cursor; this follows the cursor until it is exhausted or the
// limit is reached.
func (c *Client) Categories(ctx context.Context, tag string, limit int) ([]CategoryLemma, error) {
	tags := BaikeCategoryTags
	if tag != "" {
		tags = []string{tag}
	}

	var out []CategoryLemma
	for _, t := range tags {
		after := ""
		for {
			items, next, err := c.FetchCategoryLemmas(ctx, t, after)
			if err != nil {
				return out, fmt.Errorf("categories %q: %w", t, err)
			}
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				if it.LemmaID == 0 && it.Title == "" {
					continue
				}
				out = append(out, CategoryLemma{
					LemmaID:  it.LemmaID,
					Title:    it.Title,
					Category: t,
				})
				if limit > 0 && len(out) >= limit {
					return out, nil
				}
			}
			if next == "" || next == after {
				break
			}
			after = next
		}
	}
	return out, nil
}
