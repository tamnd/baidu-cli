package baidu

import "testing"

func TestParseBaikeHTML(t *testing.T) {
	art, err := ParseBaikeHTML([]byte(baikeHTMLFixture))
	if err != nil {
		t.Fatalf("ParseBaikeHTML: %v", err)
	}
	if art.LemmaID != 12345 {
		t.Errorf("LemmaID = %d, want 12345", art.LemmaID)
	}
	if art.Title != "人工智能" {
		t.Errorf("Title = %q, want 人工智能", art.Title)
	}
	if len(art.Infobox) != 1 || art.Infobox[0].Name != "外文名" {
		t.Errorf("Infobox = %+v", art.Infobox)
	}
	if len(art.Sections) != 1 || art.Sections[0].Name != "发展历史" {
		t.Errorf("Sections = %+v", art.Sections)
	}
	if art.BodyMarkdown == "" {
		t.Error("BodyMarkdown is empty")
	}
	wantRelated := false
	for _, id := range art.RelatedIDs {
		if id == 67890 {
			wantRelated = true
		}
	}
	if !wantRelated {
		t.Errorf("RelatedIDs missing 67890: %v", art.RelatedIDs)
	}
}

func TestExtractLemmaID(t *testing.T) {
	cases := map[string]int{
		"https://baike.baidu.com/item/%E4%BA%BA/12345": 12345,
		"/item/x/67890":      67890,
		"/item/x/67890?fr=a": 67890,
		"/item/x/":           0,
		"":                   0,
	}
	for in, want := range cases {
		if got := extractLemmaID(in); got != want {
			t.Errorf("extractLemmaID(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseArticleRef(t *testing.T) {
	cases := []struct {
		ref       string
		wantTitle string
		wantID    int
	}{
		{"12345", "", 12345},
		{"人工智能", "人工智能", 0},
		{"人工智能/12345", "人工智能", 12345},
		{"a/b/678", "a/b", 678},
	}
	for _, c := range cases {
		gotTitle, gotID := parseArticleRef(c.ref)
		if gotTitle != c.wantTitle || gotID != c.wantID {
			t.Errorf("parseArticleRef(%q) = (%q, %d), want (%q, %d)",
				c.ref, gotTitle, gotID, c.wantTitle, c.wantID)
		}
	}
}

func TestParseSERP(t *testing.T) {
	results, err := ParseSERP([]byte(serpHTMLFixture), "golang", 1)
	if err != nil {
		t.Fatalf("ParseSERP: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "The Go Programming Language" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.URL != "http://www.example.com/go" {
		t.Errorf("URL = %q", r.URL)
	}
	if r.Query != "golang" || r.Page != 1 || r.Position != 1 {
		t.Errorf("query/page/position = %q/%d/%d", r.Query, r.Page, r.Position)
	}
	if r.ID == "" {
		t.Error("ID is empty")
	}
}

func TestIsCaptchaPage(t *testing.T) {
	if !IsCaptchaPage([]byte("short")) {
		t.Error("short page should be a captcha")
	}
	big := make([]byte, 6000)
	for i := range big {
		big[i] = 'x'
	}
	if IsCaptchaPage(big) {
		t.Error("big clean page should not be a captcha")
	}
	if !IsCaptchaPage([]byte(string(big) + "wappass.baidu.com")) {
		t.Error("wappass marker should be a captcha")
	}
}

func TestEncodeTitle(t *testing.T) {
	if got := encodeTitle("AI"); got != "AI" {
		t.Errorf("ascii encodeTitle = %q, want AI", got)
	}
	got := encodeTitle("人")
	if got == "人" || got == "" {
		t.Errorf("non-ascii encodeTitle = %q, want percent-encoded", got)
	}
}
