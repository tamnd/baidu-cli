---
title: "Introduction"
description: "What tago-doks is, what it gives you, and what it does not need."
weight: 5
---

tago-doks is a documentation theme for [tago](https://github.com/tamnd/tago),
the Hugo-compatible static site generator. It reproduces the look and feel of
[Doks](https://getdoks.org): a left sidebar grouped by section, a right-hand
table of contents that tracks your scroll, full-text search, and a dark mode
that remembers your choice.

## What you get

- A sidebar built from your content folders, ordered by `weight`.
- A table of contents generated from each page's headings, with scroll-spy.
- Client-side search over every page, with a navbar dropdown and a full-page
  results view.
- Light and dark themes, chosen from the operating system and overridable with a
  toggle that persists.
- Code blocks with a copy button and optional syntax highlighting.
- A home page with a hero and a card per section.

## What it does not need

No Node, no npm, no PostCSS, no esbuild. The original Doks is a Hugo theme that
depends on all of those. tago-doks is an independent reimplementation in tago's
template language plus one prebuilt stylesheet and a small script. You install
the theme, write Markdown, and run `tago build`.

## How it relates to Doks

tago-doks copies the design, not the code. It carries no Doks source. Doks is
MIT licensed and so is tago-doks. The [credits](https://github.com/tamnd/tago-doks#credits)
in the repository spell this out.

When you are ready, [install the theme](/getting-started/installation/) and walk
through the [quick start](/getting-started/quick-start/).
