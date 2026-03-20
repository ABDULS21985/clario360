variable "project_id" {
  type        = string
  description = "GCP project ID"
}

variable "region" {
  type        = string
  description = "GCP region"
  default     = "me-central1"
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

variable "slack_webhook_url" {
  type        = string
  description = "Slack webhook URL for alert notifications"
  default     = ""
  sensitive   = true
}
