---
title: "Dark mode"
description: "How the light and dark themes are chosen and stored."
weight: 30
---

## How it works

The theme defines its colors as CSS variables on two selectors, `:root[data-theme="light"]`
and `:root[data-theme="dark"]`. A small script in the page head sets the
`data-theme` attribute before the first paint, so there is no flash of the wrong
colors.

## The order of preference

1. If the reader has toggled the theme before, the stored choice wins. It is
   kept in `localStorage` under the key `doks-theme`.
2. Otherwise the theme follows the operating system through the
   `prefers-color-scheme` media query.

The toggle button in the navbar flips between the two and saves the result.

## Changing the palette

The colors for each theme are CSS variables, so you change them by overriding
the variables in your own stylesheet. See
[colors and styling](/customization/colors-and-styling/) for the full list and
how to load an override.
