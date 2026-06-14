---
title: "Scripting with jq"
description: "Combine baidu with jq for flexible data extraction."
weight: 10
---

`baidu` outputs JSONL when piped, making it simple to process with `jq`.

```bash
# Print just the search words
baidu hot | jq -r '.word'

# Filter to items tagged 新 (new)
baidu hot | jq 'select(.tag == "新") | .word'

# Top 5 movie searches as CSV
baidu hot --tab movie -n 5 -o csv

# Get suggestion words only
baidu suggest "AI" | jq -r '.word'

# Pull a few fields from one Baike article (when reachable; geo-walled)
baidu article 北京 | jq '{title, abstract}'
```
