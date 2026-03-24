#!/bin/sh
set -e

# Codong installer
# Usage: curl -fsSL https://codong.org/install.sh | sh

REPO="brettinhere/Codong"
BINARY="codong"

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             echo "unsupported" ;;
    esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ "$OS" = "unsupported" ]; then
    echo "Error: unsupported operating system $(uname -s)"
    exit 1
fi

if [ "$ARCH" = "unsupported" ]; then
    echo "Error: unsupported architecture $(uname -m)"
    exit 1
fi

# ---------------------------------------------------------------------------
# Resolve latest version
# ---------------------------------------------------------------------------

if [ -z "$VERSION" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
fi

if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest release version"
    exit 1
fi

ASSET="${BINARY}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

echo "Installing Codong ${VERSION} (${OS}/${ARCH})..."

# ---------------------------------------------------------------------------
# Download
# ---------------------------------------------------------------------------

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${URL}..."
curl -fsSL -o "${TMP}/${BINARY}" "$URL"
chmod +x "${TMP}/${BINARY}"

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------

INSTALL_DIR="/usr/local/bin"

if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
    echo "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    INSTALL_DIR="${HOME}/bin"
    mkdir -p "$INSTALL_DIR"
    mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    echo ""
    echo "Installed to ${INSTALL_DIR}/${BINARY}"
    echo "Make sure ${INSTALL_DIR} is in your PATH:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

# ---------------------------------------------------------------------------
# Verify
# ---------------------------------------------------------------------------

if command -v codong >/dev/null 2>&1; then
    echo ""
    echo "Codong installed successfully!"
    codong version
else
    echo ""
    echo "Codong installed to ${INSTALL_DIR}/${BINARY}"
    echo "Run 'codong version' to verify."
fi
