# =============================================================================
# Clario 360 — DNS Module
# Cloud DNS zone and records for the platform
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
  zone_name   = var.dns_zone_name != "" ? var.dns_zone_name : "${local.name_prefix}-zone"

  # DNS records to create when ingress IP is available
  dns_records = var.ingress_ip != "" ? {
    app = {
      name = var.domain
      type = "A"
      data = [var.ingress_ip]
    }
    api = {
      name = "api.${var.domain}"
      type = "A"
      data = [var.ingress_ip]
    }
    argocd = {
      name = "argocd.${var.domain}"
      type = "A"
      data = [var.ingress_ip]
    }
    grafana = {
      name = "grafana.${var.domain}"
      type = "A"
      data = [var.ingress_ip]
    }
  } : {}
}

# Cloud DNS zone
resource "google_dns_managed_zone" "main" {
  count = var.create_zone ? 1 : 0

  project     = var.project_id
  name        = local.zone_name
  dns_name    = "${var.domain}."
  description = "Clario 360 ${var.environment} DNS zone"
  visibility  = "public"

  dnssec_config {
    state = var.environment == "production" ? "on" : "off"
  }

  labels = merge(var.labels, {
    environment = var.environment
    component   = "dns"
    managed-by  = "terraform"
  })
}

# A records pointing to ingress load balancer
resource "google_dns_record_set" "records" {
  for_each = local.dns_records

  project      = var.project_id
  managed_zone = var.create_zone ? google_dns_managed_zone.main[0].name : var.dns_zone_name
  name         = "${each.value.name}."
  type         = each.value.type
  ttl          = 300
  rrdatas      = each.value.data
}
