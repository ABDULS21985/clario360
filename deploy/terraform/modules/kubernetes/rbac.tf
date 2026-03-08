# =============================================================================
# Clario 360 — Kubernetes RBAC
# Namespaces and ClusterRoles for service accounts via Workload Identity
# =============================================================================

# -----------------------------------------------------------------------------
# Namespaces
# -----------------------------------------------------------------------------
resource "kubernetes_namespace" "clario360" {
  metadata {
    name = "clario360"
    labels = {
      "app.kubernetes.io/part-of" = "clario360"
      environment                 = var.environment
    }
  }

  depends_on = [google_container_cluster.main]
}

resource "kubernetes_namespace" "clario360_jobs" {
  metadata {
    name = "clario360-jobs"
    labels = {
      "app.kubernetes.io/part-of" = "clario360"
      environment                 = var.environment
      purpose                     = "batch-processing"
    }
  }

  depends_on = [google_container_cluster.main]
}

resource "kubernetes_namespace" "monitoring" {
  metadata {
    name = "monitoring"
    labels = {
      "app.kubernetes.io/part-of" = "clario360"
      environment                 = var.environment
    }
  }

  depends_on = [google_container_cluster.main]
}

resource "kubernetes_namespace" "vault" {
  metadata {
    name = "vault"
    labels = {
      "app.kubernetes.io/part-of" = "clario360"
      environment                 = var.environment
    }
  }

  depends_on = [google_container_cluster.main]
}

# -----------------------------------------------------------------------------
# ClusterRole: clario360-service
# Allows services to read pods, services, configmaps, secrets in own namespace
# -----------------------------------------------------------------------------
resource "kubernetes_cluster_role" "clario360_service" {
  metadata {
    name = "clario360-service"
    labels = {
      "app.kubernetes.io/part-of" = "clario360"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "services", "configmaps", "secrets"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = [""]
    resources  = ["events"]
    verbs      = ["get", "list"]
  }

  depends_on = [google_container_cluster.main]
}

# -----------------------------------------------------------------------------
# Service accounts + RoleBindings per service
# Maps GCP service accounts to Kubernetes service accounts via Workload Identity
# -----------------------------------------------------------------------------
locals {
  services = [
    "api-gateway",
    "iam-service",
    "audit-service",
    "workflow-engine",
    "notification-service",
    "file-service",
    "cyber-service",
    "data-service",
    "acta-service",
    "lex-service",
    "visus-service",
    "onboarding-service",
  ]
}

resource "kubernetes_service_account" "services" {
  for_each = toset(local.services)

  metadata {
    name      = each.value
    namespace = kubernetes_namespace.clario360.metadata[0].name

    annotations = {
      "iam.gke.io/gcp-service-account" = "clario360-${replace(each.value, "-", "")}@${var.project_id}.iam.gserviceaccount.com"
    }

    labels = {
      "app.kubernetes.io/name"    = each.value
      "app.kubernetes.io/part-of" = "clario360"
    }
  }
}

resource "kubernetes_role_binding" "service_bindings" {
  for_each = toset(local.services)

  metadata {
    name      = "${each.value}-binding"
    namespace = kubernetes_namespace.clario360.metadata[0].name
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.clario360_service.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = each.value
    namespace = kubernetes_namespace.clario360.metadata[0].name
  }
}
