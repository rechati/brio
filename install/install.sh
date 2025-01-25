#!/bin/bash

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

REPO="rechati/brio"
VERSION="latest"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/brio-$OS-$ARCH"

INSTALL_DIR="/usr/local/bin"

echo "Downloading brio..."
sudo curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/brio"
sudo chmod +x "$INSTALL_DIR/brio"

echo "brio has been installed to $INSTALL_DIR/brio"
