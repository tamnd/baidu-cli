---
title: "CLI reference"
description: "Every flag and subcommand for baidu."
weight: 10
---

## Global flags

```
  -o, --output string     output: table|json|jsonl|csv|tsv|url|raw (default "auto")
      --fields strings    comma-separated columns to include
      --no-header         omit the header row in table/csv/tsv
      --template string   Go text/template applied per record
  -n, --limit int         limit number of records (0 = command default)
  -q, --quiet             suppress progress on stderr
      --delay duration    minimum spacing between requests (default 200ms)
      --timeout duration  per-request timeout (default 15s)
      --retries int       retry attempts on 429/5xx (default 3)
      --user-agent string User-Agent sent with each request
```

## baidu hot

Fetch the Baidu hot search board.

```
Usage:
  baidu hot [flags]

Flags:
      --tab string   board tab: realtime|novel|movie|teleplay|car (default "realtime")
```

Fields: `rank`, `word`, `tag`, `url`

## baidu suggest

Fetch Baidu search suggestions.

```
Usage:
  baidu suggest [flags]

Flags:
  -Q, --query string   search term to get suggestions for (required)
```

Fields: `rank`, `word`

## baidu version

Print version information.

```
Usage:
  baidu version [flags]

Flags:
      --short   print just the version number
```
