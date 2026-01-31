#!/bin/bash
set -e

# jiractl installer
# Usage: curl -fsSL https://raw.githubusercontent.com/e/jiractl/main/install.sh | bash

REPO="e/jiractl"
BINARY="jiractl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *)
    echo "Error: Unsupported operating system: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected: ${OS}/${ARCH}"

# Get latest version
echo "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Error: Could not determine latest version"
  exit 1
fi

echo "Latest version: ${LATEST}"

# Download URL
VERSION="${LATEST#v}"
FILENAME="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

# Download and extract
echo "Downloading ${URL}..."
curl -fsSL "${URL}" -o "${TMP_DIR}/${FILENAME}"

echo "Extracting..."
tar -xzf "${TMP_DIR}/${FILENAME}" -C "${TMP_DIR}"

# Install
echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "Successfully installed ${BINARY} ${LATEST} to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  ${BINARY} configure    # Set up credentials"
echo "  ${BINARY} --help       # Show help"
