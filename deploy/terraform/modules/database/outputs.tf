output "instance_name" {
  description = "Cloud SQL instance name"
  value       = google_sql_database_instance.main.name
}

output "instance_connection_name" {
  description = "Cloud SQL instance connection name (project:region:instance)"
  value       = google_sql_database_instance.main.connection_name
}

output "private_ip_address" {
  description = "Private IP address of the Cloud SQL instance"
  value       = google_sql_database_instance.main.private_ip_address
}

output "database_names" {
  description = "List of database names"
  value       = [for db in google_sql_database.databases : db.name]
}

output "service_user_names" {
  description = "Map of service to database username"
  value       = { for k, v in google_sql_user.service_users : k => v.name }
}

output "backup_bucket" {
  description = "GCS bucket for database backup exports"
  value       = google_storage_bucket.db_backups.name
}

output "vault_secret_paths" {
  description = "Map of service to Vault secret path for database credentials"
  value       = { for k, v in vault_generic_secret.db_passwords : k => v.path }
}
