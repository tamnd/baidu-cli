// Package cli builds the baidu command tree on top of the baidu library and the
// any-cli/kit framework. The lookup commands are declared once as kit operations
// in the baidu package, so the CLI, the HTTP API (baidu serve), and the MCP
// server (baidu mcp) all derive from one registry. The mirror subsystem is wired
// here as escape-hatch commands, since a stateful crawl does not fit the
// emit-records shape of an operation.
package cli

import (
	"os"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/baidu-cli/baidu"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds
// them onto the resolved config in finalize.
type builder struct {
	userAgent string
	baiduID   string
}

// NewApp assembles the kit App: the baidu domain installs the client factory and
// the lookup operations, this package adds the mirror and version commands, and
// kit provides the CLI, API, and MCP surfaces.
func NewApp() *kit.App {
	b := &builder{}
	id := baidu.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(baidu.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	baidu.Domain{}.Register(app)
	registerMirror(app)
	app.AddCommand(versionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	f.StringVar(&b.userAgent, "user-agent", baidu.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.baiduID, "baiduid", "", "BAIDUID cookie for the web search surface (or $BAIDU_BAIDUID)")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	if b.userAgent != "" {
		c.Extra["user-agent"] = b.userAgent
	}
	id := b.baiduID
	if id == "" {
		id = os.Getenv("BAIDU_BAIDUID")
	}
	if id != "" {
		c.Extra["baiduid"] = id
	}
}
