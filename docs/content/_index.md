---
title: "baidu"
description: "A command line for Baidu."
heroTitle: "Baidu, from the command line"
heroLead: "A command line for Baidu. One pure-Go binary, no API key. Reads the hot board, suggest, web search, and the Baike encyclopedia, and serves the same operations over HTTP and MCP."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`baidu` reads Baidu through its public surfaces: the hot search board, the
typeahead suggest API, web search results, and the Baike (百度百科)
encyclopedia. No API key is required. It prints clean records you can read at
a terminal or pipe into the next tool, and it serves the same read operations
over HTTP (`baidu serve`) and MCP (`baidu mcp`).

```bash
baidu hot                       # realtime hot search board, in rank order
baidu hot --tab movie -n10      # top 10 movie searches
baidu suggest golang            # typeahead suggestions for a query
baidu hot -o url                # just the URLs
```

Output is a table when you are at a terminal and JSONL when you pipe, so
`baidu hot | jq` works with no flags.

Some surfaces are walled by IP and region. `hot` and `suggest` are open
anywhere; `search` is CAPTCHA-walled (best effort) and Baike (`article`,
`categories`) is geo-walled (works from China IPs). The tool detects each
block and exits cleanly instead of faking data.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/) for prebuilt
  binaries, packages, and one-line installers.
- Doing a specific job? The [guides](/guides/) are task-oriented walkthroughs.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
