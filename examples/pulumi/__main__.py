"""
Pulumi program — deploy github-authorized-keys to a remote host.

Requirements:
    pip install pulumi pulumi-command

Usage:
    pulumi config set host 192.168.1.100
    pulumi config set github_username octocat
    pulumi config set --secret ssh_private_key "$(cat ~/.ssh/id_ed25519)"
    pulumi up
"""

import pulumi
import pulumi_command as command

cfg = pulumi.Config()

host           = cfg.require("host")
ssh_user       = cfg.get("ssh_user") or "ubuntu"
ssh_port       = cfg.get_int("ssh_port") or 22
github_username = cfg.require("github_username")
sync_interval  = cfg.get("sync_interval") or "1h"

# SSH private key is stored as a Pulumi secret (never logged in plaintext).
ssh_private_key = cfg.get_secret("ssh_private_key")

connection = command.remote.ConnectionArgs(
    host=host,
    user=ssh_user,
    port=ssh_port,
    private_key=ssh_private_key,
)

install = command.remote.Command(
    "install-github-authorized-keys",
    connection=connection,
    # create: run on first deploy and whenever inputs change
    create=pulumi.Output.format(
        "curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh"
        " | bash -s -- --username '{0}' --interval '{1}'",
        github_username,
        sync_interval,
    ),
    # delete: uninstall when the resource is destroyed
    delete=(
        "systemctl --user disable --now github-authorized-keys 2>/dev/null || true && "
        "rm -f ~/.local/bin/github-authorized-keys "
        "~/.config/systemd/user/github-authorized-keys.service "
        "~/.config/github-authorized-keys/config.yaml"
    ),
    opts=pulumi.ResourceOptions(
        # Re-provision if username or interval changes.
        replace_on_changes=["create"],
    ),
)

pulumi.export("host", host)
pulumi.export("install_stdout", install.stdout)
