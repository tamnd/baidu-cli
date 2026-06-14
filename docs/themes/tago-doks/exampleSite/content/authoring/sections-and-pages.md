---
title: "Sections and pages"
description: "How folders become sidebar groups and Markdown files become pages."
weight: 10
---

The sidebar is built from your content folders. You do not configure it by hand.

## A folder is a section

Every top-level folder under `content/` becomes a group in the sidebar. The
group needs an `_index.md` to give it a title and a position:

```markdown
---
title: "Authoring"
weight: 20
---
```

The `weight` sets the order of the group in the sidebar. Lower weights come
first. Groups without a weight sort after the weighted ones.

## A Markdown file is a page

Each `.md` file inside a folder becomes an entry in that group. Its `weight`
sets the order within the group:

```markdown
---
title: "Front matter"
description: "The fields the theme reads."
weight: 20
---

## Your first heading

Write the page body here.
```

## How ordering works

Both the sidebar groups and the pages inside them sort by `weight`, ascending.
This is worth stating because tago's generic `sort` function orders a page list
the other way. The theme sorts with the `ByWeight` method instead, so a page
with `weight: 10` always appears above one with `weight: 20`. You only need to
set sensible weights; the theme does the rest.

## The home page

The `content/_index.md` at the root is the landing page. It does not appear in
the sidebar. Its hero text comes from parameters, and it shows a card for every
section marked `featured`. See [parameters](/customization/parameters/) for the
hero and feature settings.

## Nesting

Keep one level of folders for the cleanest sidebar: a section, then its pages.
That matches how the groups and the group titles are rendered.
