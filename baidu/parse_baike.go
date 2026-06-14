package baidu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ParseBaikeHTML extracts the full article content from a Baike HTML page. It
// uses J-* stable selectors rather than hashed CSS class names, plus the
// embedded window.PAGE_DATA JSON. It is best effort and never panics; the raw
// artifact is the lossless copy.
func ParseBaikeHTML(body []byte) (*BaikeArticle, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	art := &BaikeArticle{}

	// Canonical URL → lemma_id
	canonical, _ := doc.Find(`link[rel="canonical"]`).Attr("href")
	if canonical != "" {
		art.URL = canonical
		art.LemmaID = extractLemmaID(canonical)
	}

	// window.PAGE_DATA — metadata block embedded in a <script> tag
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		txt := s.Text()
		if !strings.Contains(txt, "window.PAGE_DATA") {
			return
		}
		pd := extractPageData(txt)
		if pd == nil {
			return
		}
		if art.LemmaID == 0 && pd.LemmaID > 0 {
			art.LemmaID = pd.LemmaID
		}
		art.Creator = pd.Uname
		art.LastEditor = pd.VersionUname
		if pd.VersionCreateTime > 0 {
			art.LastEditTime = time.Unix(pd.VersionCreateTime, 0).UTC()
		}
		art.VersionID = pd.VersionID
		for _, r := range pd.ExtData.RedirectFrom {
			if r.FromTitle != "" {
				art.Synonyms = append(art.Synonyms, r.FromTitle)
			}
		}
		for _, k := range pd.ExtData.KpdClassify {
			if k.Name != "" {
				art.Tags = append(art.Tags, k.Name)
			}
		}
		for _, c := range pd.ExtData.Classify {
			if c.Path != "" {
				art.Categories = append(art.Categories, c.Path)
			} else if c.Name != "" {
				art.Categories = append(art.Categories, c.Name)
			}
		}
	})

	// Title
	art.Title = strings.TrimSpace(doc.Find("h1.J-lemma-title").First().Text())

	// Subtitle (disambiguation sense label)
	subtitle := doc.Find("div.J-polysemantText").First().Text()
	if subtitle == "" {
		doc.Find("div").Each(func(_ int, s *goquery.Selection) {
			cls, _ := s.Attr("class")
			if strings.Contains(cls, "lemmaDesc") && art.Subtitle == "" {
				s.Find("div").Each(func(_ int, inner *goquery.Selection) {
					text := strings.TrimSpace(inner.Text())
					if text != "" && !strings.Contains(text, "展开") && !strings.Contains(text, "同名") {
						art.Subtitle = text
					}
				})
			}
		})
	} else {
		if idx := strings.Index(subtitle, "展开"); idx > 0 {
			subtitle = strings.TrimSpace(subtitle[:idx])
		}
		art.Subtitle = strings.TrimSpace(subtitle)
	}

	// Infobox (J-basic-info div with dl > div > dt/dd structure)
	doc.Find("div.J-basic-info dt").Each(func(_ int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Text())
		name = strings.ReplaceAll(name, "　", "") // full-width space used for alignment
		name = strings.Join(strings.Fields(name), "")
		if name == "" {
			return
		}
		dd := s.Next()
		if dd.Length() == 0 {
			return
		}
		val := strings.TrimSpace(dd.Text())
		val = reRef.ReplaceAllString(val, "")
		val = strings.TrimSpace(val)
		if val != "" {
			art.Infobox = append(art.Infobox, InfoboxItem{Name: name, Value: val})
		}
	})

	// Body content — walk data-tag blocks in document order
	var bodyParts []string
	var refCount int

	doc.Find("div[data-tag]").Each(func(_ int, s *goquery.Selection) {
		tag, _ := s.Attr("data-tag")
		switch tag {
		case "header":
			level, _ := s.Attr("data-level")
			name, _ := s.Attr("data-name")
			if name == "" {
				name = strings.TrimSpace(s.Text())
			}
			n, _ := strconv.Atoi(level)
			if n <= 0 {
				n = 1
			}
			art.Sections = append(art.Sections, Section{Level: n, Name: name})
			prefix := strings.Repeat("#", n+1) // body h1 → h2
			bodyParts = append(bodyParts, fmt.Sprintf("\n%s %s\n", prefix, name))

		case "paragraph":
			text, refs := extractParagraphText(s)
			refCount += refs
			if strings.TrimSpace(text) != "" {
				bodyParts = append(bodyParts, text+"\n")
			}

		case "list":
			for _, item := range extractListItems(s) {
				bodyParts = append(bodyParts, "- "+item)
			}
			bodyParts = append(bodyParts, "")

		case "table":
			if md := extractTableMarkdown(s); md != "" {
				bodyParts = append(bodyParts, md+"\n")
			}
		}
	})

	art.BodyMarkdown = strings.TrimSpace(strings.Join(bodyParts, "\n"))
	art.ReferencesCount = refCount

	// Images from J-lemma-content-single-image divs
	seen := make(map[string]bool)
	doc.Find(".J-lemma-content-single-image").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("a").First()
		caption, _ := link.Attr("title")
		img := s.Find("img").First()
		src, _ := img.Attr("src")
		if src == "" {
			src, _ = img.Attr("data-src")
		}
		if src != "" && !seen[src] {
			seen[src] = true
			art.Images = append(art.Images, ImageItem{URL: src, Caption: caption})
		}
	})
	if len(art.Images) > 0 && art.ImageURL == "" {
		art.ImageURL = art.Images[0].URL
	}

	// Related article IDs from inline /item/ links
	seenIDs := make(map[int]bool)
	doc.Find(`a[href^="/item/"]`).Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if id := extractLemmaID(href); id > 0 && !seenIDs[id] {
			seenIDs[id] = true
			art.RelatedIDs = append(art.RelatedIDs, id)
		}
	})

	// Categories — footer category/tag anchors
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.HasPrefix(href, "/catego") || strings.HasPrefix(href, "/view/") {
			cls, _ := s.Parent().Attr("class")
			if strings.Contains(cls, "categ") || strings.Contains(cls, "tag") {
				text := strings.TrimSpace(s.Text())
				if text != "" && len(text) < 50 {
					art.Categories = append(art.Categories, text)
				}
			}
		}
	})

	return art, nil
}

// MergeAPIData overlays the Baike card API onto an HTML article, filling any gap.
func MergeAPIData(art *BaikeArticle, api *baikeAPIResp) {
	if api == nil {
		return
	}
	if art.LemmaID == 0 {
		art.LemmaID = api.NewLemmaID
		if art.LemmaID == 0 {
			art.LemmaID = api.ID
		}
	}
	if art.Title == "" {
		art.Title = api.Title
	}
	if art.Subtitle == "" {
		art.Subtitle = api.Desc
	}
	if strings.TrimSpace(api.Abstract) != "" && art.Abstract == "" {
		art.Abstract = api.Abstract
	}
	if api.Image != "" && art.ImageURL == "" {
		art.ImageURL = api.Image
	}
	if len(art.Infobox) == 0 && len(api.Card) > 0 {
		for _, c := range api.Card {
			if c.Name == "" {
				continue
			}
			val := strings.Join(filterEmpty(c.Value), ", ")
			if val != "" {
				art.Infobox = append(art.Infobox, InfoboxItem{Name: c.Name, Value: val})
			}
		}
	}
}

// extractParagraphText extracts plain text from a data-tag="paragraph" div and
// returns the count of [N] reference markers found.
func extractParagraphText(s *goquery.Selection) (string, int) {
	var buf strings.Builder
	var refCount int
	s.Contents().Each(func(_ int, node *goquery.Selection) {
		n := node.Get(0)
		if n == nil {
			return
		}
		switch n.Type {
		case html.TextNode:
			buf.WriteString(n.Data)
		case html.ElementNode:
			switch strings.ToLower(n.Data) {
			case "sup":
				if dtag, _ := node.Attr("data-tag"); dtag == "ref" {
					buf.WriteString(strings.TrimSpace(node.Text()))
					refCount++
				}
			case "a":
				buf.WriteString(strings.TrimSpace(node.Text()))
			case "span":
				if dtxt, _ := node.Attr("data-text"); dtxt == "true" {
					node.Contents().Each(func(_ int, inner *goquery.Selection) {
						in := inner.Get(0)
						if in == nil {
							return
						}
						switch in.Type {
						case html.TextNode:
							buf.WriteString(in.Data)
						case html.ElementNode:
							buf.WriteString(strings.TrimSpace(inner.Text()))
							if strings.ToLower(in.Data) == "sup" {
								refCount++
							}
						}
					})
				} else {
					buf.WriteString(node.Text())
				}
			default:
				buf.WriteString(node.Text())
			}
		}
	})
	return strings.TrimSpace(buf.String()), refCount
}

func extractListItems(s *goquery.Selection) []string {
	var items []string
	s.Find("li").Each(func(_ int, li *goquery.Selection) {
		text := strings.TrimSpace(li.Text())
		text = reRef.ReplaceAllString(text, "")
		text = strings.TrimSpace(text)
		if text != "" {
			items = append(items, text)
		}
	})
	return items
}

func extractTableMarkdown(s *goquery.Selection) string {
	var rows [][]string
	s.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		var row []string
		tr.Find("td, th").Each(func(_ int, cell *goquery.Selection) {
			row = append(row, strings.TrimSpace(cell.Text()))
		})
		if len(row) > 0 {
			rows = append(rows, row)
		}
	})
	if len(rows) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("| " + strings.Join(rows[0], " | ") + " |\n")
	seps := make([]string, len(rows[0]))
	for i := range seps {
		seps[i] = "---"
	}
	sb.WriteString("| " + strings.Join(seps, " | ") + " |\n")
	for _, row := range rows[1:] {
		for len(row) < len(rows[0]) {
			row = append(row, "")
		}
		sb.WriteString("| " + strings.Join(row[:len(rows[0])], " | ") + " |\n")
	}
	return sb.String()
}

// ExtractRelatedLemmaIDs parses all /item/xxx/{id} links from an HTML body, a
// fast regex path used independently of a full parse.
func ExtractRelatedLemmaIDs(body []byte) []int {
	matches := reLemmaLink.FindAllSubmatch(body, -1)
	seen := make(map[int]bool)
	var ids []int
	for _, m := range matches {
		id, err := strconv.Atoi(string(m[1]))
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// LemmaURL builds the canonical Baike URL for a given lemma id and title.
func LemmaURL(title string, lemmaID int) string {
	return fmt.Sprintf("%s/item/%s/%d", BaikeBaseURL, encodeTitle(title), lemmaID)
}

// LemmaURLByID builds the Baike URL using only the lemma id; Baike redirects to
// the full canonical URL.
func LemmaURLByID(lemmaID int) string {
	return fmt.Sprintf("%s/item/%%E6%%96%%87%%E7%%AB%%A0/%d", BaikeBaseURL, lemmaID)
}

// EncodeTitle percent-encodes non-ASCII characters in a Baike title for URLs.
func EncodeTitle(title string) string { return encodeTitle(title) }

func encodeTitle(title string) string {
	var b strings.Builder
	for _, r := range title {
		if r < 128 {
			b.WriteRune(r)
		} else {
			for _, by := range []byte(string(r)) {
				fmt.Fprintf(&b, "%%%02X", by)
			}
		}
	}
	return b.String()
}

// extractLemmaID extracts the numeric lemma id from a Baike URL or path.
func extractLemmaID(rawURL string) int {
	if idx := strings.IndexByte(rawURL, '?'); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	parts := strings.Split(strings.TrimRight(rawURL, "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

var (
	reRef       = regexp.MustCompile(`\[\d+(?:[–-]\d+)?\]`)
	reLemmaLink = regexp.MustCompile(`/item/[^/"<>?#\s]+/(\d+)`)
)

// extractPageData parses the window.PAGE_DATA JSON embedded in a Baike script.
func extractPageData(scriptText string) *pageDataResp {
	idx := strings.Index(scriptText, "window.PAGE_DATA")
	if idx < 0 {
		return nil
	}
	start := strings.Index(scriptText[idx:], "{")
	if start < 0 {
		return nil
	}
	start += idx
	depth := 0
	end := start
	for end < len(scriptText) {
		switch scriptText[end] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end++
				goto done
			}
		}
		end++
	}
done:
	if depth != 0 || end <= start {
		return nil
	}
	var pd pageDataResp
	if err := json.Unmarshal([]byte(scriptText[start:end]), &pd); err != nil {
		return nil
	}
	return &pd
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		s = strings.TrimSpace(stripHTML(s))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// stripHTML removes HTML tags from a string, returning plain text.
func stripHTML(s string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return s
	}
	return doc.Text()
}
