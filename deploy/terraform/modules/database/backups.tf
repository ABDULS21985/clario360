# =============================================================================
# Long-Term Backup Configuration
# GCS bucket for exports beyond Cloud SQL's built-in retention
# Weekly SQL exports via Cloud Scheduler
# =============================================================================

# GCS bucket for database backup exports
resource "google_storage_bucket" "db_backups" {
  project       = var.project_id
  name          = "${local.name_prefix}-db-backups"
  location      = var.region
  storage_class = "NEARLINE"
  force_destroy = var.environment != "production"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = var.environment == "production" ? 365 : 90
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 3
    }
    action {
      type = "Delete"
    }
  }

  labels = merge(var.labels, {
    environment = var.environment
    component   = "database-backups"
    managed-by  = "terraform"
  })
}

# Service account for Cloud SQL export
resource "google_service_account" "db_export" {
  project      = var.project_id
  account_id   = "${local.name_prefix}-db-export"
  display_name = "Clario 360 ${var.environment} DB Export"
}

resource "google_project_iam_member" "db_export_sql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.db_export.email}"
}

resource "google_storage_bucket_iam_member" "db_export_writer" {
  bucket = google_storage_bucket.db_backups.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.db_export.email}"
}

# Weekly export scheduler — Sunday 04:00 UTC
resource "google_cloud_scheduler_job" "db_export" {
  project     = var.project_id
  name        = "${local.name_prefix}-db-export"
  region      = var.region
  schedule    = "0 4 * * 0"
  time_zone   = "UTC"
  description = "Weekly database export to GCS for long-term retention"

  http_target {
    uri         = "https://sqladmin.googleapis.com/v1/projects/${var.project_id}/instances/${google_sql_database_instance.main.name}/export"
    http_method = "POST"

    body = base64encode(jsonencode({
      exportContext = {
        fileType  = "SQL"
        uri       = "gs://${google_storage_bucket.db_backups.name}/exports/weekly/"
        databases = [for db in local.databases : db]
      }
    }))

    oauth_token {
      service_account_email = google_service_account.db_export.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "30s"
    max_backoff_duration = "300s"
  }
}
