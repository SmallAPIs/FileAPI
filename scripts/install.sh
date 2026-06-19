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

GITHUB_TOKEN="${FILEAPI_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"
CURL_AUTH=()
if [ -n "$GITHUB_TOKEN" ]; then
  CURL_AUTH=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

download_release_asset() {
  dest="$1"
  if curl --fail --show-error --location --progress-bar \
    "${CURL_AUTH[@]}" -o "$dest" "$DOWNLOAD_URL" 2>/dev/null; then
    return 0
  fi

  if [ -z "$GITHUB_TOKEN" ]; then
    error "Download failed. ${GITHUB_REPO} appears to be private.

Use one of these options:
  1. Make the repository public in GitHub Settings
  2. Install with GitHub CLI:
       gh api repos/${GITHUB_REPO}/contents/scripts/install.sh?ref=main -H \"Accept: application/vnd.github.raw\" | sh
  3. Export a token with repo read access:
       export FILEAPI_GITHUB_TOKEN=ghp_...
       curl -fsSL .../install.sh | sh"
  fi

  if [ -n "$VERSION" ]; then
    release_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${VERSION}"
  else
    release_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
  fi

  release_json=$(curl -fsSL "${CURL_AUTH[@]}" \
    -H "Accept: application/vnd.github+json" \
    "$release_url")

  if available python3; then
    asset_id=$(printf '%s' "$release_json" | python3 -c "import json,sys; data=json.load(sys.stdin); print(next(a['id'] for a in data['assets'] if a['name']=='${ARTIFACT}'))")
  elif available jq; then
    asset_id=$(printf '%s' "$release_json" | jq -r --arg name "$ARTIFACT" '.assets[] | select(.name == $name) | .id')
  else
    error "Private repo install requires python3 or jq to locate the release asset."
  fi

  if [ -z "$asset_id" ] || [ "$asset_id" = "null" ]; then
    error "Could not find release asset ${ARTIFACT} for ${GITHUB_REPO}."
  fi

  curl --fail --show-error --location --progress-bar \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/octet-stream" \
    -o "$dest" "https://api.github.com/repos/${GITHUB_REPO}/releases/assets/${asset_id}"
}

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
download_release_asset "$TEMP_DIR/fileapi"
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
