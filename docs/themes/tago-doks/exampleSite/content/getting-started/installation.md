---
title: "Installation"
description: "Add tago-doks to a tago site as a theme."
weight: 10
---

## Add the theme

Drop the theme into your site under `themes/tago-doks`. The simplest way to keep
it updated is a git submodule:

```bash
git submodule add https://github.com/tamnd/tago-doks themes/tago-doks
```

## Point your config at it

In `tago.toml`, set the theme name and a few parameters:

```toml
title   = "My project"
baseURL = "https://example.com/"
theme   = "tago-doks"
syntaxHighlight = "true"

[params]
github = "https://github.com/me/my-project"
```

## Build

```bash
tago build
```

That writes the site to `public/`. Open it with any static file server, or run
`tago serve` for live reload while you write.
