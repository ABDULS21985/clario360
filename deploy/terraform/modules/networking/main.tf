# =============================================================================
# Clario 360 — Networking Module
# VPC, subnets (public/private/isolated), Cloud NAT, Private Service Access,
# and firewall rules for a secure, multi-tier network architecture.
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"
}

# -----------------------------------------------------------------------------
# VPC
# -----------------------------------------------------------------------------
resource "google_compute_network" "vpc" {
  project                         = var.project_id
  name                            = "${local.name_prefix}-vpc"
  auto_create_subnetworks         = false
  routing_mode                    = "REGIONAL"
  delete_default_routes_on_create = true
}

# Default route to internet gateway (only used by subnets with NAT)
resource "google_compute_route" "default_internet" {
  project          = var.project_id
  name             = "${local.name_prefix}-default-internet"
  network          = google_compute_network.vpc.id
  dest_range       = "0.0.0.0/0"
  next_hop_gateway = "default-internet-gateway"
  priority         = 1000
}

# -----------------------------------------------------------------------------
# Subnets
# -----------------------------------------------------------------------------

# Public subnet — load balancer backends, NAT gateway
resource "google_compute_subnetwork" "public" {
  project                  = var.project_id
  name                     = "${local.name_prefix}-public"
  region                   = var.region
  network                  = google_compute_network.vpc.id
  ip_cidr_range            = var.public_subnet_cidr
  purpose                  = "PRIVATE"
  private_ip_google_access = true
}

# Private subnet — GKE nodes, internal services
resource "google_compute_subnetwork" "private" {
  project                  = var.project_id
  name                     = "${local.name_prefix}-private"
  region                   = var.region
  network                  = google_compute_network.vpc.id
  ip_cidr_range            = var.private_subnet_cidr
  purpose                  = "PRIVATE"
  private_ip_google_access = true

  secondary_ip_range {
    range_name    = "gke-pods"
    ip_cidr_range = var.pods_cidr
  }

  secondary_ip_range {
    range_name    = "gke-services"
    ip_cidr_range = var.services_cidr
  }
}

# Isolated subnet — Cloud SQL, Redis, Vault (no internet access)
resource "google_compute_subnetwork" "isolated" {
  project                  = var.project_id
  name                     = "${local.name_prefix}-isolated"
  region                   = var.region
  network                  = google_compute_network.vpc.id
  ip_cidr_range            = var.isolated_subnet_cidr
  purpose                  = "PRIVATE"
  private_ip_google_access = true
}

# -----------------------------------------------------------------------------
# Cloud NAT (public + private subnets only — NOT isolated)
# -----------------------------------------------------------------------------
resource "google_compute_router" "router" {
  project = var.project_id
  name    = "${local.name_prefix}-router"
  region  = var.region
  network = google_compute_network.vpc.id
}

resource "google_compute_router_nat" "nat" {
  project                            = var.project_id
  name                               = "${local.name_prefix}-nat"
  router                             = google_compute_router.router.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = google_compute_subnetwork.public.id
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }

  subnetwork {
    name                    = google_compute_subnetwork.private.id
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }

  min_ports_per_vm                 = 64
  max_ports_per_vm                 = 4096
  tcp_established_idle_timeout_sec = 1200
  tcp_transitory_idle_timeout_sec  = 30
}

# -----------------------------------------------------------------------------
# Private Service Access (for Cloud SQL private connectivity)
# -----------------------------------------------------------------------------
resource "google_compute_global_address" "psa" {
  project       = var.project_id
  name          = "${local.name_prefix}-psa"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 20
  network       = google_compute_network.vpc.id
}

resource "google_service_networking_connection" "psa" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.psa.name]
}

# -----------------------------------------------------------------------------
# Firewall Rules
# -----------------------------------------------------------------------------

# RULE 1: Allow internal communication within VPC
resource "google_compute_firewall" "allow_internal" {
  project     = var.project_id
  name        = "${local.name_prefix}-allow-internal"
  network     = google_compute_network.vpc.id
  direction   = "INGRESS"
  priority    = 1000
  description = "Allow internal communication within VPC"

  source_ranges = [
    var.public_subnet_cidr,
    var.private_subnet_cidr,
    var.isolated_subnet_cidr,
    var.pods_cidr,
    var.services_cidr,
  ]

  allow {
    protocol = "tcp"
  }

  allow {
    protocol = "udp"
  }

  allow {
    protocol = "icmp"
  }
}

# RULE 2: Allow GCP health check probes
resource "google_compute_firewall" "allow_health_checks" {
  project     = var.project_id
  name        = "${local.name_prefix}-allow-health-checks"
  network     = google_compute_network.vpc.id
  direction   = "INGRESS"
  priority    = 1000
  description = "Allow GCP health check probes"

  source_ranges = [
    "35.191.0.0/16",
    "130.211.0.0/22",
  ]

  allow {
    protocol = "tcp"
  }

  target_tags = ["gke-node"]
}

# RULE 3: Allow IAP SSH tunneling (dev/staging only)
resource "google_compute_firewall" "allow_iap_ssh" {
  count = var.enable_iap_ssh ? 1 : 0

  project     = var.project_id
  name        = "${local.name_prefix}-allow-iap-ssh"
  network     = google_compute_network.vpc.id
  direction   = "INGRESS"
  priority    = 1000
  description = "Allow IAP SSH tunneling for debugging"

  source_ranges = ["35.235.240.0/20"]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }
}

# RULE 4: Deny internet egress from isolated subnet
resource "google_compute_firewall" "deny_egress_isolated" {
  project     = var.project_id
  name        = "${local.name_prefix}-deny-egress-isolated"
  network     = google_compute_network.vpc.id
  direction   = "EGRESS"
  priority    = 65534
  description = "Deny internet egress from isolated subnet"

  destination_ranges = ["0.0.0.0/0"]

  deny {
    protocol = "all"
  }

  target_tags = ["isolated"]
}

# RULE 5: Allow isolated subnet to reach internal services only
resource "google_compute_firewall" "allow_egress_isolated_internal" {
  project     = var.project_id
  name        = "${local.name_prefix}-allow-egress-isolated-internal"
  network     = google_compute_network.vpc.id
  direction   = "EGRESS"
  priority    = 1000
  description = "Allow isolated subnet to reach internal services only"

  destination_ranges = [
    var.private_subnet_cidr,
    var.isolated_subnet_cidr,
    var.pods_cidr,
    var.services_cidr,
    google_compute_global_address.psa.address,
  ]

  allow {
    protocol = "tcp"
  }

  target_tags = ["isolated"]
}
