---
title: "Search"
description: "Client-side full-text search, and how the index is built."
weight: 10
---

Search runs entirely in the browser. There is no server and no external service.

## How it works

When tago builds the site, it writes a search index file,
`en.search-data.json`, with one entry per page: the page's title, its link, and
its text capped to a few hundred characters. The theme loads
[FlexSearch](https://github.com/nextapps-de/flexsearch) and indexes that file in
the browser. As the reader types in the navbar box, matches appear in a dropdown.

There is also a full-page search view. Create a page with the search layout and
it renders the same results in the page body:

```markdown
---
title: "Search"
layout: "search"
ExcludeSearch: true
---
```

## What is indexed

Every page is indexed except those that set `ExcludeSearch: true` in front
matter. Set it on the search page itself so it does not list itself as a result.
The indexed text is the rendered page content, so headings and prose are
searchable; front matter and code are weighted by their place in the body.

## Search and sub-path deploys

The index links each result by its site-root path, without any deploy sub-path.
When the site is served under a sub-path, for example `username.github.io/project/`,
the theme reads the configured base URL and prepends its path to each result
link, so search results resolve correctly in both a root deploy and a sub-path
deploy. You do not configure this; it follows the base URL you build with. See
[building for production](/deployment/building/) for how the base URL is set.
