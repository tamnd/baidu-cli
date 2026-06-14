package baidu

import (
	"context"
	"errors"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// Domain is the baidu driver for the any-cli/kit framework. It declares the
// lookup operations once, and kit derives the CLI subcommands, the HTTP API
// routes, and the MCP tools from that single registry. The same Domain powers
// the standalone baidu binary and a multi-domain host that mounts it.
type Domain struct{}

func init() { kit.Register(Domain{}) }

// Info describes the domain: its URI scheme, the hosts it owns, and the identity
// the standalone binary reuses for help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:  "baidu",
		Aliases: []string{"bd"},
		Hosts: []string{
			"baidu.com", "www.baidu.com",
			"top.baidu.com", "baike.baidu.com", "suggestion.baidu.com",
		},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the baidu CLI, shared by the domain and
// the standalone composition root so help and version read the same everywhere.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "baidu",
		Short:  "Crawl Baidu (百度) hot search, suggest, web search, and Baike into structured records",
		Long: `baidu reads Baidu (百度) through its public surfaces: the hot search
board, the typeahead suggest API, web search results, and the Baike
(百度百科) encyclopedia. No API key is required. It returns records as
a table, JSON, JSONL, CSV, TSV, or URLs, and serves the same
operations over HTTP and MCP.

The hot board and suggest are open from any IP. Web search is walled
(CAPTCHA) and best effort. Baike (article, categories) is geo-walled:
it answers from China IPs but returns blocks from elsewhere, which the
tool reports as a clean "blocked" exit rather than a crash.

baidu is an independent tool and is not affiliated with Baidu, Inc.`,
		Site: "https://www.baidu.com",
		Repo: "https://github.com/tamnd/baidu-cli",
	}
}

// Register installs the client factory and every lookup operation on the app.
// The mirror subsystem is wired as escape-hatch commands by the cli package,
// since it is CLI-only (stateful crawl, not the emit-records shape).
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("read", "Look up Baidu records")

	kit.Handle(app, kit.OpMeta{
		Name: "hot", Group: "read",
		Summary: "The Baidu hot search board, in rank order",
		URIType: "hot",
	}, hot)

	kit.Handle(app, kit.OpMeta{
		Name: "suggest", Group: "read",
		Summary: "Fast typeahead suggestions for a query",
		URIType: "suggestion",
		Args:    []kit.Arg{{Name: "query", Help: "search term to get suggestions for"}},
	}, suggest)

	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "read",
		Summary: "Web search results (walled, best effort)",
		URIType: "result",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, search)

	kit.Handle(app, kit.OpMeta{
		Name: "article", Group: "read", Single: true,
		Summary: "Show full detail for one Baike encyclopedia article",
		URIType: "article", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "lemma id, title, or title/id"}},
	}, article)

	kit.Handle(app, kit.OpMeta{
		Name: "categories", Group: "read",
		Summary: "List lemma stubs under Baike category tags",
		URIType: "lemma",
		Args:    []kit.Arg{{Name: "tag", Help: "category tag (empty walks all 16 built-in tags)"}},
	}, categories)
}

// newClient builds the baidu HTTP client from the resolved kit config. It is the
// factory kit calls once per run; the value is injected into handler fields
// tagged kit:"inject".
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto a baidu.Config and returns a
// client. The mirror commands reuse it so the CLI surface and the crawler share
// one notion of rate, timeout, retries, User-Agent, and BAIDUID.
func ClientFromConfig(cfg kit.Config) *Client {
	dc := DefaultConfig()
	if cfg.Rate > 0 {
		dc.Rate = cfg.Rate
	}
	if cfg.Retries >= 0 {
		dc.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		dc.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		dc.UserAgent = ua
	}
	if id := cfg.Extra["baiduid"]; id != "" {
		dc.BaiduID = id
	}
	return NewClient(dc)
}

// Defaults seeds the framework baseline with baidu's own values, so an unset
// --rate or --timeout uses the baidu default rather than the generic kit one.
// It is passed to kit.New via kit.WithDefaults.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Rate
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// mapErr translates a library error into a kit error so the exit code matches
// the rest of the fleet: a missing lemma reads as "no results" (exit 3); a
// walled surface — a CAPTCHA'd SERP or a geo-walled Baike card/article — reads
// as "rate limited" (exit 5) with a hint; everything else falls through to the
// generic failure (exit 1).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return errs.NoResults("not found")
	}
	if errors.Is(err, ErrBlocked) {
		return errs.RateLimited("Baidu declined the request (CAPTCHA, rate limit, or a region/IP wall). The web search wants a real BAIDUID cookie (--baiduid or $BAIDU_BAIDUID); Baike (article/categories) is geo-walled and may need a China IP")
	}
	return err
}

// limitOr returns the operator's --limit when set, else the command's own
// default fetch count. kit also caps emission at --limit, so this only bounds
// how much the client fetches up front.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
