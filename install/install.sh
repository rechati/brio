#!/bin/bash

set -e

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

REPO="rechati/brio"
VERSION="$(basename "$(curl -Ls -o /dev/null -w "%{url_effective}" "https://github.com/rechati/brio/releases/latest")")"
TARBALL="brio-$VERSION-$OS-$ARCH.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$TARBALL"

INSTALL_DIR="/usr/local/bin"

echo "$DOWNLOAD_URL"
echo "Downloading brio..."
curl -L "$DOWNLOAD_URL" -o "/tmp/$TARBALL"

echo "Unpacking..."
tar -xzf "/tmp/$TARBALL" -C /tmp/

echo "Moving the binary to $INSTALL_DIR..."
sudo mv /tmp/brio "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/brio"

rm "/tmp/$TARBALL"

echo "brio has been installed to $INSTALL_DIR/brio"
