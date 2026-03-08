# =============================================================================
# Clario 360 — Staging Environment Composition
# Moderate resources, 3 Kafka replicas, real DNS, SSL required
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
  environment = "staging"
  common_labels = {
    project     = "clario360"
    environment = "staging"
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

# --- Security ---
module "security" {
  source = "../../modules/security"

  project_id         = var.project_id
  region             = var.region
  environment        = local.environment
  gke_cluster_name   = module.kubernetes.cluster_name
  container_registry = var.container_registry
  labels             = local.common_labels
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

  system_pool_size      = 2
  system_pool_min       = 2
  system_pool_max       = 3
  system_machine_type   = "e2-standard-4"
  workload_pool_min     = 2
  workload_pool_max     = 6
  workload_machine_type = "e2-standard-4"
  compute_pool_max      = 2
  compute_machine_type  = "e2-standard-4"

  domain            = var.domain
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
  db_tier                 = "db-custom-4-8192"
  db_disk_size_gb         = 50
  db_disk_max_gb          = 100
  db_max_connections      = "200"
  db_shared_buffers       = "2048MB"
  db_effective_cache      = "6144MB"
  db_work_mem             = "8MB"
  db_maintenance_work_mem = "256MB"
  labels                  = local.common_labels
}

# --- Redis ---
module "redis" {
  source = "../../modules/redis"

  project_id     = var.project_id
  region         = var.region
  environment    = local.environment
  vpc_id         = module.networking.vpc_id
  memory_size_gb = 2
  replica_count  = 0
  labels         = local.common_labels
}

# --- Kafka ---
module "kafka" {
  source = "../../modules/kafka"

  environment          = local.environment
  kafka_replicas       = 3
  kafka_storage_size   = "50Gi"
  kafka_memory_request = "2Gi"
  kafka_memory_limit   = "4Gi"
  kafka_cpu_request    = "1000m"
  kafka_cpu_limit      = "2000m"
  labels               = local.common_labels
}

# --- Storage ---
module "storage" {
  source = "../../modules/storage"

  project_id         = var.project_id
  region             = var.region
  environment        = local.environment
  namespace          = module.kubernetes.clario360_namespace
  minio_storage_size = "50Gi"
  labels             = local.common_labels
}

# --- DNS ---
module "dns" {
  source = "../../modules/dns"

  project_id  = var.project_id
  environment = local.environment
  domain      = var.domain
  create_zone = true
  labels      = local.common_labels
}

# --- Monitoring ---
module "monitoring" {
  source = "../../modules/monitoring"

  environment       = local.environment
  namespace         = module.kubernetes.monitoring_namespace
  domain            = var.domain
  retention_days    = 15
  storage_size      = "50Gi"
  loki_storage_size = "50Gi"
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
  storage_size    = "10Gi"
  labels          = local.common_labels
}
