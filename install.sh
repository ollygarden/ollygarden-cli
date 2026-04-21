#!/bin/sh
# Install the ollygarden CLI on macOS or Linux.
#
# Quick install:
#   curl -fsSL https://raw.githubusercontent.com/ollygarden/ollygarden-cli/main/install.sh | sh
#
# Customize:
#   OLLYGARDEN_VERSION=v0.1.0  curl ... | sh        # pin a version
#   OLLYGARDEN_INSTALL_DIR=/usr/local/bin curl ... | sh
#
# For full help:  curl -fsSL <url> -o install.sh && sh install.sh --help

set -eu

REPO="ollygarden/ollygarden-cli"
BIN_NAME="ollygarden"
INSTALL_DIR="${OLLYGARDEN_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
    cat <<EOF
Install the ollygarden CLI.

Usage:
  sh install.sh [--help]

Environment variables:
  OLLYGARDEN_VERSION       Pin to a specific release tag (e.g. v0.1.0).
                           Defaults to the latest release.
  OLLYGARDEN_INSTALL_DIR   Directory to install into.
                           Defaults to \$HOME/.local/bin.
  GITHUB_TOKEN             Optional GitHub token for resolving the latest
                           release. Only needed if you hit API rate limits.

Supports macOS and Linux on amd64 and arm64.
For Windows, download the zip from:
  https://github.com/$REPO/releases/latest
EOF
}

case "${1:-}" in
    -h|--help) usage; exit 0 ;;
    "") ;;
    *) printf 'Error: unknown argument: %s\n\n' "$1" >&2; usage >&2; exit 2 ;;
esac

log()  { printf '%s\n' "$*" >&2; }
fail() { log "Error: $*"; exit 1; }

require() {
    command -v "$1" >/dev/null 2>&1 || fail "missing required tool: $1"
}

require curl
require tar
require uname
require mkdir
require mv
require rm

# Detect OS.
os_raw="$(uname -s)"
case "$os_raw" in
    Linux)   os="linux" ;;
    Darwin)  os="darwin" ;;
    *)       fail "unsupported OS: $os_raw (use the GitHub Release zip for Windows)" ;;
esac

# Detect arch.
arch_raw="$(uname -m)"
case "$arch_raw" in
    x86_64|amd64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)             fail "unsupported architecture: $arch_raw" ;;
esac

# Resolve version. GITHUB_TOKEN is used if set (raises API rate limits from
# 60/hr to 5000/hr — useful on shared IPs, corporate networks, and CI).
if [ -n "${OLLYGARDEN_VERSION:-}" ]; then
    version="$OLLYGARDEN_VERSION"
else
    log "Resolving latest release..."
    auth_header=""
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        auth_header="Authorization: Bearer $GITHUB_TOKEN"
    fi
    version="$(curl -fsSL ${auth_header:+-H "$auth_header"} \
        "https://api.github.com/repos/$REPO/releases/latest" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n 1)"
    [ -n "$version" ] || fail "could not resolve latest release tag"
fi

version_num="${version#v}"
archive="${BIN_NAME}_${version_num}_${os}_${arch}.tar.gz"
base_url="https://github.com/$REPO/releases/download/$version"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT HUP TERM

log "Downloading $archive..."
curl -fsSL -o "$tmp/$archive"        "$base_url/$archive"
curl -fsSL -o "$tmp/checksums.txt"   "$base_url/checksums.txt"

# Verify checksum.
log "Verifying checksum..."
expected="$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || fail "no checksum entry for $archive"

if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
else
    fail "missing required tool: sha256sum or shasum"
fi

[ "$expected" = "$actual" ] || fail "checksum mismatch: expected $expected, got $actual"

# Extract and install.
log "Extracting..."
tar -xzf "$tmp/$archive" -C "$tmp"

mkdir -p "$INSTALL_DIR"
mv "$tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

log "Installed $BIN_NAME $version to $INSTALL_DIR/$BIN_NAME"

# Strip macOS quarantine so Gatekeeper doesn't block the unsigned binary.
if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$INSTALL_DIR/$BIN_NAME" 2>/dev/null || true
fi

case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) log ""; log "Note: $INSTALL_DIR is not on your PATH. Add it with:"
       log "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
