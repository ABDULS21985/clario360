# =============================================================================
# Vault PKI Secret Engine — Internal TLS Certificate Management
# Root CA + Intermediate CA for service-to-service mTLS
# =============================================================================

# Root CA (long-lived, used only to sign intermediate)
resource "vault_mount" "pki_root" {
  path                  = "pki"
  type                  = "pki"
  description           = "Clario 360 Root PKI"
  max_lease_ttl_seconds = 315360000 # 10 years

  depends_on = [helm_release.vault]
}

resource "vault_pki_secret_backend_root_cert" "root" {
  backend     = vault_mount.pki_root.path
  type        = "internal"
  common_name = "Clario 360 Root CA (${var.environment})"
  ttl         = "87600h" # 10 years
  key_type    = "rsa"
  key_bits    = 4096

  depends_on = [vault_mount.pki_root]
}

# Intermediate CA (medium-lived, signs service certificates)
resource "vault_mount" "pki_intermediate" {
  path                  = "pki_int"
  type                  = "pki"
  description           = "Clario 360 Intermediate PKI"
  max_lease_ttl_seconds = 157680000 # 5 years

  depends_on = [helm_release.vault]
}

resource "vault_pki_secret_backend_intermediate_cert_request" "intermediate" {
  backend     = vault_mount.pki_intermediate.path
  type        = "internal"
  common_name = "Clario 360 Intermediate CA (${var.environment})"
  key_type    = "rsa"
  key_bits    = 4096

  depends_on = [vault_mount.pki_intermediate]
}

resource "vault_pki_secret_backend_root_sign_intermediate" "intermediate" {
  backend     = vault_mount.pki_root.path
  csr         = vault_pki_secret_backend_intermediate_cert_request.intermediate.csr
  common_name = "Clario 360 Intermediate CA (${var.environment})"
  ttl         = "43800h" # 5 years

  depends_on = [vault_pki_secret_backend_root_cert.root]
}

resource "vault_pki_secret_backend_intermediate_set_signed" "intermediate" {
  backend     = vault_mount.pki_intermediate.path
  certificate = vault_pki_secret_backend_root_sign_intermediate.intermediate.certificate

  depends_on = [vault_pki_secret_backend_root_sign_intermediate.intermediate]
}

# Role for issuing service certificates
resource "vault_pki_secret_backend_role" "service_cert" {
  backend          = vault_mount.pki_intermediate.path
  name             = "clario360-service"
  max_ttl          = "720h" # 30 days
  ttl              = "72h"  # 3 days default
  allow_localhost  = true
  allowed_domains  = ["clario360.svc.cluster.local", "clario360"]
  allow_subdomains = true
  allow_bare_domains = false
  generate_lease   = true
  key_type         = "rsa"
  key_bits         = 2048
  key_usage        = ["DigitalSignature", "KeyEncipherment"]
  ext_key_usage    = ["ServerAuth", "ClientAuth"]

  depends_on = [vault_pki_secret_backend_intermediate_set_signed.intermediate]
}
