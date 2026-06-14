// Package store is the persistence layer for the Baidu mirror: a crawl queue, a
// visited set, the harvested records (Baike articles, search results, suggest
// queries), the per-run job log, and the raw artifact files captured verbatim.
//
// It uses a pure-Go SQLite (modernc.org/sqlite) so the binary keeps building
// with CGO_ENABLED=0. The database lives at <dir>/baidu.db; raw artifacts go
// under <dir>/raw/ and JSONL/Markdown exports under <dir>/export/.
//
// The schema is the go-mizu mirror schema translated from DuckDB to SQLite:
// CREATE SEQUENCE / nextval become INTEGER PRIMARY KEY AUTOINCREMENT, NOW()
// becomes CURRENT_TIMESTAMP, and the final columns are declared up front rather
// than added by ALTER TABLE migrations.
package store

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Queue status values.
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// QueueItem is one row of the crawl queue.
type QueueItem struct {
	ID         int64
	EntityType string // lemma, serp
	Ref        string // lemma id, title, or "query|page" for serp
	URL        string
	Priority   int
	Status     string
	Attempts   int
	Error      string
}

// Job is one row of the per-run job log.
type Job struct {
	ID        int64
	Kind      string // seed, crawl, export, reset-failed
	Status    string // running, done, failed
	Detail    string
	Processed int
	Errors    int
	StartedAt time.Time
	EndedAt   time.Time
}

// QueueStat is a queue count for one status.
type QueueStat struct {
	Status string
	Count  int
}

// Store owns the mirror directory and its SQLite database.
type Store struct {
	dir string
	db  *sql.DB
	mu  sync.Mutex // serializes writes; SQLite is single-writer
	now func() time.Time
}

const schema = `
CREATE TABLE IF NOT EXISTS queue (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	entity_type TEXT NOT NULL,
	ref         TEXT NOT NULL,
	url         TEXT NOT NULL DEFAULT '',
	priority    INTEGER NOT NULL DEFAULT 0,
	status      TEXT NOT NULL DEFAULT 'pending',
	attempts    INTEGER NOT NULL DEFAULT 0,
	error       TEXT NOT NULL DEFAULT '',
	created_at  TEXT NOT NULL DEFAULT '',
	updated_at  TEXT NOT NULL DEFAULT '',
	UNIQUE(entity_type, ref)
);
CREATE INDEX IF NOT EXISTS idx_queue_status ON queue(status, priority DESC, id ASC);

CREATE TABLE IF NOT EXISTS visited (
	entity_type TEXT NOT NULL,
	ref         TEXT NOT NULL,
	visited_at  TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (entity_type, ref)
);

CREATE TABLE IF NOT EXISTS baike_articles (
	lemma_id    INTEGER PRIMARY KEY,
	title       TEXT NOT NULL DEFAULT '',
	json        TEXT NOT NULL,
	raw_path    TEXT NOT NULL DEFAULT '',
	fetched_at  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_articles_title ON baike_articles(title);

CREATE TABLE IF NOT EXISTS search_results (
	id         TEXT PRIMARY KEY,
	query      TEXT NOT NULL,
	page       INTEGER NOT NULL DEFAULT 0,
	position   INTEGER NOT NULL DEFAULT 0,
	json       TEXT NOT NULL,
	fetched_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_serp_query ON search_results(query, page, position);

CREATE TABLE IF NOT EXISTS suggest_queries (
	query      TEXT NOT NULL,
	rank       INTEGER NOT NULL DEFAULT 0,
	word       TEXT NOT NULL DEFAULT '',
	fetched_at TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (query, rank)
);

CREATE TABLE IF NOT EXISTS jobs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	kind       TEXT NOT NULL,
	status     TEXT NOT NULL DEFAULT 'running',
	detail     TEXT NOT NULL DEFAULT '',
	processed  INTEGER NOT NULL DEFAULT 0,
	errors     INTEGER NOT NULL DEFAULT 0,
	started_at TEXT NOT NULL DEFAULT '',
	ended_at   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Open opens (creating if needed) the mirror at dir and applies the schema.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("store: empty dir")
	}
	for _, sub := range []string{"", "raw", "export"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("store: mkdir %s: %w", sub, err)
		}
	}
	dsn := "file:" + filepath.Join(dir, "baidu.db") +
		"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}
	db.SetMaxOpenConns(1) // one writer; reads are fast and serialized
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: schema: %w", err)
	}
	return &Store{dir: dir, db: db, now: time.Now}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Dir, RawDir, ExportDir, DBPath expose the on-disk layout.
func (s *Store) Dir() string       { return s.dir }
func (s *Store) RawDir() string    { return filepath.Join(s.dir, "raw") }
func (s *Store) ExportDir() string { return filepath.Join(s.dir, "export") }
func (s *Store) DBPath() string    { return filepath.Join(s.dir, "baidu.db") }

func (s *Store) ts() string { return s.now().UTC().Format(time.RFC3339) }

// Enqueue inserts a queue item if (entity_type, ref) is not already present. It
// returns true when a new row was added. Re-seeding an existing ref is a no-op so
// status and history are preserved.
func (s *Store) Enqueue(it QueueItem) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enqueueLocked(it)
}

func (s *Store) enqueueLocked(it QueueItem) (bool, error) {
	if it.Status == "" {
		it.Status = StatusPending
	}
	now := s.ts()
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO queue
			(entity_type, ref, url, priority, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
		it.EntityType, it.Ref, it.URL, it.Priority, it.Status, now, now,
	)
	if err != nil {
		return false, fmt.Errorf("store: enqueue: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// EnqueueBatch enqueues many items in one transaction. It returns the number of
// new rows added.
func (s *Store) EnqueueBatch(items []QueueItem) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0
	for _, it := range items {
		ok, err := s.enqueueLocked(it)
		if err != nil {
			return added, err
		}
		if ok {
			added++
		}
	}
	return added, nil
}

// Pop claims up to n pending queue items, marking them done is the caller's job
// via Done/Fail. It returns the highest-priority pending rows. Rows stay pending
// while in flight so an interrupted run resumes cleanly; callers coordinate
// dequeueing through a single feeder.
func (s *Store) Pop(n int) ([]QueueItem, error) {
	rows, err := s.db.Query(
		`SELECT id, entity_type, ref, url, priority, status, attempts, error
			FROM queue WHERE status = ?
			ORDER BY priority DESC, id ASC LIMIT ?`,
		StatusPending, n,
	)
	if err != nil {
		return nil, fmt.Errorf("store: pop: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []QueueItem
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.ID, &it.EntityType, &it.Ref, &it.URL,
			&it.Priority, &it.Status, &it.Attempts, &it.Error); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Done marks a queue row done and records the (entity_type, ref) in visited, in
// one transaction.
func (s *Store) Done(it QueueItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: done begin: %w", err)
	}
	now := s.ts()
	if _, err := tx.Exec(
		`UPDATE queue SET status = ?, attempts = attempts + 1, error = '', updated_at = ?
			WHERE id = ?`,
		StatusDone, now, it.ID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("store: done update: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO visited (entity_type, ref, visited_at) VALUES (?, ?, ?)`,
		it.EntityType, it.Ref, now,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("store: done visited: %w", err)
	}
	return tx.Commit()
}

// Fail bumps a row's attempt count and records the error. After 3 attempts the
// row is moved to failed; otherwise it stays pending for a later retry.
func (s *Store) Fail(it QueueItem, cause string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := StatusPending
	if it.Attempts+1 >= 3 {
		status = StatusFailed
	}
	_, err := s.db.Exec(
		`UPDATE queue SET status = ?, attempts = attempts + 1, error = ?, updated_at = ?
			WHERE id = ?`,
		status, cause, s.ts(), it.ID,
	)
	if err != nil {
		return fmt.Errorf("store: fail: %w", err)
	}
	return nil
}

// MarkFailed moves a row straight to failed regardless of attempt count. It is
// used for terminal outcomes a retry cannot fix, such as a walled SERP.
func (s *Store) MarkFailed(it QueueItem, cause string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`UPDATE queue SET status = ?, attempts = attempts + 1, error = ?, updated_at = ?
			WHERE id = ?`,
		StatusFailed, cause, s.ts(), it.ID,
	)
	if err != nil {
		return fmt.Errorf("store: mark failed: %w", err)
	}
	return nil
}

// IsVisited reports whether (entity_type, ref) has been crawled.
func (s *Store) IsVisited(entityType, ref string) (bool, error) {
	row := s.db.QueryRow(
		`SELECT 1 FROM visited WHERE entity_type = ? AND ref = ?`, entityType, ref)
	var one int
	switch err := row.Scan(&one); err {
	case nil:
		return true, nil
	case sql.ErrNoRows:
		return false, nil
	default:
		return false, fmt.Errorf("store: is visited: %w", err)
	}
}

// PendingCount returns how many queue rows are still pending.
func (s *Store) PendingCount() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM queue WHERE status = ?`, StatusPending)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("store: pending count: %w", err)
	}
	return n, nil
}

// QueueStats returns queue counts grouped by status.
func (s *Store) QueueStats() ([]QueueStat, error) {
	rows, err := s.db.Query(
		`SELECT status, COUNT(*) FROM queue GROUP BY status ORDER BY status`)
	if err != nil {
		return nil, fmt.Errorf("store: queue stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []QueueStat
	for rows.Next() {
		var st QueueStat
		if err := rows.Scan(&st.Status, &st.Count); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// ListQueue returns queue rows filtered by status (empty = any), oldest first.
func (s *Store) ListQueue(status string, limit int) ([]QueueItem, error) {
	q := `SELECT id, entity_type, ref, url, priority, status, attempts, error FROM queue`
	var args []any
	if status != "" {
		q += " WHERE status = ?"
		args = append(args, status)
	}
	q += " ORDER BY id ASC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list queue: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []QueueItem
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.ID, &it.EntityType, &it.Ref, &it.URL,
			&it.Priority, &it.Status, &it.Attempts, &it.Error); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ResetFailed moves failed rows back to pending so a later crawl retries them. It
// returns the number requeued.
func (s *Store) ResetFailed() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		`UPDATE queue SET status = ?, error = '', updated_at = ? WHERE status = ?`,
		StatusPending, s.ts(), StatusFailed)
	if err != nil {
		return 0, fmt.Errorf("store: reset failed: %w", err)
	}
	return res.RowsAffected()
}

// UpsertArticle stores a Baike article record.
func (s *Store) UpsertArticle(lemmaID int, title string, js json.RawMessage, rawPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO baike_articles (lemma_id, title, json, raw_path, fetched_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(lemma_id) DO UPDATE SET
				title = excluded.title, json = excluded.json,
				raw_path = excluded.raw_path, fetched_at = excluded.fetched_at`,
		lemmaID, title, string(js), rawPath, s.ts(),
	)
	if err != nil {
		return fmt.Errorf("store: upsert article %d: %w", lemmaID, err)
	}
	return nil
}

// UpsertSearchResult stores a single SERP entry.
func (s *Store) UpsertSearchResult(id, query string, page, position int, js json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO search_results (id, query, page, position, json, fetched_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				query = excluded.query, page = excluded.page,
				position = excluded.position, json = excluded.json,
				fetched_at = excluded.fetched_at`,
		id, query, page, position, string(js), s.ts(),
	)
	if err != nil {
		return fmt.Errorf("store: upsert serp %s: %w", id, err)
	}
	return nil
}

// UpsertSuggest stores one suggest entry.
func (s *Store) UpsertSuggest(query string, rank int, word string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO suggest_queries (query, rank, word, fetched_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(query, rank) DO UPDATE SET
				word = excluded.word, fetched_at = excluded.fetched_at`,
		query, rank, word, s.ts(),
	)
	if err != nil {
		return fmt.Errorf("store: upsert suggest %q: %w", query, err)
	}
	return nil
}

// CreateJob inserts a running job row and returns its id.
func (s *Store) CreateJob(kind, detail string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		`INSERT INTO jobs (kind, status, detail, started_at) VALUES (?, 'running', ?, ?)`,
		kind, detail, s.ts(),
	)
	if err != nil {
		return 0, fmt.Errorf("store: create job: %w", err)
	}
	return res.LastInsertId()
}

// UpdateJob sets the final status and counters of a job.
func (s *Store) UpdateJob(id int64, status string, processed, errCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`UPDATE jobs SET status = ?, processed = ?, errors = ?, ended_at = ? WHERE id = ?`,
		status, processed, errCount, s.ts(), id,
	)
	if err != nil {
		return fmt.Errorf("store: update job %d: %w", id, err)
	}
	return nil
}

// ListJobs returns the most recent jobs, newest first.
func (s *Store) ListJobs(limit int) ([]Job, error) {
	q := `SELECT id, kind, status, detail, processed, errors, started_at, ended_at
		FROM jobs ORDER BY id DESC`
	var args []any
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		var j Job
		var started, ended string
		if err := rows.Scan(&j.ID, &j.Kind, &j.Status, &j.Detail,
			&j.Processed, &j.Errors, &started, &ended); err != nil {
			return nil, err
		}
		j.StartedAt, _ = time.Parse(time.RFC3339, started)
		j.EndedAt, _ = time.Parse(time.RFC3339, ended)
		out = append(out, j)
	}
	return out, rows.Err()
}

// ArticleCount returns the number of stored Baike articles.
func (s *Store) ArticleCount() (int, error)      { return s.count("baike_articles") }
func (s *Store) SearchResultCount() (int, error) { return s.count("search_results") }
func (s *Store) SuggestCount() (int, error)      { return s.count("suggest_queries") }
func (s *Store) VisitedCount() (int, error)      { return s.count("visited") }

func (s *Store) count(table string) (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table) //nolint:gosec // table is a fixed literal
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count %s: %w", table, err)
	}
	return n, nil
}

// WriteRaw gzips data to raw/<entityType>/<shard>/<id>.<ext>.gz and returns the
// path relative to the mirror dir plus the sha256 of the uncompressed bytes. The
// shard is the last two characters of the id, capping directory fan-out.
func (s *Store) WriteRaw(entityType, id, ext string, data []byte) (relPath, hash string, err error) {
	sum := sha256.Sum256(data)
	hash = hex.EncodeToString(sum[:])
	rel := filepath.Join("raw", entityType, shardOf(id), id+"."+ext+".gz")
	abs := filepath.Join(s.dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", "", fmt.Errorf("store: raw mkdir: %w", err)
	}
	f, err := os.Create(abs)
	if err != nil {
		return "", "", fmt.Errorf("store: raw create: %w", err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(data); err != nil {
		_ = gz.Close()
		_ = f.Close()
		return "", "", fmt.Errorf("store: raw write: %w", err)
	}
	if err := gz.Close(); err != nil {
		_ = f.Close()
		return "", "", fmt.Errorf("store: raw gzip close: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", "", fmt.Errorf("store: raw close: %w", err)
	}
	return rel, hash, nil
}

// ReadRaw returns the decompressed bytes of a raw artifact given its relative path.
func (s *Store) ReadRaw(relPath string) ([]byte, error) {
	f, err := os.Open(filepath.Join(s.dir, relPath))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	return io.ReadAll(gz)
}

// SetMeta and GetMeta store and read crawl cursors and counters.
func (s *Store) SetMeta(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("store: set meta %s: %w", key, err)
	}
	return nil
}

// GetMeta returns the value for key, or ok=false if unset.
func (s *Store) GetMeta(key string) (value string, ok bool, err error) {
	row := s.db.QueryRow(`SELECT value FROM meta WHERE key = ?`, key)
	switch err := row.Scan(&value); err {
	case nil:
		return value, true, nil
	case sql.ErrNoRows:
		return "", false, nil
	default:
		return "", false, fmt.Errorf("store: get meta %s: %w", key, err)
	}
}

// ExportJSONL streams the JSON of one entity kind to w, one object per line. kind
// is one of "article", "search", "suggest". It returns the number of rows written.
func (s *Store) ExportJSONL(ctx context.Context, kind string, w io.Writer) (int, error) {
	var query string
	switch kind {
	case "article", "":
		query = `SELECT json FROM baike_articles ORDER BY lemma_id`
	case "search":
		query = `SELECT json FROM search_results ORDER BY query, page, position`
	case "suggest":
		query = `SELECT json_object('query', query, 'rank', rank, 'word', word)
			FROM suggest_queries ORDER BY query, rank`
	default:
		return 0, fmt.Errorf("store: export: unknown kind %q", kind)
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("store: export: %w", err)
	}
	defer func() { _ = rows.Close() }()
	n := 0
	for rows.Next() {
		var js string
		if err := rows.Scan(&js); err != nil {
			return n, err
		}
		if _, err := io.WriteString(w, js+"\n"); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// EachArticle calls fn for every stored Baike article JSON, ordered by lemma id.
// It is the read side used by the Markdown exporter.
func (s *Store) EachArticle(ctx context.Context, fn func(lemmaID int, title string, js []byte) error) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT lemma_id, title, json FROM baike_articles ORDER BY lemma_id`)
	if err != nil {
		return fmt.Errorf("store: each article: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int
		var title, js string
		if err := rows.Scan(&id, &title, &js); err != nil {
			return err
		}
		if err := fn(id, title, []byte(js)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// shardOf returns the two-character shard key for an id (last two characters),
// capping directory fan-out. Short or empty ids fall back to a padded value.
func shardOf(id string) string {
	switch {
	case len(id) >= 2:
		return id[len(id)-2:]
	case len(id) == 1:
		return "0" + id
	default:
		return "00"
	}
}
