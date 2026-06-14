package store

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func open(t *testing.T) *Store {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestQueueLifecycle(t *testing.T) {
	st := open(t)

	added, err := st.Enqueue(QueueItem{EntityType: "lemma", Ref: "1", URL: "u1"})
	if err != nil || !added {
		t.Fatalf("Enqueue: added=%v err=%v", added, err)
	}
	// Re-enqueue is a no-op.
	again, _ := st.Enqueue(QueueItem{EntityType: "lemma", Ref: "1"})
	if again {
		t.Error("re-enqueue should not add a new row")
	}

	n, _ := st.PendingCount()
	if n != 1 {
		t.Errorf("PendingCount = %d, want 1", n)
	}

	rows, err := st.Pop(10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("Pop: rows=%d err=%v", len(rows), err)
	}
	if err := st.Done(rows[0]); err != nil {
		t.Fatalf("Done: %v", err)
	}
	visited, _ := st.IsVisited("lemma", "1")
	if !visited {
		t.Error("row should be visited after Done")
	}
	if n, _ := st.PendingCount(); n != 0 {
		t.Errorf("PendingCount after Done = %d, want 0", n)
	}
}

func TestFailAndReset(t *testing.T) {
	st := open(t)
	_, _ = st.Enqueue(QueueItem{EntityType: "serp", Ref: "q|1"})
	rows, _ := st.Pop(1)
	it := rows[0]

	// Three failures move the row to failed.
	for i := 0; i < 3; i++ {
		if err := st.Fail(it, "boom"); err != nil {
			t.Fatalf("Fail: %v", err)
		}
		rows, _ = st.ListQueue("", 10)
		it = rows[0]
	}
	if it.Status != StatusFailed {
		t.Errorf("status = %q, want failed", it.Status)
	}

	n, err := st.ResetFailed()
	if err != nil || n != 1 {
		t.Fatalf("ResetFailed: n=%d err=%v", n, err)
	}
	if n2, _ := st.PendingCount(); n2 != 1 {
		t.Errorf("PendingCount after reset = %d, want 1", n2)
	}
}

func TestUpsertAndExport(t *testing.T) {
	st := open(t)
	art := map[string]any{"lemma_id": 7, "title": "x"}
	js, _ := json.Marshal(art)
	if err := st.UpsertArticle(7, "x", js, ""); err != nil {
		t.Fatalf("UpsertArticle: %v", err)
	}
	if c, _ := st.ArticleCount(); c != 1 {
		t.Errorf("ArticleCount = %d, want 1", c)
	}

	var buf bytes.Buffer
	n, err := st.ExportJSONL(context.Background(), "article", &buf)
	if err != nil || n != 1 {
		t.Fatalf("ExportJSONL: n=%d err=%v", n, err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"lemma_id":7`)) {
		t.Errorf("export missing record: %s", buf.String())
	}
}

func TestRawRoundTrip(t *testing.T) {
	st := open(t)
	data := []byte("<html>hello</html>")
	rel, hash, err := st.WriteRaw("lemma", "12345", "html", data)
	if err != nil {
		t.Fatalf("WriteRaw: %v", err)
	}
	if hash == "" || rel == "" {
		t.Fatal("WriteRaw returned empty rel/hash")
	}
	got, err := st.ReadRaw(rel)
	if err != nil {
		t.Fatalf("ReadRaw: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("round-trip mismatch: %q", got)
	}
}

func TestJobs(t *testing.T) {
	st := open(t)
	id, err := st.CreateJob("crawl", "n=10")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if err := st.UpdateJob(id, StatusDone, 5, 1); err != nil {
		t.Fatalf("UpdateJob: %v", err)
	}
	jobs, err := st.ListJobs(10)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("ListJobs: n=%d err=%v", len(jobs), err)
	}
	if jobs[0].Status != StatusDone || jobs[0].Processed != 5 || jobs[0].Errors != 1 {
		t.Errorf("job = %+v", jobs[0])
	}
}

func TestShardOf(t *testing.T) {
	cases := map[string]string{"12345": "45", "5": "05", "": "00"}
	for in, want := range cases {
		if got := shardOf(in); got != want {
			t.Errorf("shardOf(%q) = %q, want %q", in, got, want)
		}
	}
}
