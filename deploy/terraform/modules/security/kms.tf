# =============================================================================
# Cloud KMS — Envelope Encryption
# KMS keys for GKE secrets encryption, database encryption, and application-level
# encryption of sensitive data (PII, credentials, API keys)
# =============================================================================

resource "google_kms_key_ring" "main" {
  project  = var.project_id
  name     = "${local.name_prefix}-keyring"
  location = var.region
}

# Key for GKE application-layer secrets encryption
resource "google_kms_crypto_key" "gke_secrets" {
  name     = "${local.name_prefix}-gke-secrets"
  key_ring = google_kms_key_ring.main.id
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s" # 90 days

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = var.environment == "production" ? "HSM" : "SOFTWARE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Key for application-level data encryption (PII, credentials)
resource "google_kms_crypto_key" "data_encryption" {
  name     = "${local.name_prefix}-data-encryption"
  key_ring = google_kms_key_ring.main.id
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s" # 90 days

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = var.environment == "production" ? "HSM" : "SOFTWARE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Key for database column encryption
resource "google_kms_crypto_key" "database_encryption" {
  name     = "${local.name_prefix}-database-encryption"
  key_ring = google_kms_key_ring.main.id
  purpose  = "ENCRYPT_DECRYPT"

  rotation_period = "7776000s" # 90 days

  version_template {
    algorithm        = "GOOGLE_SYMMETRIC_ENCRYPTION"
    protection_level = var.environment == "production" ? "HSM" : "SOFTWARE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# IAM: Allow GKE service account to use GKE secrets key
resource "google_kms_crypto_key_iam_member" "gke_encrypt" {
  crypto_key_id = google_kms_crypto_key.gke_secrets.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:service-${data.google_project.current.number}@container-engine-robot.iam.gserviceaccount.com"
}

data "google_project" "current" {
  project_id = var.project_id
}

# IAM: Allow application service accounts to use data encryption key
resource "google_kms_crypto_key_iam_member" "service_encrypt" {
  for_each = local.service_accounts

  crypto_key_id = google_kms_crypto_key.data_encryption.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.services[each.key].email}"
}
