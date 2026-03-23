output "install_id" {
  description = "Unique ID of the install resource — changes when install is re-triggered."
  value       = null_resource.install_github_authorized_keys.id
}
