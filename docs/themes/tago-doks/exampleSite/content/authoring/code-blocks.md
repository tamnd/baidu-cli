---
title: "Code blocks"
description: "Copy buttons, syntax highlighting, and how to turn it on."
weight: 30
---

Code is a big part of documentation, so the theme gives it some care.

## Copy button

Every fenced code block gets a copy button that appears on hover. The script
wraps each `<pre>` and copies the block's text to the clipboard on click. You do
not need to do anything to enable it; write a normal fenced block:

````markdown
```bash
tago build
```
````

## Syntax highlighting

Highlighting is handled by tago, not the theme. Turn it on in `tago.toml`:

```toml
syntaxHighlight = "true"
```

With it on, tago colors fenced blocks by their language tag and writes a
`chroma.css` stylesheet that the theme loads. Tag your blocks with a language so
the colors are right:

````markdown
```go
func main() {
    fmt.Println("hello")
}
```
````

## Inline code

Inline spans with single backticks, like `tago serve`, get the same monospace
and subtle background as block code, in both light and dark themes.

## A note on long lines

Blocks scroll horizontally rather than wrapping, so a long command stays on one
line and stays copyable. Keep examples readable by breaking long shell commands
with a trailing backslash where it helps.
