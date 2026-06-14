package baidu

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// FetchBaikeAPI fetches the Baike JSON card API for a given search key. It is
// the cheap probe that resolves a topic name to a numeric lemma id.
func (c *Client) FetchBaikeAPI(ctx context.Context, key string) (*baikeAPIResp, error) {
	apiURL := fmt.Sprintf("%s/api/openapi/BaikeLemmaCardApi?scope=103&format=json&appid=379020&bk_key=%s&bk_length=600",
		c.baikeBaseURL, url.QueryEscape(key))
	body, code, err := c.fetchBaike(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	if code == http.StatusNotFound {
		return nil, nil
	}
	if code == http.StatusForbidden {
		return nil, ErrBlocked
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("baike API HTTP %d for key=%s", code, key)
	}
	var resp baikeAPIResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode baike API for %q: %w", key, err)
	}
	// The card API answers a 200 with {"errno":2} (no card data) when it declines
	// the request — the anti-bot path from non-China and anonymous IPs. Surface
	// that as a block, not as a missing lemma.
	if resp.Errno != 0 {
		return nil, ErrBlocked
	}
	return &resp, nil
}

// FetchBaikeHTML fetches a Baike article HTML page by lemma URL.
func (c *Client) FetchBaikeHTML(ctx context.Context, lemmaURL string) ([]byte, int, error) {
	return c.fetchBaike(ctx, lemmaURL)
}

// FetchCategoryLemmas fetches one page of lemmas for a Baike tag/category name.
// It returns the items and the cursor for the next page (empty = last page).
func (c *Client) FetchCategoryLemmas(ctx context.Context, tagName, after string) ([]wikitagLemma, string, error) {
	apiURL := fmt.Sprintf("%s/wikitag/api/getlemmas?tagName=%s&limit=20&after=%s",
		c.baikeBaseURL, url.QueryEscape(tagName), url.QueryEscape(after))
	body, code, err := c.fetchBaike(ctx, apiURL)
	if err != nil {
		return nil, "", err
	}
	if code == http.StatusForbidden {
		return nil, "", ErrBlocked
	}
	if code != http.StatusOK {
		return nil, "", fmt.Errorf("wikitag HTTP %d for %s", code, tagName)
	}
	var items []wikitagLemma
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, "", fmt.Errorf("decode wikitag response: %w", err)
	}
	nextAfter := ""
	if len(items) > 0 {
		nextAfter = items[len(items)-1].After
	}
	return items, nextAfter, nil
}

// FetchBaiduSearch fetches a Baidu Search SERP page. It returns (body, true, nil)
// on success and (nil, false, nil) on a CAPTCHA block, so the caller can degrade
// cleanly rather than treat a challenge page as data.
func (c *Client) FetchBaiduSearch(ctx context.Context, query string, page int) ([]byte, bool, error) {
	offset := (page - 1) * 10
	rsvT := rsvTParam()
	searchURL := fmt.Sprintf(
		"%s/s?wd=%s&pn=%d&rn=10&ie=utf-8&f=8&rsv_bp=1&rsv_idx=1&tn=baidu&oq=%s&rsv_pq=%s&rsv_t=%s&rqlang=cn",
		c.searchBaseURL, url.QueryEscape(query), offset, url.QueryEscape(query),
		randomHex64(), rsvT)

	body, code, err := c.fetchSearch(ctx, searchURL, rsvT)
	if err != nil {
		return nil, false, err
	}
	// A redirect is a CAPTCHA or block bounce.
	if code == http.StatusFound || code == http.StatusMovedPermanently {
		return nil, false, nil
	}
	if code != http.StatusOK {
		return nil, false, fmt.Errorf("search HTTP %d", code)
	}
	if IsCaptchaPage(body) {
		return nil, false, nil
	}
	return body, true, nil
}

// fetchBaike performs a GET for the encyclopedia (and the open suggest sugrec
// path) with minimal anti-bot headers, following redirects.
func (c *Client) fetchBaike(ctx context.Context, rawURL string) ([]byte, int, error) {
	maxAttempts := c.retries
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.pace()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", c.ua())
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("Referer", c.baikeBaseURL+"/")
		req.AddCookie(&http.Cookie{Name: "baikeVisitId", Value: newUUID()})

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return nil, 0, err
			}
			if !sleepCtx(ctx, time.Duration(attempt)*2*time.Second) {
				return nil, 0, ctx.Err()
			}
			continue
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, resp.StatusCode, err
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt == maxAttempts {
				return nil, resp.StatusCode, fmt.Errorf("rate limited HTTP %d", resp.StatusCode)
			}
			if !sleepCtx(ctx, time.Duration(attempt*attempt)*5*time.Second) {
				return nil, resp.StatusCode, ctx.Err()
			}
			continue
		}
		return body, resp.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("all attempts failed for %s", rawURL)
}

// fetchSearch performs a GET for the SERP with full browser headers and
// synthetic cookies. Per the porting note, the manual Accept-Encoding header is
// intentionally omitted so Go's transport transparently gunzips the body.
func (c *Client) fetchSearch(ctx context.Context, rawURL, rsvT string) ([]byte, int, error) {
	maxAttempts := c.retries
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		c.pace()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", c.ua())
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Referer", c.searchBaseURL+"/")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="134", "Google Chrome";v="134"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("Connection", "close")
		req.Header.Set("Cookie", c.genSearchCookies(rsvT))

		resp, err := c.httpSearch.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return nil, 0, err
			}
			if !sleepCtx(ctx, time.Duration(attempt)*2*time.Second) {
				return nil, 0, ctx.Err()
			}
			continue
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, resp.StatusCode, err
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt == maxAttempts {
				return nil, resp.StatusCode, fmt.Errorf("rate limited HTTP %d", resp.StatusCode)
			}
			if !sleepCtx(ctx, time.Duration(attempt*attempt)*15*time.Second) {
				return nil, resp.StatusCode, ctx.Err()
			}
			continue
		}
		return body, resp.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("all attempts failed for %s", rawURL)
}

// genSearchCookies generates a plausible Baidu session cookie string. BAIDUID
// can be random; Baidu does not cryptographically validate it for the SERP. The
// pattern follows ohblue/baidu-serp-api gen_pc_cookies().
func (c *Client) genSearchCookies(rsvT string) string {
	baiduid := c.baiduID
	if baiduid == "" {
		baiduid = randomHex32() + ":FG=1"
	} else if !strings.Contains(baiduid, ":FG=") {
		baiduid += ":FG=1"
	}
	bidupsid := randomHex32()
	pstm := time.Now().Unix()
	bdsvrtm := rand.Intn(990) + 10
	bdorz := randomHex32()
	baHector := randomAlphaNum(32)
	bdUpn := rand.Intn(900000) + 100000
	pssids := make([]string, 20)
	for i := range pssids {
		pssids[i] = fmt.Sprintf("%d", rand.Intn(10000)+60000)
	}
	hPsPssid := strings.Join(pssids, "_")
	h645ec := rsvT
	if len(h645ec) > 43 {
		h645ec = h645ec[:43]
	}
	zfy := randomAlphaNumPlus(43)
	baikeVisitID := newUUID()
	sessionParts := make([]string, 18)
	for i := range sessionParts {
		sessionParts[i] = fmt.Sprintf("%d", rand.Intn(21))
	}
	cookieSession := strings.Join(sessionParts, "_") + fmt.Sprintf("%%7C5%%230_0_%d%%7C1", pstm)

	return fmt.Sprintf(
		"BIDUPSID=%s; PSTM=%d; BAIDUID=%s; "+
			"delPer=0; BD_CK_SAM=1; PSINO=5; BD_UPN=%d; "+
			"BAIDUID_BFESS=%s; BDORZ=%s; BA_HECTOR=%s; "+
			"H_WISE_SIDS=%s; ZFY=%s; H_PS_PSSID=%s; "+
			"H_PS_645EC=%s; baikeVisitId=%s; "+
			"BDSVRTM=%d; COOKIE_SESSION=%s",
		bidupsid, pstm, baiduid,
		bdUpn,
		baiduid, bdorz, baHector,
		hPsPssid, zfy, hPsPssid,
		h645ec, baikeVisitID,
		bdsvrtm, cookieSession,
	)
}

// sleepCtx waits for d or the context, returning false if the context is done.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// rsvTParam generates the rsv_t URL parameter: md5 of the current timestamp.
func rsvTParam() string {
	h := md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return fmt.Sprintf("%x", h)
}

func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// randomBytes returns n pseudo-random bytes. These feed cookie decoration that
// Baidu never validates, so the non-crypto math/rand source is intentional.
func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < len(b); i += 8 {
		v := rand.Uint64()
		for j := 0; j < 8 && i+j < len(b); j++ {
			b[i+j] = byte(v >> (8 * j))
		}
	}
	return b
}

// randomHex32 returns a random 32-char uppercase hex string.
func randomHex32() string {
	return strings.ToUpper(fmt.Sprintf("%x", randomBytes(16)))
}

// randomHex64 returns a random 64-bit hex string.
func randomHex64() string {
	return fmt.Sprintf("%016x", rand.Uint64())
}

func randomAlphaNum(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomAlphaNumPlus(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/:"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func newUUID() string {
	b := randomBytes(16)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
