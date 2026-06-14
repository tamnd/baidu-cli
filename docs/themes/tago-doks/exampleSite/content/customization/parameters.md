---
title: "Parameters"
description: "Site and page parameters the theme understands."
weight: 10
---

## Site parameters

Set these under `[params]` in `tago.toml`.

| Parameter | Purpose |
| --- | --- |
| `github` | URL of the repository. Shows the GitHub icon in the navbar. |
| `heroTitle` | Headline on the home page. Falls back to the site title. |
| `heroLead` | Sub-headline on the home page. Falls back to the description. |
| `heroPrimaryURL` | Target of the primary call-to-action button. |
| `heroPrimaryText` | Label for that button. Defaults to "Get started". |

## Section parameters

Set these in a section's `_index.md` front matter.

| Parameter | Purpose |
| --- | --- |
| `weight` | Order of the group in the sidebar and on the home page. |
| `featured` | When `true`, the section appears in the home page feature grid. |
| `linkTitle` | Short title used in the sidebar and breadcrumbs. |

## Page parameters

| Parameter | Purpose |
| --- | --- |
| `weight` | Order of the page within its section. |
| `description` | Lead paragraph under the title, also used for search and cards. |
| `linkTitle` | Short title used in the sidebar, pager, and breadcrumbs. |
