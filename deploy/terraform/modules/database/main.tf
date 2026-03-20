# =============================================================================
# Clario 360 — Database Module
# Cloud SQL PostgreSQL 16 with private networking, automated backups, PITR,
# Query Insights, and performance-tuned flags.
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
}

# -----------------------------------------------------------------------------
# Cloud SQL Instance
# -----------------------------------------------------------------------------
resource "google_sql_database_instance" "main" {
  project             = var.project_id
  name                = "${local.name_prefix}-pg"
  database_version    = "POSTGRES_16"
  region              = var.region
  deletion_protection = var.environment == "production"

  depends_on = [var.psa_connection_id]

  settings {
    tier              = var.db_tier
    edition           = "ENTERPRISE"
    availability_type = var.environment == "production" ? "REGIONAL" : "ZONAL"
    disk_type         = "PD_SSD"
    disk_size         = var.db_disk_size_gb
    disk_autoresize   = true
    disk_autoresize_limit = var.db_disk_max_gb

    ip_configuration {
      ipv4_enabled    = false
      private_network = var.vpc_self_link
      require_ssl     = var.environment != "dev"
      ssl_mode        = "ENCRYPTED_ONLY"
    }

    backup_configuration {
      enabled                        = true
      start_time                     = "02:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = var.environment == "production" ? 7 : 3

      backup_retention_settings {
        retained_backups = var.environment == "production" ? 30 : 7
        retention_unit   = "COUNT"
      }
    }

    maintenance_window {
      day          = 6
      hour         = 3
      update_track = "stable"
    }

    insights_config {
      query_insights_enabled  = true
      query_plans_per_minute  = 5
      query_string_length     = 4096
      record_application_tags = true
      record_client_address   = true
    }

    database_flags {
      name  = "max_connections"
      value = var.db_max_connections
    }
    database_flags {
      name  = "password_encryption"
      value = "scram-sha-256"
    }
    database_flags {
      name  = "log_min_duration_statement"
      value = var.environment == "production" ? "1000" : "500"
    }
    database_flags {
      name  = "log_checkpoints"
      value = "on"
    }
    database_flags {
      name  = "log_connections"
      value = "on"
    }
    database_flags {
      name  = "log_disconnections"
      value = "on"
    }
    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }
    database_flags {
      name  = "log_temp_files"
      value = "0"
    }
    database_flags {
      name  = "idle_in_transaction_session_timeout"
      value = "300000"
    }
    database_flags {
      name  = "statement_timeout"
      value = "60000"
    }
    database_flags {
      name  = "autovacuum_max_workers"
      value = "4"
    }
    database_flags {
      name  = "autovacuum_naptime"
      value = "30"
    }
    database_flags {
      name  = "autovacuum_vacuum_threshold"
      value = "50"
    }
    database_flags {
      name  = "autovacuum_analyze_threshold"
      value = "50"
    }
    database_flags {
      name  = "autovacuum_vacuum_scale_factor"
      value = "0.1"
    }
    database_flags {
      name  = "autovacuum_analyze_scale_factor"
      value = "0.05"
    }
    database_flags {
      name  = "random_page_cost"
      value = "1.1"
    }
    database_flags {
      name  = "effective_io_concurrency"
      value = "200"
    }
    database_flags {
      name  = "default_statistics_target"
      value = "100"
    }

    user_labels = merge(var.labels, {
      environment = var.environment
      component   = "database"
      managed-by  = "terraform"
    })
  }
}
