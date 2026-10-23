#!/usr/bin/env bash
# OpsVault One-Click Installation Script
# Supports: Linux (amd64 / arm64) & macOS (amd64 / arm64)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/chenxi2015/opsvault/main/scripts/install.sh | bash
#   curl -fsSL https://ghfast.top/https://raw.githubusercontent.com/chenxi2015/opsvault/main/scripts/install.sh | bash
#
# Environment Variables / Options:
#   OPSVAULT_VERSION      Target release tag (e.g. v0.1.0, default: latest)
#   OPSVAULT_INSTALL_DIR  Root directory for OpsVault (default: /data/opsvault on Linux root, ~/.opsvault otherwise)
#   OPSVAULT_BIN_DIR      Directory for global binary symlink (default: /usr/local/bin)

set -e

REPO="chenxi2015/opsvault"
VERSION="${OPSVAULT_VERSION:-latest}"
INSTALL_DIR="${OPSVAULT_INSTALL_DIR:-}"
BIN_DIR="${OPSVAULT_BIN_DIR:-/usr/local/bin}"

# Built-in GitHub mirrors for mainland China acceleration (ordered by priority, empty string means direct GitHub)
PROXIES=(
  "https://ghfast.top/"
  "https://mirror.ghproxy.com/"
  "https://ghproxy.net/"
  "https://hub.gitmirror.com/"
  ""
)

info()  { printf "\033[1;34m==>\033[0m %s\n" "$*"; }
ok()    { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn()  { printf "\033[1;33m!\033[0m %s\n" "$*"; }
error() { printf "\033[1;31m✗ Error:\033[0m %s\n" "$*" >&2; exit 1; }

# Parse CLI flags when invoked with arguments (e.g. bash -s -- --version v0.1.0)
while [ $# -gt 0 ]; do
  case "$1" in
    -v|--version)
      VERSION="$2"
      shift 2
      ;;
    -d|--install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    -b|--bin-dir)
      BIN_DIR="$2"
      shift 2
      ;;
    -h|--help)
      echo "Usage: install.sh [--version vX.Y.Z] [--install-dir /path] [--bin-dir /usr/local/bin]"
      exit 0
      ;;
    *)
      shift
      ;;
  esac
done

detect_platform() {
  local uname_s uname_m
  uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
  uname_m="$(uname -m)"

  case "$uname_s" in
    linux*)  OS="linux" ;;
    darwin*) OS="darwin" ;;
    *)       error "Unsupported operating system: ${uname_s} (only Linux and macOS are supported)" ;;
  esac

  case "$uname_m" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *)              error "Unsupported architecture: ${uname_m} (only amd64 and arm64 are supported)" ;;
  esac
}

resolve_install_dir() {
  if [ -n "$INSTALL_DIR" ]; then
    return
  fi

  if [ "$OS" = "linux" ] && [ "${EUID:-$(id -u)}" -eq 0 ]; then
    INSTALL_DIR="/data/opsvault"
  else
    INSTALL_DIR="${HOME}/.opsvault"
  fi
}

download_with_fallback() {
  local raw_url="$1"
  local dest="$2"
  local proxy target_url

  for proxy in "${PROXIES[@]}"; do
    target_url="${proxy}${raw_url}"
    if curl -fsSL --connect-timeout 8 --max-time 300 -o "$dest" "$target_url" 2>/dev/null; then
      if [ -s "$dest" ]; then
        return 0
      fi
    fi
  done
  return 1
}

resolve_latest_version() {
  if [ "$VERSION" != "latest" ] && [ -n "$VERSION" ]; then
    case "$VERSION" in
      v*) ;;
      *)  VERSION="v${VERSION}" ;;
    esac
    return
  fi

  info "Resolving latest OpsVault release version..."
  local api_url="https://api.github.com/repos/${REPO}/releases/latest"
  local tag=""

  # Strategy 1: Query GitHub Release API directly
  tag="$(curl -fsSL --connect-timeout 5 "$api_url" 2>/dev/null | grep '"tag_name"' | head -n 1 | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/' || true)"

  # Strategy 2: Follow GitHub releases/latest HTTP redirect header via proxies or direct
  if [ -z "$tag" ]; then
    local proxy redirect_url
    for proxy in "" "${PROXIES[@]}"; do
      redirect_url="$(curl -fsSLI --connect-timeout 6 -o /dev/null -w '%{url_effective}' "${proxy}https://github.com/${REPO}/releases/latest" 2>/dev/null || true)"
      case "$redirect_url" in
        */releases/tag/*)
          tag="${redirect_url##*/releases/tag/}"
          tag="${tag%%[?#]*}"
          break
          ;;
      esac
    done
  fi

  if [ -z "$tag" ]; then
    error "Failed to resolve the latest version tag from GitHub. You can specify a version explicitly via: OPSVAULT_VERSION=v0.1.0"
  fi

  VERSION="$tag"
}

verify_checksum() {
  local file_path="$1"
  local file_name="$2"
  local checksums_file="$3"

  if [ ! -s "$checksums_file" ]; then
    return 0
  fi

  local expected_hash
  expected_hash="$(grep -E "[[:space:]]${file_name}\$" "$checksums_file" | awk '{print $1}' | head -n 1 || true)"
  if [ -z "$expected_hash" ]; then
    return 0
  fi

  local actual_hash=""
  if command -v sha256sum >/dev/null 2>&1; then
    actual_hash="$(sha256sum "$file_path" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual_hash="$(shasum -a 256 "$file_path" | awk '{print $1}')"
  else
    warn "Neither sha256sum nor shasum found; skipping SHA256 verification."
    return 0
  fi

  if [ "$actual_hash" != "$expected_hash" ]; then
    error "SHA256 checksum mismatch for ${file_name} (expected ${expected_hash}, got ${actual_hash})"
  fi
  ok "SHA256 checksum verified (${file_name})"
}

main() {
  command -v curl >/dev/null 2>&1 || error "'curl' is required but not installed."

  detect_platform
  resolve_install_dir
  resolve_latest_version

  local target_name="opsvault-${OS}-${ARCH}"
  local archive_name="${target_name}-${VERSION}.tar.gz"
  local base_download_url="https://github.com/${REPO}/releases/download/${VERSION}"

  info "Detected platform: ${OS}/${ARCH}"
  info "Target version:    ${VERSION}"
  info "Install directory: ${INSTALL_DIR}"

  local tmp_dir
  tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t opsvault_install)"
  trap 'rm -rf "$tmp_dir"' EXIT

  local checksums_path="${tmp_dir}/checksums.txt"
  download_with_fallback "${base_download_url}/checksums.txt" "$checksums_path" || true

  mkdir -p "${INSTALL_DIR}/bin" "${INSTALL_DIR}/configs"
  local bin_dest="${INSTALL_DIR}/bin/opsvault"

  info "Downloading ${archive_name}..."
  if download_with_fallback "${base_download_url}/${archive_name}" "${tmp_dir}/${archive_name}"; then
    verify_checksum "${tmp_dir}/${archive_name}" "$archive_name" "$checksums_path"
    tar -xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"
    cp "${tmp_dir}/${target_name}/opsvault" "$bin_dest"
  else
    info "Archive not found, falling back to standalone binary ${target_name}..."
    download_with_fallback "${base_download_url}/${target_name}" "${tmp_dir}/${target_name}" \
      || error "Failed to download OpsVault (${VERSION}) for ${OS}/${ARCH}"
    verify_checksum "${tmp_dir}/${target_name}" "$target_name" "$checksums_path"
    cp "${tmp_dir}/${target_name}" "$bin_dest"
  fi

  chmod +x "$bin_dest"
  ok "Installed binary to ${bin_dest}"

  # Create global symlink in BIN_DIR
  local symlink_path="${BIN_DIR}/opsvault"
  mkdir -p "$BIN_DIR" 2>/dev/null || true
  if ln -sf "$bin_dest" "$symlink_path" 2>/dev/null; then
    ok "Linked global command: ${symlink_path} -> ${bin_dest}"
  elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null && sudo mkdir -p "$BIN_DIR" && sudo ln -sf "$bin_dest" "$symlink_path"; then
    ok "Linked global command (via sudo): ${symlink_path} -> ${bin_dest}"
  else
    warn "No write permission to ${BIN_DIR}."
    warn "Please run: sudo ln -sf ${bin_dest} ${symlink_path}"
    warn "Or add ${INSTALL_DIR}/bin to your PATH: export PATH=\"${INSTALL_DIR}/bin:\$PATH\""
  fi

  # Initialize default configuration files (default.yaml & ansible.yaml) without overwriting existing configs
  info "Initializing configuration files in ${INSTALL_DIR}/configs..."
  "$bin_dest" init --config-only --config-dir "${INSTALL_DIR}/configs"

  printf "\n\033[1;32m============================================================\033[0m\n"
  printf "\033[1;32m  OpsVault %s installed successfully!\033[0m\n" "$VERSION"
  printf "\033[1;32m============================================================\033[0m\n"
  printf "  Binary Path:   %s\n" "$bin_dest"
  printf "  Config File:   %s/configs/default.yaml\n" "$INSTALL_DIR"
  printf "  Ansible File:  %s/configs/ansible.yaml\n" "$INSTALL_DIR"
  printf "\n  \033[1;36mQuick Start:\033[0m\n"
  printf "    opsvault version                # Verify version & platform\n"
  printf "    opsvault init --config-only     # Export default YAML configs anytime\n"
  printf "    opsvault init                   # Interactive middleware installation\n"
  printf "    opsvault tui                    # Launch interactive TUI console\n"
  printf "    opsvault upgrade                # Self-upgrade to latest release\n\n"
}

main "$@"

