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

variable "namespace" {
  type        = string
  description = "Kubernetes namespace for MinIO"
  default     = "clario360"
}

variable "minio_storage_size" {
  type        = string
  description = "MinIO persistent volume size"
  default     = "50Gi"
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
