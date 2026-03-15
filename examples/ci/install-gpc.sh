#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-${GPC_VERSION:-v0.3.0}}"
INSTALL_DIR="${2:-${GPC_INSTALL_DIR:-$PWD/.bin}}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux|darwin)
    ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64)
    arch="amd64"
    ;;
  arm64|aarch64)
    arch="arm64"
    ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

archive="gpc_${VERSION}_${os}_${arch}.tar.gz"
base_url="https://github.com/leszko11/google-play-console-cli/releases/download/${VERSION}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

mkdir -p "$INSTALL_DIR"
curl -fsSL -o "$tmpdir/$archive" "$base_url/$archive"
tar -xzf "$tmpdir/$archive" -C "$tmpdir"
install -m 0755 "$tmpdir/gpc" "$INSTALL_DIR/gpc"

echo "installed gpc ${VERSION} to ${INSTALL_DIR}" >&2
