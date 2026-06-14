# baidu-cli

A command line for Baidu. One pure-Go binary, no API key.

```bash
baidu hot                          # realtime hot search board, in rank order
baidu hot --tab movie -n 10        # top 10 movie searches
baidu suggest golang               # typeahead suggestions for a query
baidu hot -o json                  # JSON output
baidu hot | jq                     # JSONL piped to jq
```

`baidu` reads Baidu (百度) through its public surfaces: the hot search board,
the typeahead suggest API, web search results, and the Baike (百度百科)
encyclopedia. It returns records as a table, JSON, JSONL, CSV, TSV, or URLs,
and serves the same operations over HTTP and MCP. Output is a table at a
terminal and JSONL when piped, so `baidu hot | jq` works with no flags.

## Install

### Homebrew

```bash
brew install tamnd/tap/baidu
```

### Pre-built binaries

Download from [Releases](https://github.com/tamnd/baidu-cli/releases).

### Go

```bash
go install github.com/tamnd/baidu-cli/cmd/baidu@latest
```

### Docker

```bash
docker run --rm ghcr.io/tamnd/baidu:latest hot
```

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
| `serve` | Serve the operations over HTTP (NDJSON) |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print version information |

The suggest query is positional now (`baidu suggest golang`), not a flag.

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

## License

Apache-2.0. `baidu-cli` is an independent tool and is not affiliated with
Baidu, Inc.
