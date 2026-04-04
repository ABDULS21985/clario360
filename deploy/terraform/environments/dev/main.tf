# =============================================================================
# Clario 360 — Dev Environment Composition
# Minimal resources, IAP SSH enabled, single-replica services
# =============================================================================

terraform {
  required_version = ">= 1.7.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.20"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.20"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "~> 4.2"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

# Configure Kubernetes and Helm providers after GKE is created
provider "kubernetes" {
  host                   = "https://${module.kubernetes.cluster_endpoint}"
  cluster_ca_certificate = base64decode(module.kubernetes.cluster_ca_certificate)
  token                  = data.google_client_config.current.access_token
}

provider "helm" {
  kubernetes {
    host                   = "https://${module.kubernetes.cluster_endpoint}"
    cluster_ca_certificate = base64decode(module.kubernetes.cluster_ca_certificate)
    token                  = data.google_client_config.current.access_token
  }
}

provider "vault" {
  address = module.vault.vault_endpoint
}

data "google_client_config" "current" {}

locals {
  environment = "dev"
  common_labels = {
    project     = "clario360"
    environment = "dev"
    managed-by  = "terraform"
  }
}

# --- Networking ---
module "networking" {
  source = "../../modules/networking"

  project_id     = var.project_id
  region         = var.region
  environment    = local.environment
  enable_iap_ssh = true
  labels         = local.common_labels
}

# --- Security (must be before kubernetes for KMS key) ---
module "security" {
  source = "../../modules/security"

  project_id       = var.project_id
  region           = var.region
  environment      = local.environment
  gke_cluster_name = module.kubernetes.cluster_name
  labels           = local.common_labels
}

# --- Kubernetes ---
module "kubernetes" {
  source = "../../modules/kubernetes"

  project_id                    = var.project_id
  region                        = var.region
  environment                   = local.environment
  vpc_id                        = module.networking.vpc_id
  private_subnet_id             = module.networking.private_subnet_id
  pods_secondary_range_name     = module.networking.pods_secondary_range_name
  services_secondary_range_name = module.networking.services_secondary_range_name
  kms_key_id                    = module.security.kms_key_id

  # Minimal node pools for dev
  system_pool_size      = 1
  system_pool_min       = 1
  system_pool_max       = 2
  system_machine_type   = "e2-standard-2"
  workload_pool_min     = 1
  workload_pool_max     = 3
  workload_machine_type = "e2-standard-4"
  compute_pool_max      = 1
  compute_machine_type  = "e2-standard-4"

  domain            = "dev.clario360.internal"
  letsencrypt_email = var.letsencrypt_email
  gitops_repo_url   = var.gitops_repo_url
  labels            = local.common_labels
}

# --- Database ---
module "database" {
  source = "../../modules/database"

  project_id              = var.project_id
  region                  = var.region
  environment             = local.environment
  vpc_self_link           = module.networking.vpc_self_link
  psa_connection_id       = module.networking.psa_connection_id
  db_tier                 = "db-custom-2-4096"
  db_disk_size_gb         = 20
  db_disk_max_gb          = 50
  db_max_connections      = "100"
  db_shared_buffers       = "1024MB"
  db_effective_cache      = "3072MB"
  db_work_mem             = "4MB"
  db_maintenance_work_mem = "64MB"
  labels                  = local.common_labels
}

# --- Redis ---
module "redis" {
  source = "../../modules/redis"

  project_id     = var.project_id
  region         = var.region
  environment    = local.environment
  vpc_id         = module.networking.vpc_id
  memory_size_gb = 1
  replica_count  = 0
  labels         = local.common_labels
}

# --- Kafka ---
module "kafka" {
  source = "../../modules/kafka"

  environment          = local.environment
  kafka_replicas       = 1
  kafka_storage_size   = "10Gi"
  kafka_memory_request = "1Gi"
  kafka_memory_limit   = "2Gi"
  kafka_cpu_request    = "500m"
  kafka_cpu_limit      = "1000m"
  labels               = local.common_labels
}

# --- Storage ---
module "storage" {
  source = "../../modules/storage"

  project_id         = var.project_id
  region             = var.region
  environment        = local.environment
  namespace          = module.kubernetes.clario360_namespace
  minio_storage_size = "20Gi"
  labels             = local.common_labels
}

# --- Monitoring ---
module "monitoring" {
  source = "../../modules/monitoring"

  environment       = local.environment
  namespace         = module.kubernetes.monitoring_namespace
  domain            = "dev.clario360.internal"
  retention_days    = 7
  storage_size      = "20Gi"
  loki_storage_size = "20Gi"
  slack_webhook_url = var.slack_webhook_url
  labels            = local.common_labels
}

# --- Vault ---
module "vault" {
  source = "../../modules/vault"

  project_id      = var.project_id
  region          = var.region
  environment     = local.environment
  namespace       = module.kubernetes.vault_namespace
  kms_key_ring_id = module.security.kms_key_ring_id
  storage_size    = "5Gi"
  labels          = local.common_labels
}

# DNS omitted in dev — uses nip.io or /etc/hosts
