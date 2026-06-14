// Command baidu is the single-binary command line for baidu-cli. It hands the
// kit App to kit.Run, which builds the CLI, the HTTP API (baidu serve), and the
// MCP server (baidu mcp) from the one operation registry and maps errors to
// stable exit codes.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/baidu-cli/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(kit.Run(ctx, cli.NewApp()))
}
