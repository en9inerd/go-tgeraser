#!/bin/sh
set -eu

REPO="en9inerd/go-tgeraser"
BIN_NAME="tgeraser"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

err() { echo "error: $*" >&2; exit 1; }

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  *) err "unsupported OS: $OS" ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) err "unsupported arch: $(uname -m)" ;;
esac

ASSET="${BIN_NAME}-${OS}-${ARCH}"

if [ "$VERSION" = "latest" ]; then
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
  BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION#v}"
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

echo "downloading $ASSET from $BASE_URL" >&2
curl -fsSL -o "$ASSET" "$BASE_URL/$ASSET" || err "download failed"
curl -fsSL -o SHA256SUMS "$BASE_URL/SHA256SUMS" || err "SHA256SUMS download failed"

grep " ${ASSET}\$" SHA256SUMS > CHECKSUM || err "checksum not found for $ASSET"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum -c CHECKSUM >/dev/null || err "checksum mismatch"
else
  shasum -a 256 -c CHECKSUM >/dev/null || err "checksum mismatch"
fi

chmod +x "$ASSET"
SUDO=""
mkdir -p "$INSTALL_DIR" 2>/dev/null && [ -w "$INSTALL_DIR" ] || SUDO="sudo"
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO mv "$ASSET" "$INSTALL_DIR/$BIN_NAME"

echo "installed $INSTALL_DIR/$BIN_NAME" >&2
case ":$PATH:" in
  *":$INSTALL_DIR:"*) "$INSTALL_DIR/$BIN_NAME" --version ;;
  *) echo "warning: $INSTALL_DIR is not in PATH" >&2 ;;
esac
