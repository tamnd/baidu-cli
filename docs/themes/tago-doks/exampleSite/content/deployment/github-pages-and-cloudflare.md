---
title: "GitHub Pages and Cloudflare"
description: "A GitHub Actions workflow that publishes to both from one push."
weight: 20
---

This is a worked example: one workflow that builds the site twice and publishes
to GitHub Pages and to Cloudflare Pages with a custom domain. It is the setup
this site uses.

## The shape

- One `build` job builds the site twice, once per target base URL.
- One `deploy-pages` job publishes the sub-path build to GitHub Pages.
- One `deploy-cloudflare` job publishes the root build to Cloudflare Pages.

## The workflow

```yaml
name: Docs

on:
  push:
    branches: [main]
    paths: ["docs/**", ".github/workflows/docs.yml"]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write
  deployments: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6.0.2
        with:
          submodules: true
          fetch-depth: 0
      - name: Checkout tago
        uses: actions/checkout@v6.0.2
        with:
          repository: tamnd/tago
          path: .tago-src
      - uses: actions/setup-go@v6.4.0
        with:
          go-version-file: .tago-src/go.mod
      - run: cd .tago-src && go build -o /usr/local/bin/tago ./cmd/tago/
      - name: Build for GitHub Pages
        working-directory: docs
        run: tago build --base-url "https://username.github.io/project/" --output public-pages
      - name: Build for Cloudflare
        working-directory: docs
        run: tago build --base-url "https://docs.example.com/" --output public-cf
      - uses: actions/upload-pages-artifact@v5.0.0
        with:
          path: docs/public-pages
      - uses: actions/upload-artifact@v7.0.1
        with:
          name: public-cf
          path: docs/public-cf

  deploy-pages:
    runs-on: ubuntu-latest
    needs: build
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v5.0.0

  deploy-cloudflare:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/download-artifact@v8.0.1
        with:
          name: public-cf
          path: public-cf
      - env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
        run: |
          npx -y wrangler@4 pages deploy public-cf \
            --project-name=project --branch=main --commit-dirty=true
```

## One-time setup

- Enable GitHub Pages with the Actions build type. From the CLI:
  `gh api -X POST repos/OWNER/REPO/pages -f build_type=workflow`.
- Add the repository secrets `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`.
  GitHub Pages needs no secret.
- Create the Cloudflare Pages project and attach the custom domain. The first
  deploy of a new custom domain sits in a pending state while the certificate is
  issued, then becomes active on its own.

## The result

Every push to `main` that touches the site rebuilds it and republishes to both
URLs. The Pages build carries the project sub-path and the Cloudflare build
carries the domain root, so both are correct at once.
