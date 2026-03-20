# =============================================================================
# Binary Authorization — Enforce signed container images in production
# =============================================================================

resource "google_binary_authorization_policy" "policy" {
  count   = var.environment == "production" ? 1 : 0
  project = var.project_id

  global_policy_evaluation_mode = "ENABLE"

  default_admission_rule {
    evaluation_mode  = "REQUIRE_ATTESTATION"
    enforcement_mode = "ENFORCED_BLOCK_AND_AUDIT_LOG"
    require_attestations_by = [
      google_binary_authorization_attestor.clario360[0].name,
    ]
  }

  # Allow system namespaces
  admission_whitelist_patterns {
    name_pattern = "gcr.io/google-containers/*"
  }

  admission_whitelist_patterns {
    name_pattern = "registry.k8s.io/*"
  }

  admission_whitelist_patterns {
    name_pattern = "docker.io/istio/*"
  }

  # Allow Clario 360 registry
  dynamic "admission_whitelist_patterns" {
    for_each = var.container_registry != "" ? [1] : []
    content {
      name_pattern = "${var.container_registry}/*"
    }
  }
}

# Attestor for Clario 360 CI/CD pipeline
resource "google_binary_authorization_attestor" "clario360" {
  count   = var.environment == "production" ? 1 : 0
  project = var.project_id
  name    = "clario360-ci-attestor"

  attestation_authority_note {
    note_reference = google_container_analysis_note.attestor[0].name

    public_keys {
      id = "clario360-ci-key"
      pkix_public_key {
        public_key_pem      = var.environment == "production" ? file("${path.module}/attestor-key.pem") : ""
        signature_algorithm = "ECDSA_P256_SHA256"
      }
    }
  }
}

resource "google_container_analysis_note" "attestor" {
  count   = var.environment == "production" ? 1 : 0
  project = var.project_id
  name    = "clario360-ci-attestor-note"

  attestation_authority {
    hint {
      human_readable_name = "Clario 360 CI/CD Pipeline Attestation"
    }
  }
}
