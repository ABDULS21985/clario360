variable "environment" {
  type        = string
  description = "Environment name"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "namespace" {
  type        = string
  description = "Kubernetes namespace for monitoring stack"
  default     = "monitoring"
}

variable "domain" {
  type        = string
  description = "Domain for Grafana ingress"
}

variable "grafana_admin_password" {
  type        = string
  description = "Grafana admin password (auto-generated if empty)"
  default     = ""
  sensitive   = true
}

variable "slack_webhook_url" {
  type        = string
  description = "Slack webhook URL for alert notifications"
  default     = ""
  sensitive   = true
}

variable "retention_days" {
  type        = number
  description = "Prometheus data retention in days"
  default     = 15
}

variable "storage_size" {
  type        = string
  description = "Prometheus persistent storage size"
  default     = "50Gi"
}

variable "loki_storage_size" {
  type        = string
  description = "Loki persistent storage size"
  default     = "50Gi"
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
