package baidu

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Article resolves and returns a single Baidu Baike encyclopedia article.
//
// ref is one of:
//   - a numeric lemma id ("123456"),
//   - a topic title ("人工智能"),
//   - a title/id pair ("人工智能/123456"), the canonical Baike path shape.
//
// Resolution: a bare title is first looked up through the card API to find its
// numeric lemma id and abstract; then the article HTML is fetched and parsed for
// the full body, infobox, sections, and related ids; finally the card API data is
// merged in to fill any gaps. A ref that resolves to nothing returns ErrNotFound.
func (c *Client) Article(ctx context.Context, ref string) (*BaikeArticle, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrNotFound
	}

	title, lemmaID := parseArticleRef(ref)

	var api *baikeAPIResp
	// Resolve a bare title (no id) through the card API to obtain a lemma id.
	if lemmaID == 0 {
		var err error
		api, err = c.FetchBaikeAPI(ctx, title)
		if err != nil {
			return nil, err
		}
		if api == nil {
			return nil, ErrNotFound
		}
		lemmaID = api.NewLemmaID
		if lemmaID == 0 {
			lemmaID = api.ID
		}
		if api.Title != "" {
			title = api.Title
		}
		if lemmaID == 0 {
			return nil, ErrNotFound
		}
	}

	// Fetch the article HTML. Prefer the canonical title/id URL when we have a
	// title, otherwise resolve by id and let Baike redirect to the canonical URL.
	var lemmaURL string
	if title != "" {
		lemmaURL = c.lemmaURL(title, lemmaID)
	} else {
		lemmaURL = c.lemmaURLByID(lemmaID)
	}

	body, code, err := c.FetchBaikeHTML(ctx, lemmaURL)
	if err != nil {
		return nil, err
	}
	if code == http.StatusNotFound {
		return nil, ErrNotFound
	}
	// Baike anti-bot walls article HTML behind a 403 from at least non-China and
	// anonymous IPs; surface that as a block (exit 5) rather than a generic
	// failure, so the operator knows it is an access wall, not a missing lemma.
	if code == http.StatusForbidden {
		return nil, ErrBlocked
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("baike article HTTP %d for %q", code, ref)
	}

	art, err := ParseBaikeHTML(body)
	if err != nil {
		return nil, err
	}
	if art.LemmaID == 0 {
		art.LemmaID = lemmaID
	}
	if art.Title == "" {
		art.Title = title
	}
	if art.URL == "" {
		art.URL = lemmaURL
	}

	// If we never fetched the card API (the ref carried an id), fetch it now to
	// fill the abstract and any infobox gaps.
	if api == nil {
		key := art.Title
		if key == "" {
			key = title
		}
		if key != "" {
			api, _ = c.FetchBaikeAPI(ctx, key)
		}
	}
	MergeAPIData(art, api)

	if art.LemmaID == 0 && art.Title == "" && art.BodyMarkdown == "" {
		return nil, ErrNotFound
	}

	art.FetchedAt = time.Now().UTC()
	return art, nil
}

// parseArticleRef splits a Baike ref into a title and a numeric lemma id. It
// accepts "id", "title", and "title/id".
func parseArticleRef(ref string) (title string, lemmaID int) {
	// Pure numeric id.
	if id, err := strconv.Atoi(ref); err == nil && id > 0 {
		return "", id
	}
	// title/id shape: the trailing path segment is the id.
	if idx := strings.LastIndexByte(ref, '/'); idx >= 0 {
		tail := ref[idx+1:]
		if id, err := strconv.Atoi(tail); err == nil && id > 0 {
			return strings.TrimSpace(ref[:idx]), id
		}
	}
	return ref, 0
}

// lemmaURL builds the canonical article URL against the client's configured base.
func (c *Client) lemmaURL(title string, lemmaID int) string {
	return fmt.Sprintf("%s/item/%s/%d", c.baikeBaseURL, encodeTitle(title), lemmaID)
}

// lemmaURLByID builds an article URL from only the lemma id; Baike redirects to
// the canonical title/id URL.
func (c *Client) lemmaURLByID(lemmaID int) string {
	return fmt.Sprintf("%s/item/%%E6%%96%%87%%E7%%AB%%A0/%d", c.baikeBaseURL, lemmaID)
}

// LemmaURLByID is the exported, base-aware form used by the mirror crawler so the
// queued URL points at the client's configured Baike host.
func (c *Client) LemmaURLByID(lemmaID int) string { return c.lemmaURLByID(lemmaID) }
