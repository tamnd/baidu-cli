package baidu

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseSERP parses a Baidu Search HTML page for a given query and page number.
func ParseSERP(body []byte, query string, page int) ([]SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse SERP HTML: %w", err)
	}

	now := time.Now()
	var results []SearchResult
	pos := 1

	doc.Find("div.c-container").Each(func(_ int, s *goquery.Selection) {
		tpl, _ := s.Attr("tpl")
		dataID, _ := s.Attr("data-id")
		if dataID == "" && tpl == "" {
			return
		}

		titleSel := s.Find("h3.t a").First()
		title := strings.TrimSpace(titleSel.Text())
		rawURL, _ := titleSel.Attr("href")

		displayURL := strings.TrimSpace(s.Find("span.c-showurl").First().Text())
		snippet := strings.TrimSpace(s.Find("span.c-abstract, div.c-abstract").First().Text())
		isAd := strings.Contains(s.Find(`em[class*="admark"], span[class*="admark"]`).Text(), "广告")

		if title == "" && rawURL == "" {
			return
		}

		results = append(results, SearchResult{
			ID:         serpID(query, page, pos),
			Query:      query,
			Page:       page,
			Position:   pos,
			URL:        rawURL,
			DisplayURL: displayURL,
			Title:      title,
			Snippet:    snippet,
			Tpl:        tpl,
			IsAd:       isAd,
			FetchedAt:  now,
		})
		pos++
	})

	return results, nil
}

func serpID(query string, page, pos int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", query, page, pos)))
	return fmt.Sprintf("%x", h[:8])
}

// IsCaptchaPage reports whether the body looks like a Baidu CAPTCHA or block page.
func IsCaptchaPage(body []byte) bool {
	if len(body) < 5000 {
		return true
	}
	s := string(body)
	return strings.Contains(s, "wappass.baidu.com") ||
		strings.Contains(s, "captcha") ||
		strings.Contains(s, "timeout hide-callback") ||
		strings.Contains(s, "verify.baidu.com")
}
