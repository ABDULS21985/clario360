# =============================================================================
# 22 Kafka Topics + Dead Letter Queues
# Topics are organized by domain with per-topic partition counts and retention
# =============================================================================

locals {
  kafka_topics = {
    # Platform topics
    "platform.iam.events"           = { partitions = 6, retention_ms = 604800000 }   # 7 days
    "platform.audit.events"         = { partitions = 12, retention_ms = 2592000000 }  # 30 days
    "platform.notification.events"  = { partitions = 6, retention_ms = 259200000 }    # 3 days
    "platform.file.events"          = { partitions = 3, retention_ms = 604800000 }    # 7 days
    "platform.workflow.events"      = { partitions = 6, retention_ms = 604800000 }    # 7 days
    "platform.onboarding.events"    = { partitions = 3, retention_ms = 604800000 }    # 7 days
    "platform.session.invalidation" = { partitions = 6, retention_ms = 86400000 }     # 1 day

    # Cybersecurity topics
    "cyber.alert.events" = { partitions = 12, retention_ms = 2592000000 } # 30 days
    "cyber.risk.events"  = { partitions = 6, retention_ms = 604800000 }   # 7 days
    "cyber.ctem.events"  = { partitions = 3, retention_ms = 604800000 }   # 7 days
    "cyber.asset.events" = { partitions = 6, retention_ms = 604800000 }   # 7 days

    # Data topics
    "data.source.events"        = { partitions = 6, retention_ms = 604800000 } # 7 days
    "data.pipeline.events"      = { partitions = 6, retention_ms = 604800000 } # 7 days
    "data.quality.events"       = { partitions = 6, retention_ms = 604800000 } # 7 days
    "data.contradiction.events" = { partitions = 3, retention_ms = 604800000 } # 7 days
    "data.lineage.events"       = { partitions = 3, retention_ms = 604800000 } # 7 days
    "data.model.events"         = { partitions = 3, retention_ms = 604800000 } # 7 days
    "data.darkdata.events"      = { partitions = 3, retention_ms = 604800000 } # 7 days

    # Enterprise topics
    "enterprise.acta.events"  = { partitions = 6, retention_ms = 2592000000 } # 30 days (governance)
    "enterprise.lex.events"   = { partitions = 6, retention_ms = 2592000000 } # 30 days (legal)
    "enterprise.visus.events" = { partitions = 6, retention_ms = 604800000 }  # 7 days
  }

  # Production doubles partition counts
  partition_multiplier = var.environment == "production" ? 2 : 1
}

# Main topics
resource "kubernetes_manifest" "kafka_topics" {
  for_each = local.kafka_topics

  manifest = {
    apiVersion = "kafka.strimzi.io/v1beta2"
    kind       = "KafkaTopic"
    metadata = {
      name      = each.key
      namespace = var.namespace
      labels = {
        "strimzi.io/cluster" = "clario360"
        environment          = var.environment
        managed-by           = "terraform"
      }
    }
    spec = {
      partitions = each.value.partitions * local.partition_multiplier
      replicas   = var.kafka_replicas
      config = {
        "retention.ms"       = tostring(each.value.retention_ms)
        "cleanup.policy"     = "delete"
        "min.insync.replicas" = tostring(max(1, var.kafka_replicas - 1))
      }
    }
  }

  depends_on = [kubernetes_manifest.kafka_cluster]
}

# Dead Letter Queue topics (3 partitions each, 30-day retention)
resource "kubernetes_manifest" "kafka_dlq_topics" {
  for_each = local.kafka_topics

  manifest = {
    apiVersion = "kafka.strimzi.io/v1beta2"
    kind       = "KafkaTopic"
    metadata = {
      name      = "${each.key}.dlq"
      namespace = var.namespace
      labels = {
        "strimzi.io/cluster" = "clario360"
        environment          = var.environment
        managed-by           = "terraform"
        purpose              = "dead-letter-queue"
      }
    }
    spec = {
      partitions = 3
      replicas   = var.kafka_replicas
      config = {
        "retention.ms"        = "2592000000"
        "cleanup.policy"      = "delete"
        "min.insync.replicas" = tostring(max(1, var.kafka_replicas - 1))
      }
    }
  }

  depends_on = [kubernetes_manifest.kafka_cluster]
}
