---
title: "Navigation"
description: "The sidebar, the table of contents, the pager, and small-screen behavior."
weight: 20
---

The theme keeps the reader oriented with four pieces of navigation, all built
from your content.

## The sidebar

The left sidebar lists every section as a group and every page as an entry,
ordered by `weight`. The current page is highlighted. You write content folders;
the sidebar follows. See [sections and pages](/authoring/sections-and-pages/).

## The table of contents

The right-hand column is built from the headings on the page, the `h2` and `h3`
levels. As the reader scrolls, the heading currently in view is highlighted,
using an IntersectionObserver, so they always know where they are. Each heading
also gets an anchor link on hover, for sharing a deep link into the page.

Because the table of contents uses two heading levels, structure a page with
`##` for its main sections and `###` for sub-sections. Deeper headings still
render; they just do not appear in the contents.

## The pager

At the foot of each page, a previous and next link move through the section in
weight order, so a reader can read a section straight through without returning
to the sidebar.

## Small screens

The layout is responsive. At a medium width the table of contents is hidden to
give the content room. At a small width the sidebar collapses behind a button in
the navbar and slides in over a backdrop when opened. No configuration is
needed; the breakpoints are built in.

## The navbar

The top bar carries the project name on the left, and on the right the search
box, a link to your repository when `github` is set, and the light and dark
toggle. See [parameters](/customization/parameters/) for the navbar settings.
