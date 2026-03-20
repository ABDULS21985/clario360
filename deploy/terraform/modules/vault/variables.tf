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
  description = "Kubernetes namespace for Vault"
  default     = "vault"
}

variable "kms_key_ring_id" {
  type        = string
  description = "KMS key ring ID for auto-unseal"
}

variable "storage_size" {
  type        = string
  description = "Vault storage size"
  default     = "10Gi"
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
