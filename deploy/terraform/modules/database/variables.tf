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

variable "vpc_self_link" {
  type        = string
  description = "VPC self link for private networking"
}

variable "psa_connection_id" {
  type        = string
  description = "Private Service Access connection ID (dependency for Cloud SQL)"
}

# --- Instance sizing ---
variable "db_tier" {
  type        = string
  description = "Cloud SQL machine tier"
  default     = "db-custom-2-4096"

  validation {
    condition     = can(regex("^db-", var.db_tier))
    error_message = "Database tier must start with 'db-'."
  }
}

variable "db_disk_size_gb" {
  type        = number
  description = "Initial disk size in GB"
  default     = 20

  validation {
    condition     = var.db_disk_size_gb >= 10
    error_message = "Disk size must be at least 10 GB."
  }
}

variable "db_disk_max_gb" {
  type        = number
  description = "Maximum disk size for autoresize in GB"
  default     = 50
}

# --- Performance flags ---
variable "db_max_connections" {
  type        = string
  description = "PostgreSQL max_connections"
  default     = "100"
}

variable "db_shared_buffers" {
  type        = string
  description = "PostgreSQL shared_buffers (25% of RAM recommended)"
  default     = "1024MB"
}

variable "db_effective_cache" {
  type        = string
  description = "PostgreSQL effective_cache_size (75% of RAM recommended)"
  default     = "3072MB"
}

variable "db_work_mem" {
  type        = string
  description = "PostgreSQL work_mem"
  default     = "4MB"
}

variable "db_maintenance_work_mem" {
  type        = string
  description = "PostgreSQL maintenance_work_mem"
  default     = "64MB"
}

variable "backup_bucket" {
  type        = string
  description = "GCS bucket name for long-term database backup exports"
  default     = ""
}

variable "labels" {
  type        = map(string)
  description = "Common labels applied to all resources"
  default     = {}
}
