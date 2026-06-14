---
title: "Introduction"
description: "How baidu-cli is put together."
weight: 10
---

`baidu` is a single Go binary that reads Baidu through two open, key-free
APIs and prints clean structured records you can pipe into your tools.

## APIs

`baidu hot` calls `top.baidu.com/api/board` (no authentication). `baidu suggest`
calls `suggestion.baidu.com/su` (a JSONP endpoint, also unauthenticated).

## Output

Every command returns records. At a terminal the default format is a table;
piped, it is JSONL. Pass `-o json`, `-o csv`, or `-o tsv` to change format
explicitly.

## Not affiliated

`baidu-cli` is an independent tool and is not affiliated with Baidu, Inc.
