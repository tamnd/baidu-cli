---
title: "Front matter"
description: "Every front matter field the theme reads, and what it controls."
weight: 20
---

Front matter is the block between the `---` lines at the top of a Markdown file.
The theme reads these fields. Anything else is ignored, so you can keep your own
fields alongside them.

## On any page

| Field | Controls |
| --- | --- |
| `title` | The page heading and the sidebar entry. |
| `description` | The lead paragraph under the title, and the text used on home-page cards and in search results. |
| `weight` | The order of the page within its section. |
| `linkTitle` | A shorter title used in the sidebar, the pager, and breadcrumbs when the full title is long. |

## On a section `_index.md`

| Field | Controls |
| --- | --- |
| `title` | The sidebar group heading. |
| `weight` | The order of the group in the sidebar and on the home page. |
| `featured` | When `true`, the section gets a card on the home page. |
| `linkTitle` | A short group title for the sidebar. |

## Special pages

| Field | Controls |
| --- | --- |
| `layout` | Selects a layout other than the default. Set `layout: "search"` on the search page so it renders the full-page results view. |
| `ExcludeSearch` | When `true`, the page is left out of the search index. Use it on the search page itself. |

## A complete example

A section index:

```markdown
---
title: "Authoring"
linkTitle: "Authoring"
description: "How content becomes pages."
weight: 20
featured: true
---
```

A page:

```markdown
---
title: "Sections and pages"
description: "Folders become groups, files become pages."
weight: 10
---
```

The search page:

```markdown
---
title: "Search"
layout: "search"
ExcludeSearch: true
---
```
