// Package crawl runs the Baidu mirror: it drains the queue, fetches each item
// (a Baike lemma or a web search page) through the baidu client, stores the raw
// bytes and a parsed record, expands the related lemmas it discovers, and marks
// every row's outcome. The engine is resumable (in-flight rows stay pending) and
// polite (the client paces itself), and it never silently caps; it logs what it
// blocks or skips.
package crawl

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/baidu-cli/baidu"
	"github.com/tamnd/baidu-cli/mirror/store"
)

// Entity type names used as queue keys.
const (
	EntityLemma = "lemma"
	EntitySERP  = "serp"
)

// Config tunes a crawl run.
type Config struct {
	Concurrency int // worker pool size (default 4)
	Limit       int // max items to process this run (0 = drain the queue)
	Logf        func(format string, args ...any)
}

// Stats summarizes a run.
type Stats struct {
	Processed  int
	Done       int
	Failed     int
	Blocked    int
	Discovered int
}

// Engine ties the store and the baidu client together.
type Engine struct {
	store  *store.Store
	client *baidu.Client
	cfg    Config
}

// New builds an Engine.
func New(st *store.Store, client *baidu.Client, cfg Config) *Engine {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	return &Engine{store: st, client: client, cfg: cfg}
}

// Run drains pending queue rows until none remain, the limit is hit, or the
// context is cancelled. Newly discovered related lemmas are enqueued as it goes,
// so a single Run expands the mirror outward from its seeds.
func (e *Engine) Run(ctx context.Context) (Stats, error) {
	var st Stats
	batch := e.cfg.Concurrency * 4
	if batch < 16 {
		batch = 16
	}
	for {
		if err := ctx.Err(); err != nil {
			return st, err
		}
		n := batch
		if e.cfg.Limit > 0 {
			if rem := e.cfg.Limit - st.Processed; rem < n {
				n = rem
			}
		}
		if n <= 0 {
			break
		}
		rows, err := e.store.Pop(n)
		if err != nil {
			return st, err
		}
		if len(rows) == 0 {
			break
		}
		e.runBatch(ctx, rows, &st)
	}
	return st, nil
}

func (e *Engine) runBatch(ctx context.Context, rows []store.QueueItem, st *Stats) {
	sem := make(chan struct{}, e.cfg.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, it := range rows {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(it store.QueueItem) {
			defer wg.Done()
			defer func() { <-sem }()
			out := e.process(ctx, it)
			mu.Lock()
			st.Processed++
			st.Done += out.done
			st.Failed += out.failed
			st.Blocked += out.blocked
			st.Discovered += out.discovered
			mu.Unlock()
		}(it)
	}
	wg.Wait()
}

type outcome struct {
	done, failed, blocked, discovered int
}

func (e *Engine) process(ctx context.Context, it store.QueueItem) outcome {
	switch it.EntityType {
	case EntityLemma:
		return e.processLemma(ctx, it)
	case EntitySERP:
		return e.processSERP(ctx, it)
	default:
		_ = e.store.Fail(it, "unknown entity type "+it.EntityType)
		return outcome{failed: 1}
	}
}

// processLemma fetches a Baike article HTML page, stores the raw bytes and a
// parsed record, and enqueues the related lemma ids it discovers.
func (e *Engine) processLemma(ctx context.Context, it store.QueueItem) outcome {
	lemmaID, _ := strconv.Atoi(it.Ref)
	lemmaURL := it.URL
	if lemmaURL == "" && lemmaID > 0 {
		lemmaURL = e.client.LemmaURLByID(lemmaID)
	}
	if lemmaURL == "" {
		_ = e.store.Fail(it, "no lemma URL")
		return outcome{failed: 1}
	}

	body, code, err := e.client.FetchBaikeHTML(ctx, lemmaURL)
	if err != nil {
		_ = e.store.Fail(it, err.Error())
		return outcome{failed: 1}
	}
	if code != 200 {
		_ = e.store.Fail(it, fmt.Sprintf("http %d", code))
		return outcome{failed: 1}
	}

	art, err := baidu.ParseBaikeHTML(body)
	if err != nil {
		_ = e.store.Fail(it, err.Error())
		return outcome{failed: 1}
	}
	if art.LemmaID == 0 {
		art.LemmaID = lemmaID
	}
	art.FetchedAt = time.Now().UTC()

	idStr := strconv.Itoa(art.LemmaID)
	rel, _, werr := e.store.WriteRaw(EntityLemma, idStr, "html", body)
	if werr != nil {
		e.cfg.Logf("raw write lemma %s: %v", idStr, werr)
	}
	js, _ := json.Marshal(art)
	if err := e.store.UpsertArticle(art.LemmaID, art.Title, js, rel); err != nil {
		_ = e.store.Fail(it, err.Error())
		return outcome{failed: 1}
	}

	discovered := e.enqueueRelated(art.RelatedIDs)
	_ = e.store.Done(it)
	return outcome{done: 1, discovered: discovered}
}

func (e *Engine) enqueueRelated(ids []int) int {
	if len(ids) == 0 {
		return 0
	}
	items := make([]store.QueueItem, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		ref := strconv.Itoa(id)
		visited, _ := e.store.IsVisited(EntityLemma, ref)
		if visited {
			continue
		}
		items = append(items, store.QueueItem{
			EntityType: EntityLemma,
			Ref:        ref,
			URL:        e.client.LemmaURLByID(id),
			Priority:   3,
		})
	}
	added, err := e.store.EnqueueBatch(items)
	if err != nil {
		e.cfg.Logf("enqueue related: %v", err)
	}
	return added
}

// processSERP fetches one Baidu search page and stores its results. A CAPTCHA or
// block page is recorded as blocked, not a hard failure, so reset-failed does not
// hammer a walled surface.
func (e *Engine) processSERP(ctx context.Context, it store.QueueItem) outcome {
	query, page, ok := parseSERPRef(it.Ref)
	if !ok {
		_ = e.store.Fail(it, "bad serp ref "+it.Ref)
		return outcome{failed: 1}
	}

	body, served, err := e.client.FetchBaiduSearch(ctx, query, page)
	if err != nil {
		_ = e.store.Fail(it, err.Error())
		return outcome{failed: 1}
	}
	if !served {
		e.cfg.Logf("blocked serp %q page %d (CAPTCHA)", query, page)
		// A wall is terminal for this run: mark it failed outright rather than
		// leaving it pending to be retried in a tight loop.
		_ = e.store.MarkFailed(it, "blocked (CAPTCHA)")
		return outcome{blocked: 1}
	}

	results, err := baidu.ParseSERP(body, query, page)
	if err != nil {
		_ = e.store.Fail(it, err.Error())
		return outcome{failed: 1}
	}

	rawID := fmt.Sprintf("%s_p%d", sanitizeRef(query), page)
	_, _, _ = e.store.WriteRaw(EntitySERP, rawID, "html", body)
	for _, r := range results {
		js, _ := json.Marshal(r)
		if err := e.store.UpsertSearchResult(r.ID, r.Query, r.Page, r.Position, js); err != nil {
			e.cfg.Logf("upsert serp %s: %v", r.ID, err)
		}
	}
	_ = e.store.Done(it)
	return outcome{done: 1}
}

// SERPRef encodes a (query, page) pair as a queue ref.
func SERPRef(query string, page int) string {
	return query + "|" + strconv.Itoa(page)
}

func parseSERPRef(ref string) (query string, page int, ok bool) {
	idx := strings.LastIndexByte(ref, '|')
	if idx < 0 {
		return ref, 1, ref != ""
	}
	p, err := strconv.Atoi(ref[idx+1:])
	if err != nil || p < 1 {
		return "", 0, false
	}
	return ref[:idx], p, ref[:idx] != ""
}

// sanitizeRef makes a query safe as a raw-file name component.
func sanitizeRef(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "q"
	}
	return out
}
