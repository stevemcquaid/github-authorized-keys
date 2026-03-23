# github-authorized-keys

A lightweight Go service that periodically syncs your GitHub public SSH keys into `~/.ssh/authorized_keys`.

Keys are written inside a clearly-marked managed block — any keys you add manually outside the block are never touched.

```
# BEGIN github-authorized-keys
# Managed by github-authorized-keys — do not edit this block manually
# Last synced: 2026-03-23T12:00:00Z — source: octocat
ssh-ed25519 AAAA...
# END github-authorized-keys
```

---

## One-line install (recommended)

Installs the binary, configures the service, and starts it — fully working in one command:

```bash
curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash -s -- --username YOUR_GITHUB_USER
```

Or with multiple users:

```bash
curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash -s -- --username user1,user2
```

Or via environment variable:

```bash
curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | GAK_GITHUB_USERNAME=YOUR_GITHUB_USER bash
```

### Installer flags

| Flag | Default | Description |
|------|---------|-------------|
| `--username`, `-u` | — | GitHub username(s), comma-separated. **Required.** |
| `--interval` | `1h` | Sync interval (Go duration: `30m`, `1h`, `6h`) |
| `--keys-path` | `~/.ssh/authorized_keys` | Override authorized_keys path |

The installer:
1. Detects your OS and architecture (linux/darwin × amd64/arm64)
2. Downloads the latest release binary from GitHub Releases
3. Installs it to `~/.local/bin/`
4. Writes `~/.config/github-authorized-keys/config.yaml`
5. Installs and enables a systemd user service

---

## Install from source

```bash
git clone https://github.com/stevemcquaid/github-authorized-keys
cd github-authorized-keys
make install
```

---

## Configuration

Config file: `~/.config/github-authorized-keys/config.yaml`

```yaml
# Single username
github_username: "octocat"

# Multiple usernames
github_username:
  - user1
  - user2

sync_interval: "1h"        # Go duration: 30m, 1h, 6h, 24h
# authorized_keys_path: "" # defaults to ~/.ssh/authorized_keys
log_level: "info"          # debug | info | warn | error
```

### Environment variable overrides

| Variable | Description |
|----------|-------------|
| `GAK_GITHUB_USERNAME` | Comma-separated GitHub usernames |
| `GAK_SYNC_INTERVAL` | Go duration (e.g. `30m`, `1h`) |
| `GAK_AUTHORIZED_KEYS_PATH` | Override authorized_keys path |
| `GAK_LOG_LEVEL` | `debug` / `info` / `warn` / `error` |

---

## Service management

```bash
# Status
systemctl --user status github-authorized-keys

# Logs
journalctl --user -u github-authorized-keys -f

# Force immediate re-sync (also reloads config)
kill -HUP $(systemctl --user show -p MainPID --value github-authorized-keys)

# Uninstall
make uninstall
```

---

## Binary usage

```bash
# Run as a service (loops on configured interval)
github-authorized-keys

# Run once and exit — useful for cron
github-authorized-keys --once

# Custom config file
github-authorized-keys --config /path/to/config.yaml
```

---

## Automation examples

For fleet deployments, see the [`examples/`](examples/) directory.

### Ansible

```bash
cd examples/ansible
# Edit playbook.yml to set your GitHub username, then:
ansible-playbook -i inventory.yml playbook.yml

# Or override username at the command line:
ansible-playbook -i inventory.yml playbook.yml -e "gak_github_username=octocat"
```

See [`examples/ansible/`](examples/ansible/) for the full role with defaults, handlers, and templates.

### Terraform

```bash
cd examples/terraform
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your host and GitHub username
terraform init
terraform apply
```

See [`examples/terraform/`](examples/terraform/) for variables and outputs.

### Pulumi

```bash
cd examples/pulumi
pip install -r requirements.txt
pulumi stack init prod
pulumi config set host 192.168.1.100
pulumi config set github_username octocat
pulumi config set --secret ssh_private_key "$(cat ~/.ssh/id_ed25519)"
pulumi up
```

See [`examples/pulumi/`](examples/pulumi/) for the full program.

---

## Releasing a new version

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions builds binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and publishes them to GitHub Releases automatically. Works on free GitHub plans.
