---
title: "baidu"
description: "A command line for Baidu."
heroTitle: "Baidu, from the command line"
heroLead: "A command line for Baidu. One pure-Go binary, no API key, output that pipes into the rest of your tools."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`baidu` reads Baidu through the open hot search and suggest APIs, both
key-free, and prints clean records you can read at a terminal or pipe into
the next tool.

```bash
baidu hot                       # realtime hot search (top 30)
baidu hot --tab movie -n10      # top 10 movie searches
baidu suggest --query golang    # 10 autocomplete suggestions
baidu hot -o url                # just the URLs
```

Output is a table when you are at a terminal and JSONL when you pipe, so
`baidu hot | jq` works with no flags.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/) for prebuilt
  binaries, packages, and one-line installers.
- Doing a specific job? The [guides](/guides/) are task-oriented walkthroughs.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
