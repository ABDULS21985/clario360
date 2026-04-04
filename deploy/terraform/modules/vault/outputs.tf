output "vault_endpoint" {
  description = "Vault internal endpoint"
  value       = "http://vault.${var.namespace}.svc.cluster.local:8200"
}

output "vault_service_account" {
  description = "Vault GCP service account email"
  value       = google_service_account.vault.email
}

output "kv_mount_path" {
  description = "KV v2 secret engine mount path"
  value       = vault_mount.kv.path
}

output "pki_intermediate_path" {
  description = "Intermediate PKI mount path"
  value       = vault_mount.pki_intermediate.path
}

output "service_policy_names" {
  description = "Map of service to Vault policy name"
  value       = { for k, v in vault_policy.service_policies : k => v.name }
}
