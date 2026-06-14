---
title: "Quick start"
description: "Run your first baidu command."
weight: 30
---

## Hot search

```bash
# Hot search board in rank order (table at TTY, JSONL piped)
baidu hot

# Top 10 movie searches
baidu hot --tab movie -n 10

# Output as JSON
baidu hot -o json

# Just the URLs
baidu hot -o url
```

## Suggest

The query is positional now.

```bash
baidu suggest golang
baidu suggest "machine learning" -o json
```

## Web search

```bash
baidu search golang
baidu search golang --pages 2 -o json
```

`search` is walled behind a CAPTCHA. From a datacenter or flagged IP it
bounces and the command exits cleanly (exit code 5) with a hint to pass a real
`--baiduid` cookie. A real BAIDUID improves the odds but is not guaranteed.

## Baike encyclopedia

```bash
# One article by title, id, or title/id
baidu article 北京

# Lemma stubs under a category tag (empty tag walks the built-in tags)
baidu categories 科技
```

`article` and `categories` read the Baike encyclopedia, which is geo-walled.
They answer fully from China IPs and return blocks from elsewhere; off a China
IP they exit cleanly (exit 5 when blocked, exit 3 when genuinely empty) rather
than faking data.

## Mirror Baike locally

The mirror is a CLI-only, resumable crawl of Baike into a local SQLite store.

```bash
baidu seed topics          # queue the built-in seed topics
baidu crawl                # work the queue with a worker pool
baidu export --markdown    # write mirrored articles as Markdown
baidu info                 # mirror location, counts, disk usage
```

The crawler depends on Baike being reachable, so from a non-China IP `crawl`
records each fetch as blocked rather than producing records.

## Serve over HTTP or MCP

```bash
baidu serve                # HTTP, NDJSON; /healthz and /v1/openapi.json
baidu mcp                  # MCP server over stdio
```

## Pipe to jq

```bash
baidu hot | jq '.word'
baidu suggest weather | jq '.word'
```
