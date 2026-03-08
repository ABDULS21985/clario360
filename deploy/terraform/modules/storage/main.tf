# =============================================================================
# Clario 360 — Storage Module
# GCS buckets for platform assets + MinIO for S3-compatible object storage
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"

  buckets = {
    documents = {
      storage_class = "STANDARD"
      lifecycle_age = var.environment == "production" ? 0 : 365
      description   = "User-uploaded documents, board minutes, contracts"
    }
    reports = {
      storage_class = "STANDARD"
      lifecycle_age = var.environment == "production" ? 0 : 365
      description   = "Generated reports, analytics exports"
    }
    temp = {
      storage_class = "STANDARD"
      lifecycle_age = 7
      description   = "Temporary files, upload staging area"
    }
    audit-exports = {
      storage_class = "NEARLINE"
      lifecycle_age = var.environment == "production" ? 2555 : 365
      description   = "Audit log exports for compliance (7-year retention in prod)"
    }
    malware-quarantine = {
      storage_class = "COLDLINE"
      lifecycle_age = 90
      description   = "Quarantined files detected as malware"
    }
  }
}

# GCS buckets
resource "google_storage_bucket" "buckets" {
  for_each = local.buckets

  project       = var.project_id
  name          = "${local.name_prefix}-${each.key}"
  location      = var.region
  storage_class = each.value.storage_class
  force_destroy = var.environment != "production"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  dynamic "lifecycle_rule" {
    for_each = each.value.lifecycle_age > 0 ? [1] : []
    content {
      condition {
        age = each.value.lifecycle_age
      }
      action {
        type = "Delete"
      }
    }
  }

  # Delete old versions after 30 days
  lifecycle_rule {
    condition {
      num_newer_versions = 3
      with_state         = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }

  labels = merge(var.labels, {
    environment = var.environment
    component   = "storage"
    bucket-type = each.key
    managed-by  = "terraform"
  })
}

# Service account for file-service to access storage
resource "google_service_account" "storage_access" {
  project      = var.project_id
  account_id   = "${local.name_prefix}-storage"
  display_name = "Clario 360 ${var.environment} Storage Access"
}

resource "google_storage_bucket_iam_member" "storage_access" {
  for_each = google_storage_bucket.buckets

  bucket = each.value.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.storage_access.email}"
}

# -----------------------------------------------------------------------------
# MinIO Deployment (S3-compatible API for the application layer)
# -----------------------------------------------------------------------------
resource "random_password" "minio_secret_key" {
  length  = 40
  special = false
}

resource "helm_release" "minio" {
  name             = "minio"
  repository       = "https://charts.min.io/"
  chart            = "minio"
  version          = "5.1.0"
  namespace        = var.namespace
  create_namespace = false

  set {
    name  = "mode"
    value = var.environment == "production" ? "distributed" : "standalone"
  }

  set {
    name  = "replicas"
    value = var.environment == "production" ? "4" : "1"
  }

  set_sensitive {
    name  = "rootUser"
    value = "clario360-admin"
  }

  set_sensitive {
    name  = "rootPassword"
    value = random_password.minio_secret_key.result
  }

  set {
    name  = "persistence.size"
    value = var.minio_storage_size
  }

  set {
    name  = "persistence.storageClass"
    value = "premium-rwo"
  }

  set {
    name  = "resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "resources.requests.cpu"
    value = "250m"
  }

  set {
    name  = "resources.limits.memory"
    value = "1Gi"
  }

  set {
    name  = "metrics.serviceMonitor.enabled"
    value = "true"
  }

  # Create default buckets
  set {
    name  = "buckets[0].name"
    value = "documents"
  }
  set {
    name  = "buckets[0].policy"
    value = "none"
  }
  set {
    name  = "buckets[1].name"
    value = "reports"
  }
  set {
    name  = "buckets[1].policy"
    value = "none"
  }
  set {
    name  = "buckets[2].name"
    value = "temp"
  }
  set {
    name  = "buckets[2].policy"
    value = "none"
  }
}
