# github-authorized-keys

A lightweight Go service that periodically syncs your GitHub public SSH keys into `~/.ssh/authorized_keys`.

Keys are written inside a clearly-marked managed block — any keys you add manually outside the block are never touched.

## Quick Install (no Go required)

```bash
curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash
```

The installer will:
1. Download the correct binary for your OS/arch from GitHub Releases
2. Install it to `~/.local/bin/`
3. Install and enable a systemd user service
4. Prompt you to create a config file

## Install from source

```bash
git clone https://github.com/stevemcquaid/github-authorized-keys
cd github-authorized-keys
make install
```

## Configuration

Create `~/.config/github-authorized-keys/config.yaml`:

```yaml
github_username: "your-github-username"
sync_interval: "1h"
log_level: "info"
```

Multiple users:

```yaml
github_username:
  - user1
  - user2
```

Or via environment variables:

| Variable                  | Description                          |
|---------------------------|--------------------------------------|
| `GAK_GITHUB_USERNAME`     | Comma-separated GitHub usernames     |
| `GAK_SYNC_INTERVAL`       | Go duration (e.g. `30m`, `1h`)       |
| `GAK_AUTHORIZED_KEYS_PATH`| Override authorized_keys path        |
| `GAK_LOG_LEVEL`           | `debug` / `info` / `warn` / `error`  |

## Usage

```bash
# Run as a service (default — loops on configured interval)
github-authorized-keys

# Run once and exit
github-authorized-keys --once

# Use a custom config file
github-authorized-keys --config /path/to/config.yaml
```

## Service management

```bash
# Status
systemctl --user status github-authorized-keys

# Logs
journalctl --user -u github-authorized-keys -f

# Force an immediate re-sync
kill -HUP $(systemctl --user show -p MainPID --value github-authorized-keys)

# Reload config and re-sync (also triggered by SIGHUP above)
systemctl --user reload github-authorized-keys

# Uninstall
make uninstall
```

## How it works

Keys are fetched from `https://github.com/<username>.keys` (GitHub's public key endpoint, no auth required).

The managed section in `~/.ssh/authorized_keys` looks like:

```
# BEGIN github-authorized-keys
# Managed by github-authorized-keys — do not edit this block manually
# Last synced: 2026-03-23T12:00:00Z — source: octocat
ssh-ed25519 AAAA...
# END github-authorized-keys
```

Any SSH keys you add outside this block are preserved across syncs.

## Releasing a new version

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will build binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and publish them to GitHub Releases automatically (works on free GitHub plans).
