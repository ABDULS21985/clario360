# =============================================================================
# Clario 360 — Monitoring Module
# kube-prometheus-stack (Prometheus + Grafana + AlertManager)
# =============================================================================

resource "random_password" "grafana_admin" {
  count   = var.grafana_admin_password == "" ? 1 : 0
  length  = 24
  special = true
}

locals {
  grafana_password = var.grafana_admin_password != "" ? var.grafana_admin_password : random_password.grafana_admin[0].result
}

# -----------------------------------------------------------------------------
# kube-prometheus-stack
# Includes: Prometheus, Grafana, AlertManager, node-exporter, kube-state-metrics
# -----------------------------------------------------------------------------
resource "helm_release" "prometheus_stack" {
  name             = "kube-prometheus-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  version          = "56.0.0"
  namespace        = var.namespace
  create_namespace = true

  # Prometheus
  set {
    name  = "prometheus.prometheusSpec.retention"
    value = "${var.retention_days}d"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.storage_size
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName"
    value = "premium-rwo"
  }

  set {
    name  = "prometheus.prometheusSpec.nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "prometheus.prometheusSpec.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "prometheus.prometheusSpec.tolerations[0].value"
    value = "system"
  }

  set {
    name  = "prometheus.prometheusSpec.tolerations[0].effect"
    value = "NoSchedule"
  }

  # Grafana
  set_sensitive {
    name  = "grafana.adminPassword"
    value = local.grafana_password
  }

  set {
    name  = "grafana.ingress.enabled"
    value = "true"
  }

  set {
    name  = "grafana.ingress.ingressClassName"
    value = "nginx"
  }

  set {
    name  = "grafana.ingress.hosts[0]"
    value = "grafana.${var.domain}"
  }

  set {
    name  = "grafana.ingress.tls[0].secretName"
    value = "grafana-tls"
  }

  set {
    name  = "grafana.ingress.tls[0].hosts[0]"
    value = "grafana.${var.domain}"
  }

  set {
    name  = "grafana.ingress.annotations.cert-manager\\.io/cluster-issuer"
    value = "letsencrypt"
  }

  set {
    name  = "grafana.nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "grafana.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "grafana.tolerations[0].value"
    value = "system"
  }

  set {
    name  = "grafana.tolerations[0].effect"
    value = "NoSchedule"
  }

  set {
    name  = "grafana.persistence.enabled"
    value = "true"
  }

  set {
    name  = "grafana.persistence.size"
    value = "10Gi"
  }

  # AlertManager
  set {
    name  = "alertmanager.alertmanagerSpec.nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.tolerations[0].value"
    value = "system"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.tolerations[0].effect"
    value = "NoSchedule"
  }

  # ServiceMonitor for Clario 360 services
  set {
    name  = "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues"
    value = "false"
  }
}
