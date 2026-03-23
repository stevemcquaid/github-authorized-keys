terraform {
  required_version = ">= 1.3"
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }
}

# Run the one-shot install script on a remote host via SSH.
resource "null_resource" "install_github_authorized_keys" {
  # Re-run if the username or version changes.
  triggers = {
    github_username = var.github_username
    version         = var.version
    interval        = var.sync_interval
  }

  connection {
    type        = "ssh"
    host        = var.host
    user        = var.ssh_user
    private_key = var.ssh_private_key != "" ? var.ssh_private_key : null
    port        = var.ssh_port
    timeout     = "2m"
  }

  provisioner "remote-exec" {
    inline = [
      # Download and run the installer with username pre-configured.
      "curl -fsSL https://raw.githubusercontent.com/stevemcquaid/github-authorized-keys/main/install.sh | bash -s -- --username '${var.github_username}' --interval '${var.sync_interval}'"
    ]
  }
}
