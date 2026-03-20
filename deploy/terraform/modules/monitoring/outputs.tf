output "grafana_url" {
  description = "Grafana dashboard URL"
  value       = "https://grafana.${var.domain}"
}

output "grafana_admin_password" {
  description = "Grafana admin password"
  value       = local.grafana_password
  sensitive   = true
}

output "prometheus_endpoint" {
  description = "Prometheus internal endpoint"
  value       = "http://kube-prometheus-stack-prometheus.${var.namespace}.svc.cluster.local:9090"
}

output "alertmanager_endpoint" {
  description = "AlertManager internal endpoint"
  value       = "http://kube-prometheus-stack-alertmanager.${var.namespace}.svc.cluster.local:9093"
}

output "loki_endpoint" {
  description = "Loki internal endpoint"
  value       = "http://loki.${var.namespace}.svc.cluster.local:3100"
}
