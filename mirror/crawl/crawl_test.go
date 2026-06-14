package crawl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/baidu-cli/baidu"
	"github.com/tamnd/baidu-cli/mirror/store"
)

const baikeHTML = `<!DOCTYPE html><html><head>
<link rel="canonical" href="https://baike.baidu.com/item/x/12345">
</head><body>
<h1 class="J-lemma-title">人工智能</h1>
<div data-tag="paragraph">正文。</div>
<a href="/item/y/67890">机器学习</a>
</body></html>`

func newEngine(t *testing.T) (*store.Store, *baidu.Client) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/item/"):
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(baikeHTML))
		case strings.HasPrefix(r.URL.Path, "/s"):
			if strings.Contains(r.URL.RawQuery, "blocked") {
				http.Redirect(w, r, "https://wappass.baidu.com/", http.StatusFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<div class="c-container" tpl="x" data-id="1">` +
				`<h3 class="t"><a href="http://e.com">T</a></h3>` +
				`<div class="c-abstract">snip</div></div>` +
				strings.Repeat("pad to clear the size threshold for a page. ", 200)))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)

	cfg := baidu.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.SuggestBaseURL = ts.URL
	cfg.SearchBaseURL = ts.URL
	cfg.BaikeBaseURL = ts.URL
	cfg.Rate = 0

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, baidu.NewClient(cfg)
}

func TestCrawlLemma(t *testing.T) {
	st, client := newEngine(t)
	if _, err := st.Enqueue(store.QueueItem{
		EntityType: EntityLemma, Ref: "12345", URL: client.LemmaURLByID(12345),
	}); err != nil {
		t.Fatal(err)
	}

	// Limit the run to the single seed so the assertion on discovery is clean;
	// without a limit the engine would also drain the discovered lemma.
	eng := New(st, client, Config{Concurrency: 2, Limit: 1})
	stats, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Done != 1 {
		t.Errorf("done = %d, want 1", stats.Done)
	}
	// The related lemma 67890 should have been discovered and enqueued.
	if stats.Discovered != 1 {
		t.Errorf("discovered = %d, want 1", stats.Discovered)
	}
	if c, _ := st.ArticleCount(); c != 1 {
		t.Errorf("ArticleCount = %d, want 1", c)
	}
	visited, _ := st.IsVisited(EntityLemma, "12345")
	if !visited {
		t.Error("lemma 12345 should be visited")
	}
}

func TestCrawlSERPBlocked(t *testing.T) {
	st, client := newEngine(t)
	_, _ = st.Enqueue(store.QueueItem{EntityType: EntitySERP, Ref: SERPRef("blocked", 1)})

	eng := New(st, client, Config{Concurrency: 1})
	stats, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Blocked != 1 {
		t.Errorf("blocked = %d, want 1", stats.Blocked)
	}
}

func TestCrawlSERP(t *testing.T) {
	st, client := newEngine(t)
	_, _ = st.Enqueue(store.QueueItem{EntityType: EntitySERP, Ref: SERPRef("golang", 1)})

	eng := New(st, client, Config{Concurrency: 1})
	stats, err := eng.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Done != 1 {
		t.Errorf("done = %d, want 1", stats.Done)
	}
	if c, _ := st.SearchResultCount(); c != 1 {
		t.Errorf("SearchResultCount = %d, want 1", c)
	}
}

func TestParseSERPRef(t *testing.T) {
	q, p, ok := parseSERPRef("hello world|3")
	if !ok || q != "hello world" || p != 3 {
		t.Errorf("parseSERPRef = (%q, %d, %v)", q, p, ok)
	}
	if _, _, ok := parseSERPRef(""); ok {
		t.Error("empty ref should not be ok")
	}
}
