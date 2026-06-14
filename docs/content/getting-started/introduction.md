---
title: "Introduction"
description: "How baidu-cli is put together."
weight: 10
---

`baidu` is a single Go binary built on the any-cli/kit framework. It reads
Baidu through its public surfaces and prints clean structured records you can
pipe into your tools. No API key is required.

![baidu reading the hot board into a table and piping suggest through jq](/demo.gif)

## Read operations, three ways

The read operations are defined once and exposed three ways:

- on the CLI, as `baidu hot`, `baidu suggest`, `baidu search`,
  `baidu article`, and `baidu categories`;
- over HTTP, with `baidu serve` (NDJSON responses; endpoints include
  `/healthz` and `/v1/openapi.json`);
- over MCP, with `baidu mcp` (an MCP server over stdio).

The same code answers all three, so a record you get on the CLI is the same
record you get over HTTP or MCP.

## Surfaces

- `hot` reads the hot search board, in rank order, across the realtime, novel,
  movie, teleplay, and car tabs.
- `suggest` calls the typeahead suggest API at `www.baidu.com/sugrec`, which
  returns clean UTF-8 JSON.
- `search` reads web search results.
- `article` and `categories` read the Baike (百度百科) encyclopedia: full
  article detail (abstract, body, infobox, sections, related-id link graph)
  and lemma stubs under category tags.

## The mirror

The mirror subsystem is a CLI-only escape hatch for collecting Baike at scale.
It is a stateful, resumable crawl into a local pure-Go SQLite store:
`seed` queues topics or lemma ids, `crawl` works the queue with a worker pool,
and `export` writes the mirrored records as JSONL or Markdown. `info`, `queue`,
`jobs`, and `reset-failed` inspect and manage the run. The mirror is not
exposed over HTTP or MCP.

## Output

Every read command returns records. At a terminal the default format is a
table; piped, it is JSONL. Pass `-o json`, `-o csv`, `-o tsv`, `-o url`, or
`-o markdown` to change format explicitly, and `--fields` or `--template` to
shape what each record shows.

## Open, walled, geo-walled

Baidu does not serve every surface the same way to every IP, and the tool is
honest about it:

- `hot` and `suggest` are open from any IP, no key.
- `search` is walled behind a CAPTCHA. From a datacenter or flagged IP it
  bounces; the command detects the block and exits cleanly (exit code 5, rate
  limited) with a hint to pass a real `--baiduid` cookie, rather than emitting
  a CAPTCHA page as data. A real BAIDUID improves the odds but is not
  guaranteed. Best effort.
- `article` and `categories` (Baike) are geo-walled. They answer fully from
  China IPs and return blocks from elsewhere. The tool recognises each block
  signal and degrades to a clean exit: a block reads as exit 5, a genuinely
  empty result as exit 3. It never crashes or fakes data. The Baike wall is at
  the IP/region layer, not an auth wall, so no cookie or token the client
  could send will open it (a China IP does).

## Not affiliated

`baidu-cli` is an independent tool and is not affiliated with Baidu, Inc.
