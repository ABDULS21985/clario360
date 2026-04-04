# =============================================================================
# 7 Database Definitions
# Each database serves a specific domain within Clario 360
# =============================================================================

locals {
  databases = toset([
    "clario360_iam",       # IAM service — authentication, users, roles, permissions
    "clario360_platform",  # Platform services — audit, workflow, notification, file, onboarding
    "clario360_cyber",     # Cybersecurity — alerts, risks, CTEM, assets
    "clario360_data",      # Data intelligence — sources, pipelines, quality, contradictions
    "clario360_acta",      # Governance — board minutes, resolutions, action items
    "clario360_lex",       # Legal — contracts, obligations, compliance
    "clario360_visus",     # Executive intelligence — dashboards, reports, KPIs
  ])
}

resource "google_sql_database" "databases" {
  for_each = local.databases

  project   = var.project_id
  name      = each.value
  instance  = google_sql_database_instance.main.name
  charset   = "UTF8"
  collation = "en_US.UTF8"
}
