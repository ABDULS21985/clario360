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
  description = "VPC network ID"
}

variable "private_subnet_id" {
  type        = string
  description = "Private subnet ID for GKE nodes"
}

variable "pods_secondary_range_name" {
  type        = string
  description = "Name of the secondary IP range for pods"
}

variable "services_secondary_range_name" {
  type        = string
  description = "Name of the secondary IP range for services"
}

variable "kms_key_id" {
  type        = string
  description = "Cloud KMS key ID for application-layer secrets encryption"
}

# --- System node pool ---
variable "system_pool_size" {
  type        = number
  description = "Initial number of nodes in the system pool"
  default     = 1
}

variable "system_pool_min" {
  type        = number
  description = "Minimum nodes in the system pool"
  default     = 1

  validation {
    condition     = var.system_pool_min >= 1
    error_message = "System pool must have at least 1 node."
  }
}

variable "system_pool_max" {
  type        = number
  description = "Maximum nodes in the system pool"
  default     = 3
}

variable "system_machine_type" {
  type        = string
  description = "Machine type for system pool nodes"
  default     = "e2-standard-2"
}

# --- Workload node pool ---
variable "workload_pool_min" {
  type        = number
  description = "Minimum nodes in the workload pool"
  default     = 1
}

variable "workload_pool_max" {
  type        = number
  description = "Maximum nodes in the workload pool"
  default     = 6
}

variable "workload_machine_type" {
  type        = string
  description = "Machine type for workload pool nodes"
  default     = "e2-standard-4"
}

# --- Compute node pool ---
variable "compute_pool_max" {
  type        = number
  description = "Maximum nodes in the compute pool (min is always 0 — scales to zero)"
  default     = 2
}

variable "compute_machine_type" {
  type        = string
  description = "Machine type for compute pool nodes"
  default     = "e2-standard-4"
}

# --- Addons ---
variable "domain" {
  type        = string
  description = "Domain for the environment (e.g., dev.clario360.internal)"
}

variable "letsencrypt_email" {
  type        = string
  description = "Email for Let's Encrypt certificate issuance"
}

variable "gitops_repo_url" {
  type        = string
  description = "Git repository URL for ArgoCD GitOps"
  default     = "https://github.com/clario360/clario360-gitops.git"
}

variable "labels" {
  type        = map(string)
  description = "Common labels applied to all resources"
  default     = {}
}
