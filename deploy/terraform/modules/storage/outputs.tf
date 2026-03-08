output "bucket_names" {
  description = "Map of bucket purpose to GCS bucket name"
  value       = { for k, v in google_storage_bucket.buckets : k => v.name }
}

output "bucket_urls" {
  description = "Map of bucket purpose to GCS URL"
  value       = { for k, v in google_storage_bucket.buckets : k => v.url }
}

output "storage_service_account_email" {
  description = "Service account email for storage access"
  value       = google_service_account.storage_access.email
}

output "minio_endpoint" {
  description = "MinIO internal endpoint"
  value       = "http://minio.${var.namespace}.svc.cluster.local:9000"
}

output "minio_root_user" {
  description = "MinIO root username"
  value       = "clario360-admin"
  sensitive   = true
}

output "minio_root_password" {
  description = "MinIO root password"
  value       = random_password.minio_secret_key.result
  sensitive   = true
}
