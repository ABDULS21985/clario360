# =============================================================================
# Loki Stack — Log Aggregation
# Loki for log storage + Promtail for log collection
# =============================================================================

resource "helm_release" "loki" {
  name             = "loki"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "loki-stack"
  version          = "2.10.0"
  namespace        = var.namespace
  create_namespace = false

  set {
    name  = "loki.persistence.enabled"
    value = "true"
  }

  set {
    name  = "loki.persistence.size"
    value = var.loki_storage_size
  }

  set {
    name  = "loki.persistence.storageClassName"
    value = "premium-rwo"
  }

  set {
    name  = "loki.config.limits_config.retention_period"
    value = var.environment == "production" ? "720h" : "168h"
  }

  set {
    name  = "loki.nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "loki.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "loki.tolerations[0].value"
    value = "system"
  }

  set {
    name  = "loki.tolerations[0].effect"
    value = "NoSchedule"
  }

  # Promtail — runs on all nodes to collect logs
  set {
    name  = "promtail.enabled"
    value = "true"
  }

  set {
    name  = "promtail.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "promtail.tolerations[0].operator"
    value = "Exists"
  }

  set {
    name  = "promtail.tolerations[0].effect"
    value = "NoSchedule"
  }

  # Grafana datasource (auto-configured via kube-prometheus-stack sidecar)
  set {
    name  = "grafana.enabled"
    value = "false"
  }

  depends_on = [helm_release.prometheus_stack]
}
