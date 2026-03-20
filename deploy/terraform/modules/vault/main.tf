# =============================================================================
# Clario 360 — Vault Module
# HashiCorp Vault for secrets management, dynamic credentials, PKI
# Auto-unseal via Cloud KMS
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
}

# KMS key for Vault auto-unseal
resource "google_kms_crypto_key" "vault_unseal" {
  name     = "${local.name_prefix}-vault-unseal"
  key_ring = var.kms_key_ring_id
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s" # 90 days

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = var.environment == "production" ? "HSM" : "SOFTWARE"
  }
}

# Service account for Vault KMS auto-unseal
resource "google_service_account" "vault" {
  project      = var.project_id
  account_id   = "${local.name_prefix}-vault"
  display_name = "Clario 360 ${var.environment} Vault"
}

resource "google_kms_crypto_key_iam_member" "vault_unseal" {
  crypto_key_id = google_kms_crypto_key.vault_unseal.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.vault.email}"
}

# Vault Helm deployment
resource "helm_release" "vault" {
  name             = "vault"
  repository       = "https://helm.releases.hashicorp.com"
  chart            = "vault"
  version          = "0.27.0"
  namespace        = var.namespace
  create_namespace = true

  # Server configuration
  set {
    name  = "server.ha.enabled"
    value = var.environment == "production" ? "true" : "false"
  }

  set {
    name  = "server.ha.replicas"
    value = var.environment == "production" ? "3" : "1"
  }

  set {
    name  = "server.dataStorage.size"
    value = var.storage_size
  }

  set {
    name  = "server.dataStorage.storageClass"
    value = "premium-rwo"
  }

  set {
    name  = "server.nodeSelector.pool"
    value = "system"
  }

  set {
    name  = "server.tolerations[0].key"
    value = "dedicated"
  }

  set {
    name  = "server.tolerations[0].value"
    value = "system"
  }

  set {
    name  = "server.tolerations[0].effect"
    value = "NoSchedule"
  }

  set {
    name  = "server.serviceAccount.annotations.iam\\.gke\\.io/gcp-service-account"
    value = google_service_account.vault.email
  }

  # GCP KMS auto-unseal configuration
  values = [yamlencode({
    server = {
      extraEnvironmentVars = {
        GOOGLE_PROJECT = var.project_id
        GOOGLE_REGION  = var.region
      }
      ha = {
        config = <<-EOT
          ui = true

          listener "tcp" {
            tls_disable = 1
            address = "[::]:8200"
            cluster_address = "[::]:8201"
          }

          storage "raft" {
            path = "/vault/data"
          }

          seal "gcpckms" {
            project     = "${var.project_id}"
            region      = "${var.region}"
            key_ring    = "${basename(var.kms_key_ring_id)}"
            crypto_key  = "${google_kms_crypto_key.vault_unseal.name}"
          }

          service_registration "kubernetes" {}

          telemetry {
            prometheus_retention_time = "30s"
            disable_hostname = true
          }
        EOT
      }
    }
    ui = {
      enabled = true
    }
    injector = {
      enabled = true
    }
  })]
}
