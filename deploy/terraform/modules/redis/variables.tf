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

variable "vpc_id" {
  type        = string
  description = "VPC network ID for private networking"
}

variable "memory_size_gb" {
  type        = number
  description = "Redis instance memory size in GB"
  default     = 1

  validation {
    condition     = var.memory_size_gb >= 1 && var.memory_size_gb <= 300
    error_message = "Memory size must be between 1 and 300 GB."
  }
}

variable "redis_version" {
  type        = string
  description = "Redis version"
  default     = "REDIS_7_2"
}

variable "replica_count" {
  type        = number
  description = "Number of read replicas (0 for dev, 1-2 for production)"
  default     = 0
}

variable "labels" {
  type        = map(string)
  description = "Common labels applied to all resources"
  default     = {}
}
