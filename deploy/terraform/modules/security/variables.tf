variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "me-central1"
}

variable "environment" {
  type        = string
  description = "Environment name"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "gke_cluster_name" {
  type        = string
  description = "GKE cluster name for workload identity bindings"
  default     = ""
}

variable "container_registry" {
  type        = string
  description = "Container registry URL for Binary Authorization"
  default     = ""
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
