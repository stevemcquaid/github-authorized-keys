#!/usr/bin/env bash
# install.sh — one-liner installer for github-authorized-keys
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash
#
# The script:
#   1. Detects OS and architecture
#   2. Downloads the latest release binary from GitHub Releases
#   3. Installs it to ~/.local/bin/
#   4. Installs the systemd user service
#   5. Prompts you to create a config file if one doesn't exist

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

# ── main ─────────────────────────────────────────────────────────────────────

main() {
  check_dependency curl
  check_dependency tar

  OS=$(detect_os)
  ARCH=$(detect_arch)

  info "Detecting latest release..."
  LATEST_TAG=$(curl -fsSL "${GITHUB_API}" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  if [[ -z "${LATEST_TAG}" ]]; then
    error "Could not determine latest release tag. Check your internet connection or visit https://github.com/${REPO}/releases"
  fi
  info "Latest release: ${LATEST_TAG}"

  # Build download URL — goreleaser naming convention.
  TARBALL="${BINARY}_${OS}_${ARCH}.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

  # Create install directory.
  mkdir -p "${INSTALL_DIR}"

  info "Downloading ${TARBALL}..."
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "${TMP_DIR}"' EXIT

  curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}" || \
    error "Download failed. Check that ${DOWNLOAD_URL} exists."

  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

  install -m755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  info "Binary installed to ${INSTALL_DIR}/${BINARY}"

  # Ensure ~/.local/bin is in PATH hint.
  if ! echo "${PATH}" | grep -q "${INSTALL_DIR}"; then
    warn "${INSTALL_DIR} is not in your PATH. Add the following to your shell profile:"
    warn "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
  fi

  # Install systemd user service.
  if command -v systemctl >/dev/null 2>&1; then
    mkdir -p "${SYSTEMD_DIR}"
    # Download the service file from the same release tag.
    SERVICE_URL="https://raw.githubusercontent.com/${REPO}/${LATEST_TAG}/systemd/${SERVICE_FILE}"
    curl -fsSL "${SERVICE_URL}" -o "${SYSTEMD_DIR}/${SERVICE_FILE}" || {
      warn "Could not download service file. You can install it manually from the repository."
    }
    systemctl --user daemon-reload 2>/dev/null || true
    info "systemd service installed to ${SYSTEMD_DIR}/${SERVICE_FILE}"
  else
    warn "systemctl not found — skipping systemd service installation."
    warn "You can run the binary manually: ${INSTALL_DIR}/${BINARY} --once"
  fi

  # Create a starter config if one doesn't exist.
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    mkdir -p "${CONFIG_DIR}"

    # Prompt for username only when stdin is a real terminal (not curl|bash pipe).
    GH_USER=""
    if [[ -t 0 ]]; then
      read -rp "Enter your GitHub username (or comma-separated list): " GH_USER </dev/tty
    fi

    cat > "${CONFIG_FILE}" <<EOF
# github-authorized-keys configuration
# Edit this file, then run: systemctl --user restart github-authorized-keys
github_username: "${GH_USER:-YOUR_GITHUB_USERNAME}"

# How often to sync keys (Go duration: 1h, 30m, etc.)
sync_interval: "1h"

# Optional: override the authorized_keys path
# authorized_keys_path: ""

# Log level: debug | info | warn | error
log_level: "info"
EOF
    if [[ -z "${GH_USER}" ]]; then
      warn "Config written to ${CONFIG_FILE} — edit it to set your GitHub username before starting the service."
    else
      info "Config written to ${CONFIG_FILE}"
    fi
  else
    info "Config already exists at ${CONFIG_FILE} — skipping."
  fi

  # Enable and start the service (only if config has a real username).
  if command -v systemctl >/dev/null 2>&1; then
    if grep -q "YOUR_GITHUB_USERNAME" "${CONFIG_FILE}" 2>/dev/null; then
      warn "Edit ${CONFIG_FILE} and set github_username, then run:"
      warn "  systemctl --user enable --now github-authorized-keys"
    else
      systemctl --user enable --now "${SERVICE_FILE%.service}" 2>/dev/null && \
        info "Service enabled and started." || \
        warn "Could not start service automatically. Run: systemctl --user enable --now github-authorized-keys"
    fi
  fi

  info "Installation complete!"
  info "Check service status: systemctl --user status github-authorized-keys"
  info "View logs:            journalctl --user -u github-authorized-keys -f"
  info "Run once manually:    ${INSTALL_DIR}/${BINARY} --once"
}

main "$@"
