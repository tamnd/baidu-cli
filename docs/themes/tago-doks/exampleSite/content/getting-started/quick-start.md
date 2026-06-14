---
title: "Quick start"
description: "Write your first page and see it in the sidebar."
weight: 20
---

## Create a section

Each top-level folder under `content/` becomes a sidebar group. Give it an
`_index.md` with a title and a weight to control its order:

```markdown
---
title: "Guides"
weight: 20
---
```

## Add a page

Pages inside the folder become entries in that group. Their `weight` sets the
order within the group:

```markdown
---
title: "Reading data"
description: "Query rows over HTTP."
weight: 10
---

## Your first request

Write your content here. Headings become the table of contents on the right.
```

## What you get

- The page appears in the left sidebar under its section.
- Every `h2` and `h3` is collected into the table of contents on the right.
- The text is added to the search index automatically.
- Code blocks get a copy button on hover.
