---
title: "Building for production"
description: "The base URL, the sub-path problem, and the one rule that solves it."
weight: 10
---

## Build

```bash
tago build
```

That renders the site into `public/`. For production, set the base URL so links
and assets point at the real site:

```bash
tago build --base-url https://docs.example.com/
```

## The sub-path problem

Many sites are not served from a domain root. A GitHub project site lives under
a path, for example `https://username.github.io/project/`. Root-relative links
like `/getting-started/` break there, because the real page is at
`/project/getting-started/`.

tago-doks handles this by linking every page and every asset through `absURL`,
which prepends the full base URL including its path. So when you build with a
sub-path base URL, every link in the output already carries that prefix:

```bash
tago build --base-url https://username.github.io/project/
```

The output has `href="https://username.github.io/project/getting-started/"`,
not `href="/getting-started/"`. Nothing in your content changes; you only change
the base URL you build with.

## One site, two URLs

If you publish the same site at two URLs with different paths, build it once per
URL. That is exactly what a GitHub Pages plus custom-domain setup needs: a
sub-path build for Pages and a root build for the custom domain.

```bash
tago build --base-url https://username.github.io/project/ --output public-pages
tago build --base-url https://docs.example.com/           --output public-domain
```

Each output is self-consistent for its URL. The next page wires both into one
GitHub Actions workflow.

## Local preview

While writing, skip the base URL and use the dev server for live reload:

```bash
tago serve
```
