---
title: "Colors and styling"
description: "The CSS variables the theme exposes and how to override them."
weight: 20
---

The theme's colors and key sizes are CSS variables. You restyle the site by
overriding the variables, not by editing the theme stylesheet.

## How to override

Add your own stylesheet to the site's `static/` folder, loaded after the theme,
and set the variables you want to change. Because color variables are defined
per theme, set them under the matching `data-theme` selector:

```css
:root[data-theme="light"] {
  --accent: #c2410c;
}
:root[data-theme="dark"] {
  --accent: #fb923c;
}
```

Layout variables are not theme-specific and can be set on `:root`:

```css
:root {
  --doks-sidebar-w: 320px;
  --doks-max: 1600px;
}
```

## Color variables

These are defined for both `:root[data-theme="light"]` and
`:root[data-theme="dark"]`. The default accent is `#2f6df6` in light and
`#6f9bff` in dark.

| Variable | Role |
| --- | --- |
| `--bg`, `--bg-soft`, `--bg-mute` | Page and panel backgrounds, from base to muted. |
| `--surface` | Cards, the navbar, and raised panels. |
| `--fg`, `--fg-soft`, `--fg-mute` | Text, from primary to muted. |
| `--border`, `--border-strong` | Hairlines and stronger dividers. |
| `--accent`, `--accent-fg`, `--accent-soft` | The accent, its readable foreground, and its tinted background. |
| `--code-bg`, `--code-fg` | Code block background and text. |
| `--mark` | Highlighted text. |
| `--shadow` | Elevation shadow. |

## Layout variables

These set the frame. They live on `:root`.

| Variable | Default | Role |
| --- | --- | --- |
| `--doks-nav-h` | `60px` | Navbar height. |
| `--doks-sidebar-w` | `280px` | Left sidebar width. |
| `--doks-toc-w` | `240px` | Right table-of-contents width. |
| `--doks-max` | `1480px` | Maximum content width. |
| `--doks-radius` | `8px` | Corner radius. |
| `--doks-radius-lg` | `12px` | Larger corner radius for cards. |
| `--doks-font` | Inter stack | Body font. |
| `--doks-mono` | JetBrains Mono stack | Code font. |

## Going further

For anything beyond the variables, your override stylesheet can target the
theme's classes directly. The class names are stable and namespaced with a
`doks-` prefix, so you can style the navbar, sidebar, or pager without touching
the theme. Keep your overrides in your own file so theme updates stay clean.
