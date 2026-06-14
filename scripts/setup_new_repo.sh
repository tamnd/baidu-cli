#!/bin/bash
# setup_new_repo.sh OWNER/REPO
# Enables GitHub Pages (workflow mode) and sets Cloudflare secrets.
set -euo pipefail

REPO="${1:?usage: setup_new_repo.sh OWNER/REPO}"

echo "enabling GitHub Pages for $REPO..."
gh api "repos/$REPO/pages" --method POST -f build_type=workflow 2>/dev/null || true

echo "setting Cloudflare secrets on $REPO..."
gh secret set CLOUDFLARE_ACCOUNT_ID --repo "$REPO" --body "${CLOUDFLARE_ACCOUNT_ID:?CLOUDFLARE_ACCOUNT_ID not set}"
gh secret set CLOUDFLARE_API_TOKEN  --repo "$REPO" --body "${CLOUDFLARE_API_TOKEN:?CLOUDFLARE_API_TOKEN not set}"

echo "done"
