#!/bin/sh
# This script installs FileAPI on Linux and macOS.
# It detects the OS/architecture and installs the matching release binary.

main() {
set -eu

red="$( (/usr/bin/tput bold || :; /usr/bin/tput setaf 1 || :) 2>&-)"
plain="$( (/usr/bin/tput sgr0 || :) 2>&-)"

status() { echo ">>> $*" >&2; }
error() { echo "${red}ERROR:${plain} $*"; exit 1; }

TEMP_DIR=$(mktemp -d)
cleanup() { rm -rf "$TEMP_DIR"; }
trap cleanup EXIT

available() { command -v "$1" >/dev/null 2>&1; }

GITHUB_REPO="${FILEAPI_GITHUB_REPO:-SmallAPIs/FileAPI}"
VERSION="${FILEAPI_VERSION:-}"
UNINSTALL="${FILEAPI_UNINSTALL:-0}"

OS="$(uname -s)"
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) error "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
  Linux) PLATFORM="linux" ;;
  Darwin) PLATFORM="darwin" ;;
  *) error "Unsupported operating system: $OS" ;;
esac

ARTIFACT="fileapi-${PLATFORM}-${ARCH}"
DOWNLOAD_BASE="https://github.com/${GITHUB_REPO}/releases"
if [ -n "$VERSION" ]; then
  DOWNLOAD_URL="${DOWNLOAD_BASE}/download/${VERSION}/${ARTIFACT}"
else
  DOWNLOAD_URL="${DOWNLOAD_BASE}/latest/download/${ARTIFACT}"
fi

if [ -n "${FILEAPI_INSTALL_DIR:-}" ]; then
  INSTALL_DIR="$FILEAPI_INSTALL_DIR"
  BINDIR="$INSTALL_DIR"
else
  case "$OS" in
    Darwin)
      INSTALL_DIR="/usr/local/bin"
      BINDIR="$INSTALL_DIR"
      ;;
    Linux)
      for candidate in /usr/local/bin /usr/bin /bin; do
        case ":$PATH:" in
          *:"$candidate":*) BINDIR="$candidate"; break ;;
        esac
      done
      BINDIR="${BINDIR:-/usr/local/bin}"
      INSTALL_DIR="$(dirname "$BINDIR")"
      ;;
  esac
fi

TARGET="${BINDIR}/fileapi"

uninstall() {
  status "Removing FileAPI from ${TARGET}..."
  if [ -f "$TARGET" ]; then
    if [ -w "$TARGET" ]; then
      rm -f "$TARGET"
    elif available sudo; then
      sudo rm -f "$TARGET"
    else
      error "Cannot remove ${TARGET}. Re-run with sudo or remove it manually."
    fi
    status "FileAPI has been uninstalled."
  else
    status "FileAPI is not installed at ${TARGET}."
  fi
}

if [ "$UNINSTALL" = "1" ]; then
  uninstall
  exit 0
fi

NEEDS=""
for tool in curl; do
  if ! available "$tool"; then
    NEEDS="$NEEDS $tool"
  fi
done
if [ -n "$NEEDS" ]; then
  error "Missing required tools:$NEEDS"
fi

SUDO=""
if [ ! -w "$BINDIR" ] 2>/dev/null; then
  if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
  elif available sudo; then
    SUDO="sudo"
  elif [ -z "${FILEAPI_INSTALL_DIR:-}" ] && [ "$OS" = "Linux" ]; then
    BINDIR="${HOME}/.local/bin"
    TARGET="${BINDIR}/fileapi"
    mkdir -p "$BINDIR"
    SUDO=""
  else
    error "Cannot write to ${BINDIR}. Set FILEAPI_INSTALL_DIR or re-run with sudo."
  fi
fi

status "Downloading FileAPI for ${PLATFORM}/${ARCH}..."
curl --fail --show-error --location --progress-bar \
  -o "$TEMP_DIR/fileapi" "$DOWNLOAD_URL"
chmod +x "$TEMP_DIR/fileapi"

status "Installing FileAPI to ${TARGET}..."
$SUDO mkdir -p "$BINDIR"
$SUDO install -m 755 "$TEMP_DIR/fileapi" "$TARGET"

status "Install complete. Run 'fileapi' from the command line."
if [ "$BINDIR" = "${HOME}/.local/bin" ]; then
  case ":$PATH:" in
    *:"$BINDIR":*) ;;
    *)
      status "Add ${BINDIR} to your PATH, for example:"
      status "  export PATH=\"${BINDIR}:\$PATH\""
      ;;
  esac
fi
status "Start the agent with: fileapi serve"
}

main
