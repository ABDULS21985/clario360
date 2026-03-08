output "service_account_emails" {
  description = "Map of service to GCP service account email"
  value       = { for k, v in google_service_account.services : k => v.email }
}

output "kms_key_ring_id" {
  description = "KMS key ring ID"
  value       = google_kms_key_ring.main.id
}

output "kms_key_id" {
  description = "KMS key ID for GKE secrets encryption"
  value       = google_kms_crypto_key.gke_secrets.id
}

output "kms_data_key_id" {
  description = "KMS key ID for application data encryption"
  value       = google_kms_crypto_key.data_encryption.id
}

output "kms_database_key_id" {
  description = "KMS key ID for database column encryption"
  value       = google_kms_crypto_key.database_encryption.id
}
