package baidu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// TestDomainRegistersOps builds a kit App, registers the baidu domain, and
// asserts the five lookup operations are present. Those same ops back the CLI,
// the HTTP API, and the MCP server.
func TestDomainRegistersOps(t *testing.T) {
	app := kit.New(Identity())
	Domain{}.Register(app)
	have := map[string]bool{}
	for _, op := range app.Ops() {
		have[op.Meta().Name] = true
	}
	for _, w := range []string{"hot", "suggest", "search", "article", "categories"} {
		if !have[w] {
			t.Errorf("missing operation %q", w)
		}
	}
}

func TestClientFromConfig(t *testing.T) {
	def := DefaultConfig()
	c := ClientFromConfig(kit.Config{
		Rate: 0, Retries: -1, Timeout: 0,
		Extra: map[string]string{"user-agent": "x/1", "baiduid": "ABC123"},
	})
	// Zero/negative framework values keep the baidu defaults.
	if c.cfg.Rate != def.Rate {
		t.Errorf("rate = %v, want default %v", c.cfg.Rate, def.Rate)
	}
	if c.cfg.UserAgent != "x/1" {
		t.Errorf("user-agent = %q, want x/1", c.cfg.UserAgent)
	}
	if c.cfg.BaiduID != "ABC123" {
		t.Errorf("baiduid = %q, want ABC123", c.cfg.BaiduID)
	}
}

func TestDefaults(t *testing.T) {
	var c kit.Config
	Defaults(&c)
	def := DefaultConfig()
	if c.Rate != def.Rate || c.Timeout != def.Timeout || c.UserAgent != def.UserAgent {
		t.Errorf("Defaults did not seed baidu baseline: %+v", c)
	}
}

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Error("nil should map to nil")
	}
	if got := mapErr(ErrNotFound); errs.KindOf(got) != errs.KindNoResults {
		t.Errorf("ErrNotFound maps to %v, want no-results", errs.KindOf(got))
	}
	if got := mapErr(ErrBlocked); errs.KindOf(got) != errs.KindRateLimited {
		t.Errorf("ErrBlocked maps to %v, want rate-limited", errs.KindOf(got))
	}
	boom := errors.New("boom")
	if got := mapErr(boom); !errors.Is(got, boom) {
		t.Errorf("other error not passed through: %v", got)
	}
}

func TestLimitOr(t *testing.T) {
	if got := limitOr(0, 30); got != 30 {
		t.Errorf("limitOr(0,30) = %d, want 30", got)
	}
	if got := limitOr(5, 30); got != 5 {
		t.Errorf("limitOr(5,30) = %d, want 5", got)
	}
}

// muxClient returns a baidu Client whose four surfaces all point at one httptest
// server, routed by request path. It is the in-process fixture the handler tests
// drive instead of the live Baidu hosts.
func muxClient(t *testing.T) *Client {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/board"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"cards":[{"content":[
				{"isTop":false,"index":0,"word":"first","url":"u1","hotTag":"3"},
				{"isTop":false,"index":1,"word":"second","url":"u2"}
			]}]}}`))
		case strings.HasPrefix(r.URL.Path, "/sugrec"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"q":"go","p":false,"g":[{"type":"sug","q":"golang"},{"type":"sug","q":"go lang"}]}`))
		case strings.HasPrefix(r.URL.Path, "/api/openapi/BaikeLemmaCardApi"):
			key := r.URL.Query().Get("bk_key")
			if strings.Contains(key, "missing") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"newLemmaId":12345,"title":"人工智能","desc":"AI","abstract":"the study of intelligent agents","card":[{"name":"外文名","value":["Artificial Intelligence"]}]}`))
		case strings.HasPrefix(r.URL.Path, "/wikitag/api/getlemmas"):
			after := r.URL.Query().Get("after")
			if after != "" {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"lemma_id":111,"title":"甲","after":""},{"lemma_id":222,"title":"乙","after":""}]`))
		case strings.HasPrefix(r.URL.Path, "/item/"):
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(baikeHTMLFixture))
		case strings.HasPrefix(r.URL.Path, "/s"):
			// The SERP. A query containing "blocked" returns a CAPTCHA bounce.
			if strings.Contains(r.URL.RawQuery, "blocked") {
				http.Redirect(w, r, "https://wappass.baidu.com/", http.StatusFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(serpHTMLFixture))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.SuggestBaseURL = ts.URL
	cfg.SearchBaseURL = ts.URL
	cfg.BaikeBaseURL = ts.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestHandlersEmit(t *testing.T) {
	c := muxClient(t)
	ctx := context.Background()

	var hots []HotItem
	if err := hot(ctx, hotIn{Client: c, Tab: "realtime"}, func(h HotItem) error {
		hots = append(hots, h)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(hots) != 2 || hots[0].Word != "first" {
		t.Errorf("hot emitted %+v", hots)
	}

	var sug []Suggestion
	if err := suggest(ctx, suggestIn{Client: c, Query: "go"}, func(s Suggestion) error {
		sug = append(sug, s)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(sug) != 2 || sug[0].Word != "golang" {
		t.Errorf("suggest emitted %+v", sug)
	}

	var art *BaikeArticle
	if err := article(ctx, articleIn{Client: c, Ref: "人工智能"}, func(a *BaikeArticle) error {
		art = a
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if art == nil || art.LemmaID != 12345 {
		t.Errorf("article emitted %+v", art)
	}
	if art.Abstract == "" {
		t.Error("article abstract not merged from card API")
	}

	var cats []CategoryLemma
	if err := categories(ctx, categoriesIn{Client: c, Tag: "历史"}, func(l CategoryLemma) error {
		cats = append(cats, l)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 || cats[0].LemmaID != 111 {
		t.Errorf("categories emitted %+v", cats)
	}

	var res []SearchResult
	if err := search(ctx, searchIn{Client: c, Query: "golang", Pages: 1}, func(r SearchResult) error {
		res = append(res, r)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Error("search emitted nothing")
	}
}

func TestArticleNotFound(t *testing.T) {
	c := muxClient(t)
	err := article(context.Background(), articleIn{Client: c, Ref: "missing"}, func(*BaikeArticle) error { return nil })
	if errs.KindOf(err) != errs.KindNoResults {
		t.Fatalf("err kind = %v, want no-results", errs.KindOf(err))
	}
}

func TestSearchBlocked(t *testing.T) {
	c := muxClient(t)
	err := search(context.Background(), searchIn{Client: c, Query: "blocked", Pages: 1}, func(SearchResult) error { return nil })
	if errs.KindOf(err) != errs.KindRateLimited {
		t.Fatalf("err kind = %v, want rate-limited", errs.KindOf(err))
	}
}

const baikeHTMLFixture = `<!DOCTYPE html><html><head>
<link rel="canonical" href="https://baike.baidu.com/item/%E4%BA%BA%E5%B7%A5%E6%99%BA%E8%83%BD/12345">
<script>window.PAGE_DATA = {"lemmaId":12345,"lemmaTitle":"人工智能","uname":"editor"};</script>
</head><body>
<h1 class="J-lemma-title">人工智能</h1>
<div class="J-basic-info"><dl><dt>外文名</dt><dd>Artificial Intelligence</dd></dl></div>
<div data-tag="paragraph">人工智能是研究智能体的学科。</div>
<div data-tag="header" data-level="1" data-name="发展历史"></div>
<div data-tag="paragraph">早期研究始于二十世纪。</div>
<a href="/item/%E6%9C%BA%E5%99%A8%E5%AD%A6%E4%B9%A0/67890">机器学习</a>
</body></html>`

var serpHTMLFixture = `<!DOCTYPE html><html><body>` +
	`<div class="c-container" tpl="se_com_default" data-id="1">` +
	`<h3 class="t"><a href="http://www.example.com/go">The Go Programming Language</a></h3>` +
	`<span class="c-showurl">example.com</span>` +
	`<div class="c-abstract">Go is an open source programming language.</div>` +
	`</div>` +
	strings.Repeat("padding to clear the size threshold for a real page. ", 200) +
	`</body></html>`
