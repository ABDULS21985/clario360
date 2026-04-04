# =============================================================================
# Per-Service Kafka Users with ACLs
# Each service gets a KafkaUser CRD with topic-level access control
# Only created in staging/production where SASL auth is enabled
# =============================================================================

locals {
  # Service → topic ACL mapping based on the event flow matrix
  kafka_user_acls = var.environment != "dev" ? {
    "iam-service" = {
      produce = ["platform.iam.events", "platform.session.invalidation"]
      consume = ["platform.audit.events"]
    }
    "audit-service" = {
      produce = ["platform.audit.events"]
      consume = [
        "platform.iam.events", "platform.notification.events",
        "platform.file.events", "platform.workflow.events",
        "platform.onboarding.events", "cyber.alert.events",
        "enterprise.acta.events", "enterprise.lex.events",
      ]
    }
    "workflow-engine" = {
      produce = ["platform.workflow.events", "platform.notification.events"]
      consume = ["platform.iam.events", "platform.file.events"]
    }
    "notification-service" = {
      produce = []
      consume = ["platform.notification.events"]
    }
    "file-service" = {
      produce = ["platform.file.events"]
      consume = ["platform.iam.events"]
    }
    "onboarding-service" = {
      produce = ["platform.onboarding.events", "platform.iam.events"]
      consume = []
    }
    "cyber-service" = {
      produce = ["cyber.alert.events", "cyber.risk.events", "cyber.ctem.events", "cyber.asset.events"]
      consume = ["platform.iam.events", "platform.file.events", "data.source.events", "data.pipeline.events"]
    }
    "data-service" = {
      produce = [
        "data.source.events", "data.pipeline.events", "data.quality.events",
        "data.contradiction.events", "data.lineage.events", "data.model.events", "data.darkdata.events",
      ]
      consume = ["platform.iam.events", "platform.file.events"]
    }
    "acta-service" = {
      produce = ["enterprise.acta.events"]
      consume = ["platform.iam.events", "platform.workflow.events", "platform.file.events"]
    }
    "lex-service" = {
      produce = ["enterprise.lex.events"]
      consume = ["platform.iam.events", "platform.workflow.events", "platform.file.events"]
    }
    "visus-service" = {
      produce = ["enterprise.visus.events"]
      consume = [
        "platform.iam.events", "cyber.alert.events", "cyber.risk.events",
        "data.source.events", "data.quality.events",
        "enterprise.acta.events", "enterprise.lex.events",
      ]
    }
  } : {}
}

resource "kubernetes_manifest" "kafka_users" {
  for_each = local.kafka_user_acls

  manifest = {
    apiVersion = "kafka.strimzi.io/v1beta2"
    kind       = "KafkaUser"
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
      authentication = {
        type = "scram-sha-512"
      }
      authorization = {
        type = "simple"
        acls = concat(
          # Produce ACLs
          [for topic in each.value.produce : {
            resource = {
              type        = "topic"
              name        = topic
              patternType = "literal"
            }
            operations = ["Write", "Describe"]
            host       = "*"
          }],
          # Consume ACLs
          [for topic in each.value.consume : {
            resource = {
              type        = "topic"
              name        = topic
              patternType = "literal"
            }
            operations = ["Read", "Describe"]
            host       = "*"
          }],
          # Consumer group ACL
          [{
            resource = {
              type        = "group"
              name        = each.key
              patternType = "prefix"
            }
            operations = ["Read"]
            host       = "*"
          }],
          # DLQ produce ACLs (service can write to its own DLQs)
          [for topic in each.value.consume : {
            resource = {
              type        = "topic"
              name        = "${topic}.dlq"
              patternType = "literal"
            }
            operations = ["Write", "Describe"]
            host       = "*"
          }]
        )
      }
    }
  }

  depends_on = [kubernetes_manifest.kafka_cluster]
}
