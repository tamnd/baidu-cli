package baidu_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/baidu-cli/baidu"
)

func TestHot(t *testing.T) {
	// The realtime board nests items a second level deep: cards[].content[]
	// holds groups, and each group carries its own content[] of items.
	payload := map[string]any{
		"data": map[string]any{
			"cards": []any{
				map[string]any{
					"content": []any{
						map[string]any{
							"content": []any{
								map[string]any{
									"isTop": true,
									"word":  "pinned item",
									"url":   "https://m.baidu.com/s?word=pinned+item",
								},
								map[string]any{
									"isTop":      false,
									"index":      1,
									"word":       "hot item",
									"url":        "https://m.baidu.com/s?word=hot+item",
									"hotTag":     "3",
									"newHotName": "",
								},
								map[string]any{
									"isTop":      false,
									"index":      2,
									"word":       "new item",
									"url":        "https://m.baidu.com/s?word=new+item",
									"hotTag":     "1",
									"newHotName": "新",
								},
							},
						},
					},
				},
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer ts.Close()

	cfg := baidu.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0

	client := baidu.NewClient(cfg)
	items, err := client.Hot(context.Background(), "realtime", 0)
	if err != nil {
		t.Fatalf("Hot: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}

	// pinned item: rank 0, no tag
	if items[0].Rank != 0 {
		t.Errorf("items[0].Rank: want 0, got %d", items[0].Rank)
	}
	if items[0].Word != "pinned item" {
		t.Errorf("items[0].Word: want %q, got %q", "pinned item", items[0].Word)
	}
	if items[0].Tag != "" {
		t.Errorf("items[0].Tag: want empty, got %q", items[0].Tag)
	}

	// hot item: rank 2 (index=1 → rank=index+1=2), tag "热" from hotTag="3"
	if items[1].Rank != 2 {
		t.Errorf("items[1].Rank: want 2, got %d", items[1].Rank)
	}
	if items[1].Tag != "热" {
		t.Errorf("items[1].Tag: want %q, got %q", "热", items[1].Tag)
	}

	// new item: rank 3, tag "新" from newHotName
	if items[2].Rank != 3 {
		t.Errorf("items[2].Rank: want 3, got %d", items[2].Rank)
	}
	if items[2].Tag != "新" {
		t.Errorf("items[2].Tag: want %q, got %q", "新", items[2].Tag)
	}
}

func TestHotLimit(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"cards": []any{
				map[string]any{
					"content": []any{
						map[string]any{"isTop": false, "index": 1, "word": "a", "url": ""},
						map[string]any{"isTop": false, "index": 2, "word": "b", "url": ""},
						map[string]any{"isTop": false, "index": 3, "word": "c", "url": ""},
					},
				},
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer ts.Close()

	cfg := baidu.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	client := baidu.NewClient(cfg)

	items, err := client.Hot(context.Background(), "realtime", 2)
	if err != nil {
		t.Fatalf("Hot: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items with limit=2, got %d", len(items))
	}
}

func TestSuggest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"q":"test","p":false,"g":[{"type":"sug","q":"alpha"},{"type":"sug","q":"beta"},{"type":"sug","q":"gamma"}]}`))
	}))
	defer ts.Close()

	cfg := baidu.DefaultConfig()
	cfg.SuggestBaseURL = ts.URL
	cfg.Rate = 0
	client := baidu.NewClient(cfg)

	suggestions, err := client.Suggest(context.Background(), "test")
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if len(suggestions) != 3 {
		t.Fatalf("want 3 suggestions, got %d", len(suggestions))
	}
	if suggestions[0].Rank != 1 || suggestions[0].Word != "alpha" {
		t.Errorf("suggestions[0]: want {1, alpha}, got {%d, %q}", suggestions[0].Rank, suggestions[0].Word)
	}
	if suggestions[1].Rank != 2 || suggestions[1].Word != "beta" {
		t.Errorf("suggestions[1]: want {2, beta}, got {%d, %q}", suggestions[1].Rank, suggestions[1].Word)
	}
	if suggestions[2].Rank != 3 || suggestions[2].Word != "gamma" {
		t.Errorf("suggestions[2]: want {3, gamma}, got {%d, %q}", suggestions[2].Rank, suggestions[2].Word)
	}
}
