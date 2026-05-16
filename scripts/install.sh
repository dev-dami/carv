#!/usr/bin/env bash
#
# Carv installer — downloads the latest release binary from GitHub and installs it.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dev-dami/carv/main/scripts/install.sh | bash
#   # or with a specific version:
#   curl -fsSL https://raw.githubusercontent.com/dev-dami/carv/main/scripts/install.sh | bash -s -- v0.5.1-beta
#
set -euo pipefail

REPO="dev-dami/carv"
BIN_NAME="carv"

# --- colour helpers (disabled when stdout is not a tty) ---
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    RED='\033[0;31m'
    NC='\033[0m'
else
    GREEN='' YELLOW='' RED='' NC=''
fi

info()  { echo -e "${GREEN}[info]${NC} $*"; }
warn()  { echo -e "${YELLOW}[warn]${NC} $*"; }
error() { echo -e "${RED}[error]${NC} $*" >&2; }

# --- determine version ---
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    info "No version specified — fetching latest release..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases?per_page=1" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
    if [ -z "$VERSION" ]; then
        error "Could not determine latest release version."
        exit 1
    fi
fi
info "Installing Carv ${VERSION}"

# --- detect OS / arch ---
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
    linux)  GOOS="linux" ;;
    darwin) GOOS="darwin" ;;
    msys*|mingw*|cygwin*) GOOS="windows" ;;
    *) error "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64|amd64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH"; exit 1 ;;
esac

info "Platform: ${GOOS}/${GOARCH}"

# --- build download URL ---
EXT=""
[ "$GOOS" = "windows" ] && EXT=".exe"
ASSET="${BIN_NAME}-${VERSION}-${GOOS}-${GOARCH}${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

# --- download ---
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${ASSET}..."
if ! curl -fsSL -o "${TMPDIR}/${BIN_NAME}${EXT}" "$URL"; then
    error "Download failed. Check that ${VERSION} exists at https://github.com/${REPO}/releases"
    exit 1
fi

# --- install ---
if [ "$GOOS" = "windows" ]; then
    # Windows: try common locations
    INSTALL_DIR="${CARV_INSTALL_DIR:-$HOME/.local/bin}"
    mkdir -p "$INSTALL_DIR"
    mv "${TMPDIR}/${BIN_NAME}${EXT}" "${INSTALL_DIR}/${BIN_NAME}${EXT}"
    info "Installed to ${INSTALL_DIR}\\${BIN_NAME}${EXT}"
    info "Ensure ${INSTALL_DIR} is in your PATH."
else
    # Unix: try /usr/local/bin first, fall back to ~/.local/bin
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${CARV_INSTALL_DIR:-$HOME/.local/bin}"
        mkdir -p "$INSTALL_DIR"
    fi

    mv "${TMPDIR}/${BIN_NAME}${EXT}" "${INSTALL_DIR}/${BIN_NAME}"
    chmod +x "${INSTALL_DIR}/${BIN_NAME}"
    info "Installed to ${INSTALL_DIR}/${BIN_NAME}"

    if ! echo "$PATH" | tr ':' '\n' | grep -qxF "$INSTALL_DIR"; then
        warn "${INSTALL_DIR} is not in your PATH. Add it with:"
        warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi
fi

# --- verify ---
if command -v "$BIN_NAME" >/dev/null 2>&1; then
    echo ""
    info "Installation successful!"
    "$BIN_NAME" version
else
    echo ""
    warn "Binary installed but not found in PATH. Restart your shell or run:"
    warn "  export PATH=\"\$PATH:${INSTALL_DIR}\""
fi
