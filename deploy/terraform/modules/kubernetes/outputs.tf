output "cluster_id" {
  description = "The ID of the GKE cluster"
  value       = google_container_cluster.main.id
}

output "cluster_name" {
  description = "The name of the GKE cluster"
  value       = google_container_cluster.main.name
}

output "cluster_endpoint" {
  description = "The endpoint of the GKE cluster"
  value       = google_container_cluster.main.endpoint
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "The CA certificate of the GKE cluster"
  value       = google_container_cluster.main.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "clario360_namespace" {
  description = "The main application namespace"
  value       = kubernetes_namespace.clario360.metadata[0].name
}

output "jobs_namespace" {
  description = "The batch processing namespace"
  value       = kubernetes_namespace.clario360_jobs.metadata[0].name
}

output "monitoring_namespace" {
  description = "The monitoring namespace"
  value       = kubernetes_namespace.monitoring.metadata[0].name
}

output "vault_namespace" {
  description = "The Vault namespace"
  value       = kubernetes_namespace.vault.metadata[0].name
}

output "service_accounts" {
  description = "Map of service name to Kubernetes service account name"
  value       = { for k, v in kubernetes_service_account.services : k => v.metadata[0].name }
}
