---
title: "CLI reference"
description: "Every flag and subcommand for baidu."
weight: 10
---

`baidu` reads Baidu through its public surfaces: the hot search board, the
typeahead suggest API, web search results, and the Baike (百度百科)
encyclopedia. No API key is required. Read operations also work over HTTP
(`baidu serve`) and MCP (`baidu mcp`).

## Global flags

These apply to every command. Output is a table at a terminal and JSONL when
piped.

```
  -o, --output string      output: auto|table|markdown|json|jsonl|csv|tsv|url|raw (default "auto")
      --fields strings     comma-separated columns to include
      --no-header          omit the header row in table/csv/tsv
      --template string    Go text/template applied per record
  -n, --limit int          stop after N records (0 = no limit)
  -q, --quiet              suppress progress on stderr
      --rate duration      minimum delay between requests
      --timeout duration   per-request timeout
      --retries int        retry attempts
      --user-agent string  User-Agent sent with each request
      --baiduid string     BAIDUID cookie (or $BAIDU_BAIDUID)
      --color              colorize table output
      --db string          tee records into a store
```

## Read commands

### baidu hot

The Baidu hot search board, in rank order.

```
Usage:
  baidu hot [flags]

Flags:
  -t, --tab string   board tab: realtime|novel|movie|teleplay|car (default "realtime")
```

Fields: `rank`, `word`, `tag`, `url`

### baidu suggest

Fast typeahead suggestions for a query. The query is positional.

```
Usage:
  baidu suggest <query> [flags]
```

Fields: `rank`, `word`

### baidu search

Web search results (walled, best effort).

```
Usage:
  baidu search <query> [flags]

Flags:
      --pages int        number of result pages (default 1)
      --baiduid string   BAIDUID cookie (or $BAIDU_BAIDUID)
```

Fields: `id`, `query`, `page`, `position`, `url`, `display_url`, `title`,
`snippet`, `tpl`, `is_ad`, `fetched_at`

`search` is walled behind a CAPTCHA. From a datacenter or flagged IP it
bounces; the command detects the block and exits with code 5 (rate limited)
and a hint to pass a real `--baiduid`, rather than emitting a CAPTCHA page as
data. A real BAIDUID improves the odds but is not guaranteed.

### baidu article

Full detail for one Baike encyclopedia article. `ref` is a lemma id, a title,
or `title/id`. Returns a single record.

```
Usage:
  baidu article <ref> [flags]
```

Fields include `lemma_id`, `url`, `title`, `subtitle`, `abstract`,
`body_markdown`, `infobox`, `sections`, `categories`, `tags`, `related_ids`,
`images`, plus editor and version metadata.

`article` is geo-walled. It answers fully from China IPs and returns blocks
from elsewhere (HTTP 403 on the article HTML). A block reads as exit 5, a
genuinely empty result as exit 3.

### baidu categories

List lemma stubs under Baike category tags. An empty tag walks the 16 built-in
tags.

```
Usage:
  baidu categories <tag> [flags]
```

Fields: `lemma_id`, `title`, `category`

Geo-walled like `article`: blocks read as exit 5, empty results as exit 3.

## Mirror commands

The mirror is a CLI-only escape hatch: a stateful, resumable crawl of Baike
into a local pure-Go SQLite store. It is not exposed over HTTP or MCP. The
crawler depends on Baike being reachable, so from a non-China IP `crawl`
records each fetch as blocked rather than producing records.

### baidu seed topics

Seed the built-in Baike seed topics (or category tags) into the crawl queue.

```
Usage:
  baidu seed topics [flags]
```

### baidu seed url

Seed one or more explicit topics or lemma ids.

```
Usage:
  baidu seed url <ref>... [flags]
```

### baidu seed list

Seed topics or lemma ids from a file, one per line.

```
Usage:
  baidu seed list <file> [flags]
```

### baidu crawl

Crawl pending queue items into the mirror.

```
Usage:
  baidu crawl [flags]

Flags:
      --workers int      worker count (default 4)
      --retry-failed     also retry rows that previously failed
      --data string      mirror dir (or $BAIDU_DATA, default $HOME/data/baidu)
```

### baidu export

Export mirrored records as JSONL or Markdown.

```
Usage:
  baidu export [flags]

Flags:
      --kind string   record kind: article|search|suggest (default "article")
      --out string    output directory
      --markdown      write Markdown instead of JSONL
```

### baidu info

Show mirror location, counts, and disk usage.

```
Usage:
  baidu info [flags]
```

### baidu queue

Inspect queue rows by status.

```
Usage:
  baidu queue [flags]

Flags:
      --status string   filter by status: pending|done|failed
```

### baidu jobs

Show the crawl job log, newest first.

```
Usage:
  baidu jobs [flags]
```

### baidu reset-failed

Requeue failed queue rows for another crawl.

```
Usage:
  baidu reset-failed [flags]
```

## Server commands

### baidu serve

Serve the read operations over HTTP (NDJSON). Endpoints include `/healthz` and
`/v1/openapi.json`.

```
Usage:
  baidu serve [flags]
```

### baidu mcp

Run as an MCP server over stdio.

```
Usage:
  baidu mcp [flags]
```

### baidu version

Print version information.

```
Usage:
  baidu version [flags]
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | ok |
| 1 | generic error |
| 2 | usage |
| 3 | no results |
| 4 | needs auth |
| 5 | rate limited / blocked |
| 6 | not found |
| 7 | unsupported |
| 8 | network |
