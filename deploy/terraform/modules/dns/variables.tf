variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "environment" {
  type        = string
  description = "Environment name"

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be one of: dev, staging, production."
  }
}

variable "domain" {
  type        = string
  description = "Domain name for the environment (e.g., clario360.app)"
}

variable "dns_zone_name" {
  type        = string
  description = "Cloud DNS managed zone name"
  default     = ""
}

variable "create_zone" {
  type        = bool
  description = "Whether to create a new DNS zone or use an existing one"
  default     = true
}

variable "ingress_ip" {
  type        = string
  description = "External IP of the ingress load balancer"
  default     = ""
}

variable "labels" {
  type        = map(string)
  description = "Common labels"
  default     = {}
}
