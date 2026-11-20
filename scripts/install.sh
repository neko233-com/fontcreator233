#!/usr/bin/env bash
# fontcreator233 one-click installer for macOS / Linux
# Usage:
#   curl -fsSL https://.../install.sh | bash
#   bash install.sh v1.0.0 /usr/local/bin
set -euo pipefail

VERSION="${1:-}"
INSTALL_DIR="${2:-$HOME/.local/bin}"
REPO="neko233-com/fontcreator233"
MIRRORS=(
  "https://github.com"
  "https://ghproxy.net/https://github.com"
  "https://mirror.ghproxy.com/https://github.com"
  "https://ghfast.top/https://github.com"
)

echo "fontcreator233 installer"

if [ -z "$VERSION" ]; then
  echo "  resolving latest tag ..."
  TAG=""
  for m in "${MIRRORS[@]}"; do
    if TAG=$(curl -fsSL "$m/api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep -o '"tag_name":"v[^"]*"' | head -1 | cut -d'"' -f4); then
      echo "    latest: $TAG (via $m)"; break
    fi
  done
  if [ -z "$TAG" ]; then echo "cannot resolve latest; check network or pass version arg" >&2; exit 1; fi
  VERSION="$TAG"
fi
echo "  version:   $VERSION"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *) echo "unsupported arch: $ARCH" >&2; exit 1 ;;
esac
PLATFORM="$OS-$GOARCH"
ARCHIVE="fontcreator233-$VERSION-$PLATFORM.tar.gz"

echo "  platform:  $PLATFORM"
echo "  probe mirrors ..."
BASE=""
for m in "${MIRRORS[@]}"; do
  if curl -fsSI "$m/$REPO/releases/download/$VERSION/$ARCHIVE" >/dev/null 2>&1; then
    BASE="$m"; break
  fi
done
if [ -z "$BASE" ]; then echo "no reachable mirror for $ARCHIVE" >&2; exit 1; fi
echo "    using $BASE"

TMP="$(mktemp -d)"
trap "rm -rf $TMP" EXIT
URL="$BASE/$REPO/releases/download/$VERSION/$ARCHIVE"
echo "  downloading $ARCHIVE ..."
curl -fsSL "$URL" -o "$TMP/$ARCHIVE"

# verify sha256 if available
SHA_URL="$URL.sha256"
if curl -fsSL "$SHA_URL" -o "$TMP/${ARCHIVE}.sha256" 2>/dev/null; then
  (cd "$TMP" && sha256sum -c "${ARCHIVE}.sha256") || echo "  sha256 mismatch — continuing anyway" >&2
  echo "  sha256 OK"
fi

mkdir -p "$INSTALL_DIR"
echo "  extracting to $INSTALL_DIR ..."
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
# find binary in archive
BIN_PATH="$(find "$TMP" -name fontcreator -type f | head -1)"
if [ -z "$BIN_PATH" ]; then echo "binary not found in archive" >&2; exit 1; fi
install -m 0755 "$BIN_PATH" "$INSTALL_DIR/fontcreator"

# add to PATH if needed
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    PROFILE=""
    [ -f "$HOME/.zshrc" ] && PROFILE="$HOME/.zshrc"
    [ -f "$HOME/.bashrc" ] && PROFILE="$HOME/.bashrc"
    [ -f "$HOME/.profile" ] && PROFILE="$HOME/.profile"
    if [ -n "$PROFILE" ]; then
      echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE"
      echo "  added $INSTALL_DIR to PATH in $PROFILE"
    else
      echo "  install dir $INSTALL_DIR is not on PATH — add it manually" >&2
    fi
    ;;
esac

echo "  done. Run: fontcreator version"
