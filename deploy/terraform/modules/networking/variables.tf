variable "project_id" {
  type        = string
  description = "GCP project ID"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "Project ID must be 6-30 characters, lowercase letters, digits, and hyphens."
  }
}

variable "region" {
  type        = string
  description = "GCP region for resource deployment"
  default     = "me-central1"

  validation {
    condition     = can(regex("^[a-z]+-[a-z]+[0-9]$", var.region))
    error_message = "Region must be a valid GCP region identifier."
  }
}

variable "environment" {
  type        = string
  description = "Environment name (dev, staging, production)"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "public_subnet_cidr" {
  type        = string
  description = "CIDR range for the public subnet (load balancer backends, NAT)"
  default     = "10.0.0.0/24"

  validation {
    condition     = can(cidrhost(var.public_subnet_cidr, 0))
    error_message = "Must be a valid CIDR block."
  }
}

variable "private_subnet_cidr" {
  type        = string
  description = "CIDR range for the private subnet (GKE nodes, internal services)"
  default     = "10.0.1.0/24"

  validation {
    condition     = can(cidrhost(var.private_subnet_cidr, 0))
    error_message = "Must be a valid CIDR block."
  }
}

variable "isolated_subnet_cidr" {
  type        = string
  description = "CIDR range for the isolated subnet (Cloud SQL, Redis, Vault — no internet)"
  default     = "10.0.2.0/24"

  validation {
    condition     = can(cidrhost(var.isolated_subnet_cidr, 0))
    error_message = "Must be a valid CIDR block."
  }
}

variable "pods_cidr" {
  type        = string
  description = "Secondary CIDR range for GKE pods"
  default     = "10.1.0.0/16"

  validation {
    condition     = can(cidrhost(var.pods_cidr, 0))
    error_message = "Must be a valid CIDR block."
  }
}

variable "services_cidr" {
  type        = string
  description = "Secondary CIDR range for GKE services"
  default     = "10.2.0.0/20"

  validation {
    condition     = can(cidrhost(var.services_cidr, 0))
    error_message = "Must be a valid CIDR block."
  }
}

variable "enable_iap_ssh" {
  type        = bool
  description = "Enable IAP SSH tunneling (dev/staging only, not for production)"
  default     = false
}

variable "labels" {
  type        = map(string)
  description = "Common labels applied to all resources"
  default     = {}
}
