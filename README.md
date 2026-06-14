# baidu

[![CI](https://github.com/tamnd/baidu-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/baidu-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tamnd/baidu-cli)](https://github.com/tamnd/baidu-cli/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/tamnd/baidu-cli.svg)](https://pkg.go.dev/github.com/tamnd/baidu-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/tamnd/baidu-cli)](https://goreportcard.com/report/github.com/tamnd/baidu-cli)
[![License](https://img.shields.io/github/license/tamnd/baidu-cli)](./LICENSE)

A command line for Baidu. `baidu` reads public Baidu data and prints clean,
pipeable records. One pure-Go binary, no API key.

[Install](#install) • [Commands](#commands) • [Usage](#usage) • [Anti-bot](#anti-bot) • [Serve](#serve-it) • [Docs](https://baidu-cli.tamnd.com)

![baidu reading the hot board into a table and piping suggest through jq](docs/static/demo.gif)

`baidu` reads Baidu (百度) through its public surfaces: the hot search board,
the typeahead suggest API, web search results, and the Baike (百度百科)
encyclopedia. It returns records as a table, JSON, JSONL, CSV, TSV, or URLs,
and serves the same read operations over HTTP and MCP. Output is a table at a
terminal and JSONL when piped, so `baidu hot | jq` works with no flags.

`baidu` is an independent tool. It is not affiliated with, authorized, or
endorsed by Baidu, Inc.

## Install

```bash
go install github.com/tamnd/baidu-cli/cmd/baidu@latest
```

Or with Homebrew:

```bash
brew install tamnd/tap/baidu
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/baidu-cli/releases),
or run the container image:

```bash
docker run --rm ghcr.io/tamnd/baidu:latest hot
```

Shell completion is built in: `baidu completion bash|zsh|fish|powershell`.

## Commands

Read operations work from the CLI, over HTTP (`baidu serve`), and over MCP
(`baidu mcp`).

| Command | Description |
|---|---|
| `hot` | Baidu hot search board, in rank order (`-t/--tab` realtime\|novel\|movie\|teleplay\|car) |
| `suggest <query>` | Typeahead suggestions for a query |
| `search <query>` | Web search results (walled, best effort) |
| `article <ref>` | Full detail for one Baike encyclopedia article |
| `categories <tag>` | Lemma stubs under Baike category tags |

The mirror subsystem is a CLI-only escape hatch: a stateful, resumable crawl
of Baike into a local SQLite store. It is not exposed over HTTP or MCP.

| Command | Description |
|---|---|
| `seed topics` | Seed the built-in Baike seed topics (or category tags) into the queue |
| `seed url <ref>...` | Seed explicit topics or lemma ids |
| `seed list <file>` | Seed topics or lemma ids from a file, one per line |
| `crawl` | Crawl pending queue items into the mirror |
| `export` | Export mirrored records as JSONL or Markdown |
| `info` | Show mirror location, counts, and disk usage |
| `queue` | Inspect queue rows by status |
| `jobs` | Show the crawl job log, newest first |
| `reset-failed` | Requeue failed queue rows for another crawl |

| Command | Description |
|---|---|
| `serve` | Serve the read operations over HTTP (NDJSON) |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print version information |

## Usage

```bash
baidu hot                          # realtime hot search board, in rank order
baidu hot --tab movie -n 10        # top 10 movie searches
baidu hot --fields rank,word,tag   # pick the columns
baidu suggest golang               # typeahead suggestions for a query
baidu hot -o json                  # JSON output
baidu hot -o url                   # just the URLs
baidu hot | jq                     # JSONL piped to jq
```

Output is a table when you are at a terminal and JSONL when you pipe, so the
last line works with no flags. Every command takes the same global flags
(`-o/--output`, `--fields`, `-n/--limit`, `--rate`, `--timeout`, and more); see
the [CLI reference](https://baidu-cli.tamnd.com/reference/cli/) for the full
surface.

## Anti-bot

Baidu walls some surfaces by IP and region. The tool is honest about this: it
recognises each block signal and exits cleanly rather than emitting a CAPTCHA
page or faking data.

- **Open** anywhere, no key: `hot` and `suggest`.
- **Walled** behind a CAPTCHA: `search`. From a datacenter or flagged IP it
  bounces, and the command exits with code 5 (rate limited) and a hint to pass
  a real `--baiduid` cookie. A real BAIDUID improves the odds but is not
  guaranteed. Best effort.
- **Geo-walled**: `article` and `categories` (Baike). They answer fully from
  China IPs and return blocks from elsewhere. A block reads as exit 5, a
  genuinely empty result as exit 3. The wall is at the IP/region layer, not an
  auth wall, so no cookie or token the client could send will open it. The
  mirror crawler depends on Baike being reachable, so from a non-China IP
  `crawl` records each fetch as blocked rather than producing records.

## Serve it

The read operations are also an HTTP API and an MCP server, backed by the same
code as the CLI.

```bash
baidu serve                 # HTTP on :8080, NDJSON; /healthz and /v1/openapi.json
baidu mcp                   # MCP server over stdio
```

## Docs

Full documentation lives at [baidu-cli.tamnd.com](https://baidu-cli.tamnd.com).

## License

Apache-2.0. See [LICENSE](./LICENSE). `baidu` is an independent tool and is not
affiliated with Baidu, Inc.
