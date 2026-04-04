# =============================================================================
# Clario 360 — Kubernetes Module
# GKE private cluster with 3 node pools: system, workload, compute
# Workload Identity, Shielded Nodes, Dataplane V2, Binary Authorization
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
}

# -----------------------------------------------------------------------------
# GKE Cluster
# -----------------------------------------------------------------------------
resource "google_container_cluster" "main" {
  provider = google-beta

  project  = var.project_id
  name     = local.name_prefix
  location = var.region

  network    = var.vpc_id
  subnetwork = var.private_subnet_id

  # Private cluster configuration
  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = var.environment == "production"
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  # IP allocation for pods and services
  ip_allocation_policy {
    cluster_secondary_range_name  = var.pods_secondary_range_name
    services_secondary_range_name = var.services_secondary_range_name
  }

  # Security
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  binary_authorization {
    evaluation_mode = var.environment == "production" ? "PROJECT_SINGLETON_POLICY_ENFORCE" : "DISABLED"
  }

  enable_shielded_nodes = true

  database_encryption {
    state    = "ENCRYPTED"
    key_name = var.kms_key_id
  }

  # Networking
  networking_mode = "VPC_NATIVE"
  datapath_provider = "ADVANCED_DATAPATH"

  network_policy {
    enabled  = true
    provider = "CALICO"
  }

  # Maintenance window: Saturday 02:00-06:00 UTC
  maintenance_policy {
    recurring_window {
      start_time = "2026-01-01T02:00:00Z"
      end_time   = "2026-01-01T06:00:00Z"
      recurrence = "FREQ=WEEKLY;BYDAY=SA"
    }
  }

  # Logging & Monitoring
  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
    managed_prometheus {
      enabled = true
    }
  }

  # Release channel
  release_channel {
    channel = var.environment == "production" ? "STABLE" : "REGULAR"
  }

  # Remove default node pool — we define our own
  remove_default_node_pool = true
  initial_node_count       = 1

  resource_labels = merge(var.labels, {
    environment = var.environment
    component   = "kubernetes"
    managed-by  = "terraform"
  })
}

# -----------------------------------------------------------------------------
# Node Pool 1: System
# Purpose: cluster infrastructure (kube-system, ingress, cert-manager, monitoring)
# -----------------------------------------------------------------------------
resource "google_container_node_pool" "system" {
  project    = var.project_id
  name       = "system-pool"
  location   = var.region
  cluster    = google_container_cluster.main.name
  node_count = var.system_pool_size

  autoscaling {
    min_node_count = var.system_pool_min
    max_node_count = var.system_pool_max
  }

  node_config {
    machine_type = var.system_machine_type
    disk_size_gb = 50
    disk_type    = "pd-ssd"

    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = {
      pool        = "system"
      environment = var.environment
    }

    taint {
      key    = "dedicated"
      value  = "system"
      effect = "NO_SCHEDULE"
    }

    tags = ["gke-node", "system-pool"]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# -----------------------------------------------------------------------------
# Node Pool 2: Workload
# Purpose: Clario 360 application services
# -----------------------------------------------------------------------------
resource "google_container_node_pool" "workload" {
  project  = var.project_id
  name     = "workload-pool"
  location = var.region
  cluster  = google_container_cluster.main.name

  autoscaling {
    min_node_count = var.workload_pool_min
    max_node_count = var.workload_pool_max
  }

  node_config {
    machine_type = var.workload_machine_type
    disk_size_gb = 100
    disk_type    = "pd-ssd"

    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = {
      pool        = "workload"
      environment = var.environment
    }

    tags = ["gke-node", "workload-pool"]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}

# -----------------------------------------------------------------------------
# Node Pool 3: Compute
# Purpose: heavy processing (pipelines, contradiction detection, reports)
# Scales to zero when idle.
# -----------------------------------------------------------------------------
resource "google_container_node_pool" "compute" {
  project  = var.project_id
  name     = "compute-pool"
  location = var.region
  cluster  = google_container_cluster.main.name

  autoscaling {
    min_node_count = 0
    max_node_count = var.compute_pool_max
  }

  node_config {
    machine_type = var.compute_machine_type
    disk_size_gb = 200
    disk_type    = "pd-ssd"

    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    labels = {
      pool        = "compute"
      environment = var.environment
    }

    taint {
      key    = "dedicated"
      value  = "compute"
      effect = "NO_SCHEDULE"
    }

    tags = ["gke-node", "compute-pool"]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }
}
