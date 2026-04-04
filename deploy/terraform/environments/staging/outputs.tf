output "cluster_name" {
  description = "GKE cluster name"
  value       = module.kubernetes.cluster_name
}

output "cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = module.kubernetes.cluster_endpoint
  sensitive   = true
}

output "database_instance" {
  description = "Cloud SQL instance name"
  value       = module.database.instance_name
}

output "database_private_ip" {
  description = "Cloud SQL private IP"
  value       = module.database.private_ip_address
}

output "redis_host" {
  description = "Redis host"
  value       = module.redis.host
}

output "kafka_bootstrap" {
  description = "Kafka bootstrap servers"
  value       = module.kafka.bootstrap_servers
}

output "kafka_bootstrap_tls" {
  description = "Kafka TLS bootstrap servers"
  value       = module.kafka.bootstrap_servers_tls
}

output "grafana_url" {
  description = "Grafana dashboard URL"
  value       = module.monitoring.grafana_url
}

output "vault_endpoint" {
  description = "Vault endpoint"
  value       = module.vault.vault_endpoint
}

output "dns_name_servers" {
  description = "DNS name servers (configure at registrar)"
  value       = module.dns.name_servers
}
