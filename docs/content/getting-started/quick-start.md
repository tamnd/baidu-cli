---
title: "Quick start"
description: "Run your first baidu command."
weight: 30
---

## Hot search

```bash
# Top 30 realtime hot searches (table at TTY, JSONL piped)
baidu hot

# Top 10 movie searches
baidu hot --tab movie -n 10

# Output as JSON
baidu hot -o json

# Just the URLs
baidu hot -o url
```

## Suggest

```bash
baidu suggest --query golang
baidu suggest -Q "machine learning" -o json
```

## Pipe to jq

```bash
baidu hot | jq '.word'
baidu suggest -Q weather | jq '.word'
```
