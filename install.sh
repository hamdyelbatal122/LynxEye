#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────
#  LynxEye — one-line installer
#  Usage: curl -sSL https://raw.githubusercontent.com/hamdyelbatal122/LynxEye/main/install.sh | bash
# ─────────────────────────────────────────────

REPO="hamdyelbatal122/LynxEye"
BINARY="lynxeye"
INSTALL_DIR="/usr/local/bin"

# ── helpers ──────────────────────────────────

print_info()  { printf "\033[1;34m[LynxEye]\033[0m %s\n" "$1"; }
print_ok()    { printf "\033[1;32m[LynxEye]\033[0m %s\n" "$1"; }
print_error() { printf "\033[1;31m[LynxEye]\033[0m %s\n" "$1" >&2; }

need() {
  if ! command -v "$1" &>/dev/null; then
    print_error "Required command not found: $1"
    exit 1
  fi
}

need curl
need tar

# ── detect OS ────────────────────────────────

OS="$(uname -s)"
case "$OS" in
  Linux)   OS="linux"   ;;
  Darwin)  OS="darwin"  ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *)
    print_error "Unsupported OS: $OS"
    exit 1
    ;;
esac

# ── detect arch ──────────────────────────────

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    print_error "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# ── fetch latest release tag ─────────────────

print_info "Fetching latest release…"

LATEST_TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

if [ -z "$LATEST_TAG" ]; then
  print_error "Could not determine latest release tag."
  exit 1
fi

print_info "Latest version: ${LATEST_TAG}"

# ── build download URL ───────────────────────

VERSION="${LATEST_TAG#v}"   # strip leading 'v' for filename

if [ "$OS" = "windows" ]; then
  ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.zip"
  need unzip
else
  ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/checksums.txt"

# ── download ─────────────────────────────────

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

print_info "Downloading ${ARCHIVE}…"
curl -fsSL --progress-bar "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE}"

# ── verify checksum ──────────────────────────

print_info "Verifying checksum…"
curl -fsSL "$CHECKSUM_URL" -o "${TMP_DIR}/checksums.txt"

(
  cd "$TMP_DIR"
  grep "$ARCHIVE" checksums.txt | sha256sum --check --status
)
print_ok "Checksum verified."

# ── extract ──────────────────────────────────

if [ "$OS" = "windows" ]; then
  unzip -q "${TMP_DIR}/${ARCHIVE}" -d "$TMP_DIR"
else
  tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
fi

# ── install ──────────────────────────────────

EXTRACTED_BINARY="${TMP_DIR}/${BINARY}"
[ "$OS" = "windows" ] && EXTRACTED_BINARY="${EXTRACTED_BINARY}.exe"

if [ ! -f "$EXTRACTED_BINARY" ]; then
  print_error "Binary not found after extraction: ${EXTRACTED_BINARY}"
  exit 1
fi

chmod +x "$EXTRACTED_BINARY"

# Use sudo only when needed
if [ -w "$INSTALL_DIR" ]; then
  mv "$EXTRACTED_BINARY" "${INSTALL_DIR}/${BINARY}"
else
  print_info "Installing to ${INSTALL_DIR} (sudo required)…"
  sudo mv "$EXTRACTED_BINARY" "${INSTALL_DIR}/${BINARY}"
fi

# ── done ─────────────────────────────────────

print_ok "LynxEye ${LATEST_TAG} installed → $(command -v ${BINARY})"
print_ok "Run: ${BINARY} --help"
