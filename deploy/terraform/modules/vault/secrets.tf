# =============================================================================
# Vault Secret Engines, Policies, and Auth Methods
# =============================================================================

# Enable KV v2 secret engine for static secrets
resource "vault_mount" "kv" {
  path        = "secret"
  type        = "kv-v2"
  description = "KV v2 secret engine for Clario 360 ${var.environment}"

  options = {
    version = "2"
  }

  depends_on = [helm_release.vault]
}

# Enable database secret engine for dynamic credentials
resource "vault_mount" "database" {
  path        = "database"
  type        = "database"
  description = "Dynamic database credentials for Clario 360 ${var.environment}"

  depends_on = [helm_release.vault]
}

# Kubernetes auth method for pod authentication
resource "vault_auth_backend" "kubernetes" {
  type = "kubernetes"
  path = "kubernetes"

  depends_on = [helm_release.vault]
}

resource "vault_kubernetes_auth_backend_config" "config" {
  backend            = vault_auth_backend.kubernetes.path
  kubernetes_host    = "https://kubernetes.default.svc"
  disable_iss_validation = true

  depends_on = [helm_release.vault]
}

# -----------------------------------------------------------------------------
# Vault Policies (per-service access)
# Each service can only read its own secrets
# -----------------------------------------------------------------------------
locals {
  service_policies = {
    iam-service          = ["secret/data/clario360/${var.environment}/database/iam", "secret/data/clario360/${var.environment}/jwt", "secret/data/clario360/${var.environment}/encryption"]
    audit-service        = ["secret/data/clario360/${var.environment}/database/platform", "secret/data/clario360/${var.environment}/encryption"]
    workflow-engine      = ["secret/data/clario360/${var.environment}/database/platform"]
    notification-service = ["secret/data/clario360/${var.environment}/database/platform", "secret/data/clario360/${var.environment}/smtp", "secret/data/clario360/${var.environment}/twilio"]
    file-service         = ["secret/data/clario360/${var.environment}/database/platform", "secret/data/clario360/${var.environment}/storage"]
    cyber-service        = ["secret/data/clario360/${var.environment}/database/cyber"]
    data-service         = ["secret/data/clario360/${var.environment}/database/data"]
    acta-service         = ["secret/data/clario360/${var.environment}/database/acta"]
    lex-service          = ["secret/data/clario360/${var.environment}/database/lex"]
    visus-service        = ["secret/data/clario360/${var.environment}/database/visus"]
    onboarding-service   = ["secret/data/clario360/${var.environment}/database/platform", "secret/data/clario360/${var.environment}/database/iam"]
    api-gateway          = ["secret/data/clario360/${var.environment}/jwt"]
  }
}

resource "vault_policy" "service_policies" {
  for_each = local.service_policies

  name = "clario360-${each.key}"

  policy = join("\n\n", [
    for path in each.value : <<-EOT
      path "${path}" {
        capabilities = ["read"]
      }
      path "${path}/*" {
        capabilities = ["read"]
      }
    EOT
  ])

  depends_on = [helm_release.vault]
}

# Kubernetes auth roles — bind K8s service accounts to Vault policies
resource "vault_kubernetes_auth_backend_role" "service_roles" {
  for_each = local.service_policies

  backend                          = vault_auth_backend.kubernetes.path
  role_name                        = each.key
  bound_service_account_names      = [each.key]
  bound_service_account_namespaces = ["clario360"]
  token_policies                   = [vault_policy.service_policies[each.key].name]
  token_ttl                        = 3600
  token_max_ttl                    = 86400

  depends_on = [vault_kubernetes_auth_backend_config.config]
}

# Admin policy for operators
resource "vault_policy" "admin" {
  name = "clario360-admin"

  policy = <<-EOT
    path "secret/*" {
      capabilities = ["create", "read", "update", "delete", "list"]
    }
    path "database/*" {
      capabilities = ["create", "read", "update", "delete", "list"]
    }
    path "pki/*" {
      capabilities = ["create", "read", "update", "delete", "list"]
    }
    path "sys/*" {
      capabilities = ["create", "read", "update", "delete", "list", "sudo"]
    }
    path "auth/*" {
      capabilities = ["create", "read", "update", "delete", "list"]
    }
  EOT

  depends_on = [helm_release.vault]
}
