#!/bin/bash
# Install P2P Download Client
# Usage: curl -sSL https://p2p.idist.dev/install.sh | bash

set -e

INSTALL_DIR="/usr/local/bin"
TRACKER_URL="https://p2p.idist.dev"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Determine binary name
case "$OS" in
    linux)
        BINARY_NAME="p2p-download-linux-amd64"
        ;;
    darwin)
        BINARY_NAME="p2p-download-darwin-arm64"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "=== P2P Download Client Installer ==="
echo "OS: $OS, Architecture: $ARCH"
echo ""

# Download URL (adjust based on your hosting)
DOWNLOAD_URL="${TRACKER_URL}/releases/${BINARY_NAME}"

# Check if we can download
echo "Downloading from: $DOWNLOAD_URL"

# Try to download
if command -v curl &> /dev/null; then
    curl -sSL -o /tmp/p2p-download "$DOWNLOAD_URL" || {
        echo "Download failed. Please download manually from the dashboard."
        exit 1
    }
elif command -v wget &> /dev/null; then
    wget -q -O /tmp/p2p-download "$DOWNLOAD_URL" || {
        echo "Download failed. Please download manually from the dashboard."
        exit 1
    }
else
    echo "curl or wget is required to download the client."
    exit 1
fi

# Make executable
chmod +x /tmp/p2p-download

# Install (may require sudo)
if [ -w "$INSTALL_DIR" ]; then
    mv /tmp/p2p-download "$INSTALL_DIR/p2p-download"
else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv /tmp/p2p-download "$INSTALL_DIR/p2p-download"
fi

echo ""
echo "✓ p2p-download installed successfully!"
echo ""
echo "Usage:"
echo "  p2p-download --list              # List available files"
echo "  p2p-download <hash>              # Download file by hash"
echo "  p2p-download --help              # Show help"
echo ""

