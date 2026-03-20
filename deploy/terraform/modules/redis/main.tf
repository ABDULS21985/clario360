# =============================================================================
# Clario 360 — Redis Module
# Google Cloud Memorystore for Redis — session caching, rate limiting,
# real-time pub/sub, and distributed locking.
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
}

resource "google_redis_instance" "main" {
  project        = var.project_id
  name           = "${local.name_prefix}-redis"
  region         = var.region
  display_name   = "Clario 360 ${var.environment} Redis"
  memory_size_gb = var.memory_size_gb
  redis_version  = var.redis_version

  tier = var.environment == "production" ? "STANDARD_HA" : "BASIC"

  authorized_network = var.vpc_id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"

  transit_encryption_mode = var.environment == "production" ? "SERVER_AUTHENTICATION" : "DISABLED"

  redis_configs = {
    maxmemory-policy  = "allkeys-lru"
    notify-keyspace-events = "Ex"
    activedefrag      = "yes"
  }

  maintenance_policy {
    weekly_maintenance_window {
      day = "SATURDAY"
      start_time {
        hours   = 3
        minutes = 0
      }
    }
  }

  replica_count          = var.replica_count
  read_replicas_mode     = var.replica_count > 0 ? "READ_REPLICAS_ENABLED" : "READ_REPLICAS_DISABLED"

  labels = merge(var.labels, {
    environment = var.environment
    component   = "redis"
    managed-by  = "terraform"
  })
}
