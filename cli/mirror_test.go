package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/baidu-cli/baidu"
)

func TestDataPathOf(t *testing.T) {
	if got := dataPathOf("/explicit"); got != "/explicit" {
		t.Errorf("explicit dir = %q, want /explicit", got)
	}
	t.Setenv("BAIDU_DATA", "/from/env")
	if got := dataPathOf(""); got != "/from/env" {
		t.Errorf("env dir = %q, want /from/env", got)
	}
	t.Setenv("BAIDU_DATA", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "data", "baidu")
	if got := dataPathOf(""); got != want {
		t.Errorf("default dir = %q, want %q", got, want)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		512:     "512 B",
		1024:    "1.0 KiB",
		1 << 20: "1.0 MiB",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestReadRefList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "refs.txt")
	if err := os.WriteFile(path, []byte("人工智能\n# comment\n\n12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err := readRefList(path)
	if err != nil {
		t.Fatalf("readRefList: %v", err)
	}
	if len(refs) != 2 || refs[0] != "人工智能" || refs[1] != "12345" {
		t.Errorf("refs = %v", refs)
	}
}

func TestArticleMarkdown(t *testing.T) {
	art := &baidu.BaikeArticle{
		Title:        "人工智能",
		Subtitle:     "AI",
		Abstract:     "the study of intelligent agents",
		Infobox:      []baidu.InfoboxItem{{Name: "外文名", Value: "AI"}},
		BodyMarkdown: "## 历史\n\nbody text",
		URL:          "https://baike.baidu.com/item/x/1",
	}
	md := articleMarkdown(art)
	for _, want := range []string{"# 人工智能", "*AI*", "外文名", "## 历史", art.URL} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}
