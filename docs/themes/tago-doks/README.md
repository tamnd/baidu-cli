# tago-doks

A clean documentation theme for [tago](https://github.com/tamnd/tago), styled
after [Doks](https://getdoks.org). It gives a tago site the layout people expect
from a modern docs portal: a left sidebar grouped by section, a right-hand table
of contents that tracks your scroll position, full-text search, and a dark mode
that remembers the reader's choice.

Everything renders from Go templates and one prebuilt stylesheet. There is no
Node toolchain to install and no asset pipeline to run.

## Features

- Sidebar navigation grouped by section, ordered by `weight`.
- Right-hand table of contents built from the page headings, with scroll-spy.
- Full-text search over tago's generated index, using FlexSearch.
- Light and dark themes, chosen from `prefers-color-scheme` and remembered in
  `localStorage`.
- Copy buttons on code blocks.
- Works under a sub-path base URL (for example `https://user.github.io/project/`)
  because every internal link and asset reference uses `absURL`.
- Responsive layout with a slide-in sidebar on small screens.

## Requirements

- tago 0.1.0 or newer.

## Install

Add the theme to your site under `themes/tago-doks`, most easily as a submodule:

```bash
git submodule add https://github.com/tamnd/tago-doks themes/tago-doks
```

Point your `tago.toml` at it:

```toml
title   = "My project"
baseURL = "https://example.com/"
theme   = "tago-doks"
syntaxHighlight = "true"

[params]
github = "https://github.com/me/my-project"
```

Build with `tago build`, or `tago serve` for live reload.

## Content structure

Each top-level folder under `content/` becomes a sidebar group. Give it an
`_index.md` with a `title` and a `weight`. Pages inside the folder become
entries in that group, ordered by their own `weight`.

```
content/
  _index.md                 home page
  getting-started/
    _index.md               group: "Getting started", weight 10
    installation.md         weight 10
    quick-start.md          weight 20
  search.md                 layout: search
```

See `exampleSite/` for a complete, buildable example.

## Parameters

Set under `[params]` in `tago.toml`:

| Parameter | Purpose |
| --- | --- |
| `github` | Repository URL. Shows the GitHub icon in the navbar. |
| `heroTitle` | Home page headline. Falls back to the site title. |
| `heroLead` | Home page sub-headline. Falls back to the description. |
| `heroPrimaryURL` | Target of the primary button on the home page. |
| `heroPrimaryText` | Label for that button. Defaults to "Get started". |

Section front matter understands `weight`, `featured` (show in the home feature
grid), and `linkTitle`. Page front matter understands `weight`, `description`,
and `linkTitle`.

## Search page

Add a page with `layout: "search"` to get a full search page. The navbar search
box works on every page once the index exists. tago writes the index to
`en.search-data.json` during the build.

## Credits

This theme reproduces the look and feel of **Doks** by Thulite. Doks is the
original Hugo theme; tago-doks is an independent reimplementation in tago's
template language and does not include Doks source code.

- Doks: https://getdoks.org
- Doks repository: https://github.com/thulite/doks
- Doks is licensed under the MIT License, Copyright (c) 2020-2026 Thulite.

Thanks to the Doks authors for the design this theme is modeled on.

## License

[MIT](LICENSE), the same license as Doks. Copyright (c) 2026 Tam Nguyen Duc.
