# =============================================================================
# Clario 360 — Kubernetes Addons
# Helm releases: ingress-nginx, cert-manager, metrics-server, ArgoCD,
# external-secrets
# =============================================================================

# -----------------------------------------------------------------------------
# 1. Ingress NGINX
# -----------------------------------------------------------------------------
resource "helm_release" "ingress_nginx" {
  name             = "ingress-nginx"
  repository       = "https://kubernetes.github.io/ingress-nginx"
  chart            = "ingress-nginx"
  version          = "4.9.0"
  namespace        = "ingress-nginx"
  create_namespace = true

  values = [templatefile("${path.module}/helm-values/ingress-nginx.yaml", {
    replica_count = var.environment == "production" ? 3 : 1
    min_available = var.environment == "production" ? 2 : 1
    environment   = var.environment
  })]

  depends_on = [
    google_container_node_pool.system,
  ]
}

# -----------------------------------------------------------------------------
# 2. cert-manager
# -----------------------------------------------------------------------------
resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = "1.14.0"
  namespace        = "cert-manager"
  create_namespace = true

  set {
    name  = "installCRDs"
    value = "true"
  }

  set {
    name  = "nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "tolerations[0].value"
    value = "system"
  }

  set {
    name  = "tolerations[0].effect"
    value = "NoSchedule"
  }

  depends_on = [
    google_container_node_pool.system,
  ]
}

# ClusterIssuer for Let's Encrypt
resource "kubernetes_manifest" "letsencrypt_issuer" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "ClusterIssuer"
    metadata = {
      name = "letsencrypt"
    }
    spec = {
      acme = {
        email  = var.letsencrypt_email
        server = var.environment == "production" ? "https://acme-v02.api.letsencrypt.org/directory" : "https://acme-staging-v02.api.letsencrypt.org/directory"
        privateKeySecretRef = {
          name = "letsencrypt-account-key"
        }
        solvers = [{
          http01 = {
            ingress = {
              class = "nginx"
            }
          }
        }]
      }
    }
  }

  depends_on = [helm_release.cert_manager]
}

# -----------------------------------------------------------------------------
# 3. Metrics Server (required for HPA)
# -----------------------------------------------------------------------------
resource "helm_release" "metrics_server" {
  name       = "metrics-server"
  repository = "https://kubernetes-sigs.github.io/metrics-server/"
  chart      = "metrics-server"
  version    = "3.12.0"
  namespace  = "kube-system"

  depends_on = [
    google_container_node_pool.system,
  ]
}

# -----------------------------------------------------------------------------
# 4. ArgoCD (GitOps)
# -----------------------------------------------------------------------------
resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "6.0.0"
  namespace        = "argocd"
  create_namespace = true

  values = [templatefile("${path.module}/helm-values/argocd.yaml", {
    domain       = "argocd.${var.domain}"
    repo_url     = var.gitops_repo_url
    environment  = var.environment
  })]

  depends_on = [
    google_container_node_pool.system,
  ]
}

# -----------------------------------------------------------------------------
# 5. External Secrets Operator (for Vault integration)
# -----------------------------------------------------------------------------
resource "helm_release" "external_secrets" {
  name             = "external-secrets"
  repository       = "https://charts.external-secrets.io"
  chart            = "external-secrets"
  version          = "0.9.0"
  namespace        = "external-secrets"
  create_namespace = true

  set {
    name  = "nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "tolerations[0].value"
    value = "system"
  }

  set {
    name  = "tolerations[0].effect"
    value = "NoSchedule"
  }

  depends_on = [
    google_container_node_pool.system,
  ]
}
