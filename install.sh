#!/usr/bin/env bash
# install.sh — one-liner installer for github-authorized-keys
#
# One-shot usage (fully configured, service starts immediately):
#   curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash -s -- --username YOUR_GITHUB_USER
#
# Or with an environment variable:
#   curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | GAK_GITHUB_USERNAME=YOUR_GITHUB_USER bash
#
# Interactive usage (prompts for username if stdin is a terminal):
#   curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash
#
# Flags:
#   --username, -u  GitHub username (or comma-separated list)
#   --interval      Sync interval as Go duration (default: 1h)
#   --keys-path     Override authorized_keys path
#   --help, -h      Show this help

set -euo pipefail

REPO="stevemcquaid/github-authorized-keys"
BINARY="github-authorized-keys"
INSTALL_DIR="${HOME}/.local/bin"
SYSTEMD_DIR="${HOME}/.config/systemd/user"
CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/github-authorized-keys"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
SERVICE_FILE="github-authorized-keys.service"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { echo "[INFO]  $*"; }
warn()  { echo "[WARN]  $*" >&2; }
error() { echo "[ERROR] $*" >&2; exit 1; }

usage() {
  echo "Usage: install.sh [--username GITHUB_USER] [--interval 1h] [--keys-path PATH]"
  echo ""
  echo "  --username, -u   GitHub username(s) to sync keys from (comma-separated)"
  echo "  --interval       Sync interval as Go duration, e.g. 30m, 1h, 6h (default: 1h)"
  echo "  --keys-path      Override path to authorized_keys file"
  echo "  --help, -h       Show this help"
  echo ""
  echo "Environment variable alternative: GAK_GITHUB_USERNAME=user1,user2"
}

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux";;
    Darwin*) echo "darwin";;
    *)       error "Unsupported OS: $(uname -s)";;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64";;
    aarch64|arm64) echo "arm64";;
    *)             error "Unsupported architecture: $(uname -m)";;
  esac
}

check_dependency() {
  command -v "$1" >/dev/null 2>&1 || error "Required tool not found: $1"
}

# ── argument parsing ──────────────────────────────────────────────────────────

GH_USER="${GAK_GITHUB_USERNAME:-}"
SYNC_INTERVAL="1h"
KEYS_PATH=""

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --username|-u)
        GH_USER="${2:-}"
        shift 2
        ;;
      --interval)
        SYNC_INTERVAL="${2:-1h}"
        shift 2
        ;;
      --keys-path)
        KEYS_PATH="${2:-}"
        shift 2
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        # treat bare first argument as username for convenience
        if [[ -z "${GH_USER}" && "$1" != -* ]]; then
          GH_USER="$1"
          shift
        else
          error "Unknown argument: $1"
        fi
        ;;
    esac
  done
}

# ── main ─────────────────────────────────────────────────────────────────────

main() {
  parse_args "$@"

  check_dependency curl
  check_dependency tar

  OS=$(detect_os)
  ARCH=$(detect_arch)

  info "Detecting latest release..."
  LATEST_TAG=$(curl -fsSL "${GITHUB_API}" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  if [[ -z "${LATEST_TAG}" ]]; then
    error "Could not determine latest release tag. Check https://github.com/${REPO}/releases"
  fi
  info "Latest release: ${LATEST_TAG}"

  TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

  mkdir -p "${INSTALL_DIR}"

  info "Downloading ${TARBALL}..."
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "${TMP_DIR}"' EXIT

  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}" || \
    error "Download failed. Check that ${DOWNLOAD_URL} exists."

  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"
  install -m755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  info "Binary installed to ${INSTALL_DIR}/${BINARY}"

  if ! echo "${PATH}" | grep -q "${INSTALL_DIR}"; then
    warn "${INSTALL_DIR} is not in your PATH. Add to your shell profile:"
    warn "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
  fi

  # Install systemd user service.
  if command -v systemctl >/dev/null 2>&1; then
    mkdir -p "${SYSTEMD_DIR}"
    SERVICE_URL="https://raw.githubusercontent.com/${REPO}/${LATEST_TAG}/systemd/${SERVICE_FILE}"
    curl -fsSL "${SERVICE_URL}" -o "${SYSTEMD_DIR}/${SERVICE_FILE}" || {
      warn "Could not download service file from ${SERVICE_URL}"
    }
    systemctl --user daemon-reload 2>/dev/null || true
    info "systemd service installed to ${SYSTEMD_DIR}/${SERVICE_FILE}"
  else
    warn "systemctl not found — skipping service installation."
    warn "Run manually: ${INSTALL_DIR}/${BINARY} --once"
  fi

  # Resolve username: flag > env var > interactive > placeholder.
  if [[ -z "${GH_USER}" ]] && [[ -t 0 ]]; then
    read -rp "Enter your GitHub username (or comma-separated list): " GH_USER </dev/tty || true
  fi

  # Write config if it doesn't exist yet.
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    mkdir -p "${CONFIG_DIR}" || { warn "Could not create config directory ${CONFIG_DIR}"; }

    KEYS_PATH_LINE='# authorized_keys_path: ""'
    if [[ -n "${KEYS_PATH}" ]]; then
      KEYS_PATH_LINE="authorized_keys_path: \"${KEYS_PATH}\""
    fi

    printf '%s\n' \
      '# github-authorized-keys configuration' \
      "github_username: \"${GH_USER:-YOUR_GITHUB_USERNAME}\"" \
      "sync_interval: \"${SYNC_INTERVAL}\"" \
      "${KEYS_PATH_LINE}" \
      'log_level: "info"' \
      > "${CONFIG_FILE}" || { warn "Could not write config to ${CONFIG_FILE}"; }

    info "Config written to ${CONFIG_FILE}"
  else
    info "Config already exists at ${CONFIG_FILE} — skipping."
  fi

  # Enable and start service if we have a real username.
  if command -v systemctl >/dev/null 2>&1; then
    if grep -q "YOUR_GITHUB_USERNAME" "${CONFIG_FILE}" 2>/dev/null; then
      warn "Edit ${CONFIG_FILE} and set github_username, then run:"
      warn "  systemctl --user enable --now github-authorized-keys"
    else
      systemctl --user enable --now "${SERVICE_FILE%.service}" 2>/dev/null && \
        info "Service enabled and started." || \
        warn "Run manually: systemctl --user enable --now github-authorized-keys"
    fi
  fi

  info "Installation complete!"
  info "Check status: systemctl --user status github-authorized-keys"
  info "View logs:    journalctl --user -u github-authorized-keys -f"
  info "Run once:     ${INSTALL_DIR}/${BINARY} --once"
}

main "$@"
