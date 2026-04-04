# =============================================================================
# Clario 360 — Kafka Module
# Strimzi operator + Kafka cluster (KRaft mode, no ZooKeeper)
# =============================================================================

# -----------------------------------------------------------------------------
# Strimzi Operator (via Helm)
# -----------------------------------------------------------------------------
resource "helm_release" "strimzi" {
  name             = "strimzi-kafka-operator"
  repository       = "https://strimzi.io/charts/"
  chart            = "strimzi-kafka-operator"
  version          = "0.40.0"
  namespace        = var.namespace
  create_namespace = true

  set {
    name  = "watchAnyNamespace"
    value = "false"
  }
}

# -----------------------------------------------------------------------------
# Kafka Cluster (Strimzi CRD — KRaft mode)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "kafka_cluster" {
  manifest = {
    apiVersion = "kafka.strimzi.io/v1beta2"
    kind       = "Kafka"
    metadata = {
      name      = "clario360"
      namespace = var.namespace
      labels = merge(var.labels, {
        environment = var.environment
        managed-by  = "terraform"
      })
    }
    spec = {
      kafka = {
        version  = "3.7.0"
        replicas = var.kafka_replicas

        listeners = concat(
          [{
            name = "plain"
            port = 9092
            type = "internal"
            tls  = false
          }],
          var.environment != "dev" ? [{
            name = "tls"
            port = 9093
            type = "internal"
            tls  = true
            authentication = {
              type = "scram-sha-512"
            }
          }] : []
        )

        config = {
          "offsets.topic.replication.factor"          = var.kafka_replicas
          "transaction.state.log.replication.factor"  = var.kafka_replicas
          "transaction.state.log.min.isr"             = max(1, var.kafka_replicas - 1)
          "default.replication.factor"                = var.kafka_replicas
          "min.insync.replicas"                       = max(1, var.kafka_replicas - 1)
          "log.retention.hours"                       = 168
          "log.segment.bytes"                         = 1073741824
          "auto.create.topics.enable"                 = false
          "num.partitions"                            = 6
          "message.max.bytes"                         = 10485760
        }

        storage = {
          type  = "persistent-claim"
          size  = var.kafka_storage_size
          class = "premium-rwo"
        }

        resources = {
          requests = {
            memory = var.kafka_memory_request
            cpu    = var.kafka_cpu_request
          }
          limits = {
            memory = var.kafka_memory_limit
            cpu    = var.kafka_cpu_limit
          }
        }

        template = {
          pod = {
            affinity = var.kafka_replicas > 1 ? {
              podAntiAffinity = {
                requiredDuringSchedulingIgnoredDuringExecution = [{
                  topologyKey = "kubernetes.io/hostname"
                  labelSelector = {
                    matchLabels = {
                      "strimzi.io/cluster" = "clario360"
                      "strimzi.io/kind"    = "Kafka"
                    }
                  }
                }]
              }
            } : null
          }
        }
      }

      entityOperator = {
        topicOperator = {}
        userOperator  = {}
      }
    }
  }

  depends_on = [helm_release.strimzi]
}
