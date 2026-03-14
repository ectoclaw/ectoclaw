#!/usr/bin/env sh
set -e

REPO="ectoclaw/ectoclaw"
BIN="ectoclaw"
INSTALL_DIR="/usr/local/bin"

# ── Detect OS ────────────────────────────────────────────────────────────────
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# ── Detect arch ──────────────────────────────────────────────────────────────
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="x86_64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# ── Resolve latest version ───────────────────────────────────────────────────
LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
if command -v curl >/dev/null 2>&1; then
  VERSION="$(curl -fsSL "$LATEST_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
elif command -v wget >/dev/null 2>&1; then
  VERSION="$(wget -qO- "$LATEST_URL" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
else
  echo "curl or wget is required" >&2
  exit 1
fi

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version" >&2
  exit 1
fi

# ── Download and install ─────────────────────────────────────────────────────
TARBALL="${BIN}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

echo "Installing ${BIN} ${VERSION} (${OS}/${ARCH})..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" | tar xz -C "$TMP"
else
  wget -qO- "$DOWNLOAD_URL" | tar xz -C "$TMP"
fi

# ── Place binary ─────────────────────────────────────────────────────────────
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BIN" "$INSTALL_DIR/$BIN"
  chmod +x "$INSTALL_DIR/$BIN"
else
  echo "Installing to $INSTALL_DIR requires sudo..."
  sudo mv "$TMP/$BIN" "$INSTALL_DIR/$BIN"
  sudo chmod +x "$INSTALL_DIR/$BIN"
fi

echo ""
echo "${BIN} ${VERSION} installed to ${INSTALL_DIR}/${BIN}"
echo ""
echo "Next steps:"
echo "  ${BIN} onboard          # initialise config & workspace"
echo "  sudo ${BIN} service install   # install as a system service (Linux)"
echo "  sudo ${BIN} service start"
