#!/bin/bash

set -e

# Gorphanage installer script
# Usage: curl -sSL https://raw.githubusercontent.com/mirrir0/gorphanage/main/install.sh | bash

REPO="mirrir0/gorphanage"
BINARY="gorphanage"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
x86_64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

case $OS in
linux | darwin) ;;
*) echo "Unsupported OS: $OS" && exit 1 ;;
esac

# Get latest release version
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
  echo "Failed to get latest version"
  exit 1
fi

echo "Installing $BINARY $LATEST_VERSION..."

# Download binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY}_${OS^}_${ARCH}.tar.gz"
if [ "$OS" = "windows" ]; then
  DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY}_${OS^}_${ARCH}.zip"
fi

TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

echo "Downloading from $DOWNLOAD_URL..."
curl -sL "$DOWNLOAD_URL" -o "archive"

# Extract
if [ "$OS" = "windows" ]; then
  unzip -q archive
else
  tar -xzf archive
fi

# Install
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

mv "$BINARY" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BINARY"

# Cleanup
cd -
rm -rf "$TEMP_DIR"

echo "✅ $BINARY installed to $INSTALL_DIR/$BINARY"
echo ""
echo "Make sure $INSTALL_DIR is in your PATH:"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
echo ""
echo "Usage:"
echo "  $BINARY /path/to/your/go/project"
echo "  $BINARY --json /path/to/your/go/project"
