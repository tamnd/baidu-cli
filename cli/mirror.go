package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"

	"github.com/tamnd/baidu-cli/baidu"
	"github.com/tamnd/baidu-cli/mirror/crawl"
	"github.com/tamnd/baidu-cli/mirror/store"
)

// registerMirror adds the local-mirror commands. They are escape hatches, not
// kit operations: a stateful, resumable crawl into a SQLite store is not the
// emit-records shape, so it lives on the CLI only and is absent from the API and
// MCP surfaces.
func registerMirror(app *kit.App) {
	app.AddCommand(seedCmd())
	app.AddCommand(crawlCmd())
	app.AddCommand(exportCmd())
	app.AddCommand(infoCmd())
	app.AddCommand(queueCmd())
	app.AddCommand(jobsCmd())
	app.AddCommand(resetFailedCmd())
}

// mirrorEnv carries the per-run handles a mirror command needs: the resolved
// state (config, renderer), the shared baidu client, and the data dir parsed
// from the command's own flags.
type mirrorEnv struct {
	st      *kit.State
	client  *baidu.Client
	dataDir string
}

func mirrorFrom(ctx context.Context, f *mirrorFlags) *mirrorEnv {
	return &mirrorEnv{
		st:      kit.FromContext(ctx),
		client:  kit.MustClient[*baidu.Client](ctx),
		dataDir: f.data,
	}
}

func (e *mirrorEnv) dataPath() string { return dataPathOf(e.dataDir) }

func (e *mirrorEnv) openStore() (*store.Store, error) {
	st, err := store.Open(e.dataPath())
	if err != nil {
		return nil, errs.Wrap(errs.KindGeneric, err, "open store")
	}
	return st, nil
}

func (e *mirrorEnv) limit() int {
	if e.st != nil {
		return e.st.Globals.Limit
	}
	return 0
}

func (e *mirrorEnv) progressf(format string, args ...any) {
	if e.st != nil && e.st.Config.Quiet {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// mirrorFlags holds the data dir override every mirror command shares.
type mirrorFlags struct {
	data string
}

func (m *mirrorFlags) bind(f *kit.FlagSet) {
	f.StringVar(&m.data, "data", "", "mirror directory (default $BAIDU_DATA or $HOME/data/baidu)")
}

// --- seed ---

func seedCmd() kit.Command {
	return kit.Command{
		Use:   "seed",
		Short: "Add items to the crawl queue",
		Long: `Populate the crawl queue from the built-in Baike seed topics, the category
tags, explicit topics or ids, or a file of refs. Seeding is idempotent: an item
already in the queue keeps its status and history.`,
		Sub: []kit.Command{seedTopicsCmd(), seedURLCmd(), seedListCmd()},
	}
}

func seedTopicsCmd() kit.Command {
	f := &mirrorFlags{}
	var useCategories bool
	return kit.Command{
		Use:   "topics",
		Short: "Seed the built-in Baike seed topics (or category tags)",
		Long: `Resolve the built-in seed topics through the Baike card API and enqueue the
articles that resolve. With --categories, walk the 16 built-in category tags and
enqueue the lemma stubs found under them instead.

  baidu seed topics
  baidu seed topics --categories --limit 200`,
		Args: kit.NoArgs,
		Flags: func(fs *kit.FlagSet) {
			fs.BoolVar(&useCategories, "categories", false, "walk category tags instead of seed topics")
			f.bind(fs)
		},
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			limit := env.limit()
			var added int
			if useCategories {
				added, err = seedCategories(ctx, env, st, limit)
			} else {
				added, err = seedTopics(ctx, env, st, limit)
			}
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "seeded %d items\n", added)
			if added == 0 {
				return errs.NoResults("no items seeded")
			}
			return nil
		},
	}
}

func seedURLCmd() kit.Command {
	f := &mirrorFlags{}
	return kit.Command{
		Use:   "url <ref>...",
		Short: "Seed one or more explicit topics or lemma ids",
		Args:  kit.MinimumNArgs(1),
		Flags: f.bind,
		Run: func(ctx context.Context, args []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()
			added := seedRefs(env, st, args)
			_, _ = fmt.Fprintf(os.Stdout, "seeded %d items\n", added)
			if added == 0 {
				return errs.NoResults("no items seeded")
			}
			return nil
		},
	}
}

func seedListCmd() kit.Command {
	f := &mirrorFlags{}
	return kit.Command{
		Use:   "list <file>",
		Short: "Seed topics or lemma ids from a file, one per line",
		Args:  kit.ExactArgs(1),
		Flags: f.bind,
		Run: func(ctx context.Context, args []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()
			refs, err := readRefList(args[0])
			if err != nil {
				return err
			}
			added := seedRefs(env, st, refs)
			_, _ = fmt.Fprintf(os.Stdout, "seeded %d items\n", added)
			if added == 0 {
				return errs.NoResults("no items seeded")
			}
			return nil
		},
	}
}

// --- crawl ---

func crawlCmd() kit.Command {
	f := &mirrorFlags{}
	var workers int
	var retryFailed bool
	return kit.Command{
		Use:   "crawl",
		Short: "Crawl pending queue items into the mirror",
		Long: `Drain the queue: fetch each pending Baike article or search page, archive
the raw bytes, record the parsed entity, and enqueue the related lemmas it
discovers. The crawl is resumable, so an interrupted run picks up where it left
off, and --limit bounds a single pass.

  baidu crawl --limit 50
  baidu crawl --workers 8
  baidu crawl --retry-failed`,
		Args:  kit.NoArgs,
		Write: true,
		Flags: func(fs *kit.FlagSet) {
			fs.IntVar(&workers, "workers", 4, "number of concurrent workers")
			fs.BoolVar(&retryFailed, "retry-failed", false, "requeue failed items before crawling")
			f.bind(fs)
		},
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if retryFailed {
				n, err := st.ResetFailed()
				if err != nil {
					return errs.Wrap(errs.KindGeneric, err, "reset failed")
				}
				env.progressf("requeued %d failed items", n)
			}

			jobID, _ := st.CreateJob("crawl", fmt.Sprintf("workers=%d limit=%d", workers, env.limit()))
			eng := crawl.New(st, env.client, crawl.Config{
				Concurrency: workers,
				Limit:       env.limit(),
				Logf:        env.progressf,
			})
			env.progressf("crawling...")
			stats, err := eng.Run(ctx)
			if err != nil {
				_ = st.UpdateJob(jobID, store.StatusFailed, stats.Processed, stats.Failed)
				return errs.Wrap(errs.KindGeneric, err, "crawl")
			}
			_ = st.UpdateJob(jobID, store.StatusDone, stats.Processed, stats.Failed)
			_, _ = fmt.Fprintf(os.Stdout,
				"processed %d: %d done, %d blocked, %d failed, %d discovered\n",
				stats.Processed, stats.Done, stats.Blocked, stats.Failed, stats.Discovered)
			if stats.Processed == 0 {
				return errs.NoResults("nothing to crawl")
			}
			return nil
		},
	}
}

// --- export ---

func exportCmd() kit.Command {
	f := &mirrorFlags{}
	var kind, out string
	var markdown bool
	return kit.Command{
		Use:   "export",
		Short: "Export mirrored records as JSONL or Markdown",
		Long: `Stream mirrored records. With no --out the JSONL goes to stdout; with --out
DIR it is written to DIR/<kind>.jsonl. With --markdown, every Baike article is
written as a Markdown file under DIR.

  baidu export --kind article | jq .
  baidu export --out ./export
  baidu export --markdown --out ./articles`,
		Args: kit.NoArgs,
		Flags: func(fs *kit.FlagSet) {
			fs.StringVar(&kind, "kind", "article", "record kind (article|search|suggest)")
			fs.StringVar(&out, "out", "", "write to DIR instead of stdout")
			fs.BoolVar(&markdown, "markdown", false, "write Baike articles as Markdown files (requires --out)")
			f.bind(fs)
		},
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if markdown {
				if out == "" {
					return errs.Usage("--markdown requires --out DIR")
				}
				n, err := exportMarkdown(ctx, st, out)
				if err != nil {
					return errs.Wrap(errs.KindGeneric, err, "export markdown")
				}
				_, _ = fmt.Fprintf(os.Stdout, "wrote %d articles to %s\n", n, out)
				if n == 0 {
					return errs.NoResults("no articles")
				}
				return nil
			}

			if out == "" {
				n, err := st.ExportJSONL(ctx, kind, os.Stdout)
				if err != nil {
					return errs.Wrap(errs.KindGeneric, err, "export")
				}
				if n == 0 {
					return errs.NoResults("no records")
				}
				return nil
			}

			if err := os.MkdirAll(out, 0o755); err != nil {
				return errs.Wrap(errs.KindGeneric, err, "create export dir")
			}
			path := filepath.Join(out, kind+".jsonl")
			file, err := os.Create(path)
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "create export file")
			}
			defer func() { _ = file.Close() }()
			n, err := st.ExportJSONL(ctx, kind, file)
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "export")
			}
			_, _ = fmt.Fprintf(os.Stdout, "wrote %d records to %s\n", n, path)
			if n == 0 {
				return errs.NoResults("no records")
			}
			return nil
		},
	}
}

// --- info ---

func infoCmd() kit.Command {
	f := &mirrorFlags{}
	return kit.Command{
		Use:   "info",
		Short: "Show mirror location, counts, and disk usage",
		Args:  kit.NoArgs,
		Flags: f.bind,
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			dir := env.dataPath()
			_, _ = fmt.Fprintf(os.Stdout, "data dir: %s\n", dir)
			_, _ = fmt.Fprintf(os.Stdout, "database: %s\n", st.DBPath())
			_, _ = fmt.Fprintf(os.Stdout, "disk:     %s\n", humanBytes(dirSize(dir)))

			articles, _ := st.ArticleCount()
			serps, _ := st.SearchResultCount()
			suggests, _ := st.SuggestCount()
			visited, _ := st.VisitedCount()
			_, _ = fmt.Fprintf(os.Stdout, "\nrecords:\n")
			_, _ = fmt.Fprintf(os.Stdout, "  baike articles  %d\n", articles)
			_, _ = fmt.Fprintf(os.Stdout, "  search results  %d\n", serps)
			_, _ = fmt.Fprintf(os.Stdout, "  suggest queries %d\n", suggests)
			_, _ = fmt.Fprintf(os.Stdout, "  visited refs    %d\n", visited)

			stats, err := st.QueueStats()
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "queue stats")
			}
			_, _ = fmt.Fprintf(os.Stdout, "\nqueue by status:\n")
			for _, s := range stats {
				_, _ = fmt.Fprintf(os.Stdout, "  %-8s %d\n", s.Status, s.Count)
			}
			return nil
		},
	}
}

// --- queue ---

// QueueRow is the rendered shape of a queue row.
type QueueRow struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	Ref        string `json:"ref"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	Priority   int    `json:"priority"`
	Error      string `json:"error,omitempty" table:",truncate"`
}

func queueCmd() kit.Command {
	f := &mirrorFlags{}
	var status string
	return kit.Command{
		Use:   "queue",
		Short: "Inspect queue rows by status",
		Long: `List queue rows, optionally filtered by status.

  baidu queue --status failed
  baidu queue --status pending -n 50`,
		Args: kit.NoArgs,
		Flags: func(fs *kit.FlagSet) {
			fs.StringVar(&status, "status", "", "filter by status (pending|done|failed)")
			f.bind(fs)
		},
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			limit := env.limit()
			if limit == 0 {
				limit = 100
			}
			rows, err := st.ListQueue(status, limit)
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "list queue")
			}
			r, err := env.st.Renderer(os.Stdout)
			if err != nil {
				return errs.Usage("%v", err)
			}
			for _, row := range rows {
				if err := r.Emit(QueueRow{
					ID: row.ID, EntityType: row.EntityType, Ref: row.Ref,
					Status: row.Status, Attempts: row.Attempts,
					Priority: row.Priority, Error: row.Error,
				}); err != nil {
					return err
				}
			}
			if err := r.Flush(); err != nil {
				return err
			}
			if len(rows) == 0 {
				return errs.NoResults("no rows")
			}
			return nil
		},
	}
}

// --- jobs ---

// JobRow is the rendered shape of a job-log row.
type JobRow struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Processed int    `json:"processed"`
	Errors    int    `json:"errors"`
}

func jobsCmd() kit.Command {
	f := &mirrorFlags{}
	return kit.Command{
		Use:   "jobs",
		Short: "Show the crawl job log, newest first",
		Args:  kit.NoArgs,
		Flags: f.bind,
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			limit := env.limit()
			if limit == 0 {
				limit = 50
			}
			jobs, err := st.ListJobs(limit)
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "list jobs")
			}
			r, err := env.st.Renderer(os.Stdout)
			if err != nil {
				return errs.Usage("%v", err)
			}
			for _, j := range jobs {
				if err := r.Emit(JobRow{
					ID: j.ID, Kind: j.Kind, Status: j.Status, Detail: j.Detail,
					Processed: j.Processed, Errors: j.Errors,
				}); err != nil {
					return err
				}
			}
			if err := r.Flush(); err != nil {
				return err
			}
			if len(jobs) == 0 {
				return errs.NoResults("no jobs")
			}
			return nil
		},
	}
}

// --- reset-failed ---

func resetFailedCmd() kit.Command {
	f := &mirrorFlags{}
	return kit.Command{
		Use:   "reset-failed",
		Short: "Requeue failed queue rows for another crawl",
		Args:  kit.NoArgs,
		Write: true,
		Flags: f.bind,
		Run: func(ctx context.Context, _ []string) error {
			env := mirrorFrom(ctx, f)
			st, err := env.openStore()
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()
			n, err := st.ResetFailed()
			if err != nil {
				return errs.Wrap(errs.KindGeneric, err, "reset failed")
			}
			_, _ = fmt.Fprintf(os.Stdout, "requeued %d items\n", n)
			if n == 0 {
				return errs.NoResults("no failed items")
			}
			return nil
		},
	}
}

// --- pure helpers (no kit context, so they are unit-testable directly) ---

// dataPathOf resolves the mirror directory: the --data flag, then $BAIDU_DATA,
// then $HOME/data/baidu.
func dataPathOf(dataDir string) string {
	if dataDir != "" {
		return dataDir
	}
	if env := os.Getenv("BAIDU_DATA"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "baidu-data")
	}
	return filepath.Join(home, "data", "baidu")
}

// seedTopics resolves each built-in seed topic through the card API and enqueues
// the articles that resolve.
func seedTopics(ctx context.Context, env *mirrorEnv, st *store.Store, limit int) (int, error) {
	added := 0
	for _, topic := range baidu.BaikeSeedTopics {
		if limit > 0 && added >= limit {
			break
		}
		if ctx.Err() != nil {
			return added, ctx.Err()
		}
		art, err := env.client.Article(ctx, topic)
		if err != nil {
			env.progressf("skip %q: %v", topic, err)
			continue
		}
		if art.LemmaID == 0 {
			continue
		}
		ref := strconv.Itoa(art.LemmaID)
		ok, _ := st.Enqueue(store.QueueItem{
			EntityType: crawl.EntityLemma,
			Ref:        ref,
			URL:        env.client.LemmaURLByID(art.LemmaID),
			Priority:   5,
		})
		if ok {
			added++
		}
	}
	return added, nil
}

// seedCategories walks the built-in category tags and enqueues the lemma stubs
// found under them.
func seedCategories(ctx context.Context, env *mirrorEnv, st *store.Store, limit int) (int, error) {
	lemmas, err := env.client.Categories(ctx, "", limit)
	if err != nil {
		return 0, errs.Wrap(errs.KindGeneric, err, "categories")
	}
	added := 0
	for _, l := range lemmas {
		if l.LemmaID == 0 {
			continue
		}
		ref := strconv.Itoa(l.LemmaID)
		ok, _ := st.Enqueue(store.QueueItem{
			EntityType: crawl.EntityLemma,
			Ref:        ref,
			URL:        env.client.LemmaURLByID(l.LemmaID),
			Priority:   4,
		})
		if ok {
			added++
		}
	}
	return added, nil
}

// seedRefs enqueues explicit refs. A numeric ref is a lemma id; anything else is
// resolved through the card API to a lemma id first.
func seedRefs(env *mirrorEnv, st *store.Store, refs []string) int {
	ctx := context.Background()
	added := 0
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		lemmaID, _ := strconv.Atoi(ref)
		if lemmaID <= 0 {
			art, err := env.client.Article(ctx, ref)
			if err != nil || art.LemmaID == 0 {
				env.progressf("skip %q: not resolvable", ref)
				continue
			}
			lemmaID = art.LemmaID
		}
		ok, _ := st.Enqueue(store.QueueItem{
			EntityType: crawl.EntityLemma,
			Ref:        strconv.Itoa(lemmaID),
			URL:        env.client.LemmaURLByID(lemmaID),
			Priority:   5,
		})
		if ok {
			added++
		}
	}
	return added
}

// readRefList reads refs from a file, one per line, skipping blanks and comments.
func readRefList(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(errs.KindGeneric, err, "read list")
	}
	var refs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		refs = append(refs, line)
	}
	return refs, nil
}

// exportMarkdown writes every stored Baike article as a Markdown file under dir,
// returning the count written.
func exportMarkdown(ctx context.Context, st *store.Store, dir string) (int, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	n := 0
	err := st.EachArticle(ctx, func(lemmaID int, _ string, js []byte) error {
		var art baidu.BaikeArticle
		if err := json.Unmarshal(js, &art); err != nil {
			return nil // skip a corrupt row rather than abort the whole export
		}
		md := articleMarkdown(&art)
		name := fmt.Sprintf("%d.md", lemmaID)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(md), 0o644); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}

// articleMarkdown renders a Baike article as a Markdown document.
func articleMarkdown(art *baidu.BaikeArticle) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", art.Title)
	if art.Subtitle != "" {
		fmt.Fprintf(&b, "*%s*\n\n", art.Subtitle)
	}
	if art.Abstract != "" {
		fmt.Fprintf(&b, "%s\n\n", art.Abstract)
	}
	if len(art.Infobox) > 0 {
		b.WriteString("## 基本信息\n\n")
		for _, it := range art.Infobox {
			fmt.Fprintf(&b, "- **%s**: %s\n", it.Name, it.Value)
		}
		b.WriteString("\n")
	}
	if art.BodyMarkdown != "" {
		b.WriteString(art.BodyMarkdown)
		b.WriteString("\n")
	}
	if art.URL != "" {
		fmt.Fprintf(&b, "\n---\n\n[%s](%s)\n", art.URL, art.URL)
	}
	return b.String()
}

func dirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil && !d.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
