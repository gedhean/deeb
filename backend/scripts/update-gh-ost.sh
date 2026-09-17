#!/usr/bin/env bash
# Download the gh-ost Linux amd64 release binary into backend/ext-bin/gh-ost.
# Usage: ./scripts/update-gh-ost.sh [version]   (default: latest)
set -euo pipefail

BACKEND_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST_DIR="$BACKEND_ROOT/ext-bin"
DEST="$DEST_DIR/gh-ost"

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  VERSION=$(curl -sL https://api.github.com/repos/github/gh-ost/releases/latest | \
    grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
fi
if [[ -z "$VERSION" ]]; then
  echo "error: could not resolve gh-ost version" >&2
  exit 1
fi
echo "gh-ost version: $VERSION"

URL="https://github.com/github/gh-ost/releases/download/${VERSION}/gh-ost-linux-amd64.tar.gz"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "downloading $URL"
curl -fsSL "$URL" -o "$TMP/gh-ost.tar.gz"

# Verify against the release checksum
SUMS="$(curl -fsSL "https://github.com/github/gh-ost/releases/download/${VERSION}/SHA256SUMS")"
EXPECTED="$(echo "$SUMS" | grep 'gh-ost-linux-amd64.tar.gz' | awk '{print $1}')"
ACTUAL="$(shasum -a 256 "$TMP/gh-ost.tar.gz" | awk '{print $1}')"
if [[ -z "$EXPECTED" ]]; then
  echo "error: no checksum found for gh-ost-linux-amd64.tar.gz in SHA256SUMS" >&2
  exit 1
fi
if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "error: checksum mismatch (expected $EXPECTED, got $ACTUAL)" >&2
  exit 1
fi
echo "checksum ok: $ACTUAL"

tar -xzf "$TMP/gh-ost.tar.gz" -C "$TMP"
mkdir -p "$DEST_DIR"
mv "$TMP/gh-ost" "$DEST"
chmod +x "$DEST"

file "$DEST"
echo "updated: $DEST"
