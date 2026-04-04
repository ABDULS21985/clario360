# =============================================================================
# Clario 360 — Security Module
# GCP Service Accounts, IAM Bindings, Workload Identity Federation
# =============================================================================

locals {
  name_prefix = "clario360-${var.environment}"

  # Service accounts for each Clario 360 service
  service_accounts = {
    apigateway          = { display = "API Gateway", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    iamservice          = { display = "IAM Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    auditservice        = { display = "Audit Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent", "roles/storage.objectViewer"] }
    workflowengine      = { display = "Workflow Engine", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    notificationservice = { display = "Notification Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    fileservice         = { display = "File Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent", "roles/storage.objectAdmin"] }
    cyberservice        = { display = "Cyber Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    dataservice         = { display = "Data Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent", "roles/storage.objectViewer"] }
    actaservice         = { display = "Acta Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    lexservice          = { display = "Lex Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    visusservice        = { display = "Visus Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
    onboardingservice   = { display = "Onboarding Service", roles = ["roles/monitoring.metricWriter", "roles/logging.logWriter", "roles/cloudtrace.agent"] }
  }
}

# Create GCP service accounts
resource "google_service_account" "services" {
  for_each = local.service_accounts

  project      = var.project_id
  account_id   = "clario360-${each.key}"
  display_name = "Clario 360 ${each.value.display} (${var.environment})"
}

# Bind IAM roles to service accounts
resource "google_project_iam_member" "service_roles" {
  for_each = merge([
    for sa_key, sa_val in local.service_accounts : {
      for role in sa_val.roles : "${sa_key}-${replace(role, "/", "-")}" => {
        sa_key = sa_key
        role   = role
      }
    }
  ]...)

  project = var.project_id
  role    = each.value.role
  member  = "serviceAccount:${google_service_account.services[each.value.sa_key].email}"
}

# Workload Identity bindings — allow K8s service accounts to impersonate GCP service accounts
resource "google_service_account_iam_member" "workload_identity" {
  for_each = var.gke_cluster_name != "" ? local.service_accounts : {}

  service_account_id = google_service_account.services[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[clario360/${replace(each.key, "service", "-service")}]"
}
