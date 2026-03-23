variable "host" {
  description = "IP address or hostname of the target server."
  type        = string
}

variable "ssh_user" {
  description = "SSH user to connect as."
  type        = string
  default     = "ubuntu"
}

variable "ssh_private_key" {
  description = "Contents of the SSH private key (not the path). Leave empty to use the SSH agent."
  type        = string
  default     = ""
  sensitive   = true
}

variable "ssh_port" {
  description = "SSH port on the target server."
  type        = number
  default     = 22
}

variable "github_username" {
  description = "GitHub username(s) whose public SSH keys to sync. Comma-separate multiple users."
  type        = string
}

variable "sync_interval" {
  description = "How often to sync keys, as a Go duration string (e.g. 30m, 1h, 6h)."
  type        = string
  default     = "1h"
}

variable "version" {
  description = "Release version to install (e.g. v1.0.0). Triggers re-install if changed."
  type        = string
  default     = "latest"
}
