# =============================================================================
# Per-Service Database Users with Least Privilege
# Each service gets its own user with SELECT/INSERT/UPDATE/DELETE only.
# The migrator user gets CREATE privileges for schema migrations.
# =============================================================================

locals {
  # Map of service → database mapping
  service_db_map = {
    iam       = "clario360_iam"
    platform  = "clario360_platform"
    cyber     = "clario360_cyber"
    data      = "clario360_data"
    acta      = "clario360_acta"
    lex       = "clario360_lex"
    visus     = "clario360_visus"
    migrator  = "all"
  }
}

# Generate random passwords for each service user
resource "random_password" "db_passwords" {
  for_each = local.service_db_map

  length           = 32
  special          = true
  override_special = "!@#$%"
}

# Create Cloud SQL users
resource "google_sql_user" "service_users" {
  for_each = local.service_db_map

  project  = var.project_id
  name     = "clario360_${each.key}"
  instance = google_sql_database_instance.main.name
  password = random_password.db_passwords[each.key].result
}

# -----------------------------------------------------------------------------
# Grant least-privilege permissions via SQL
# Service users: SELECT, INSERT, UPDATE, DELETE (no CREATE/DROP)
# Migrator user: ALL PRIVILEGES including CREATE
# -----------------------------------------------------------------------------

# Grants for per-service users (least privilege — no schema modification)
resource "null_resource" "service_grants" {
  for_each = { for k, v in local.service_db_map : k => v if k != "migrator" }

  triggers = {
    user     = google_sql_user.service_users[each.key].name
    database = each.value
    instance = google_sql_database_instance.main.connection_name
  }

  provisioner "local-exec" {
    command = <<-EOT
      gcloud sql connect ${google_sql_database_instance.main.name} \
        --project=${var.project_id} \
        --database=${each.value} \
        --quiet <<SQL
      -- Grant connect and schema usage
      GRANT CONNECT ON DATABASE ${each.value} TO clario360_${each.key};
      GRANT USAGE ON SCHEMA public TO clario360_${each.key};

      -- Grant DML permissions on existing tables
      GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO clario360_${each.key};
      GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO clario360_${each.key};

      -- Set default privileges for future tables
      ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO clario360_${each.key};
      ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO clario360_${each.key};

      -- Explicitly deny schema modification
      REVOKE CREATE ON SCHEMA public FROM clario360_${each.key};
      REVOKE ALL ON DATABASE ${each.value} FROM PUBLIC;
      SQL
    EOT
  }

  depends_on = [
    google_sql_database.databases,
    google_sql_user.service_users,
  ]
}

# Grants for migrator user (full privileges — can create/drop tables for migrations)
resource "null_resource" "migrator_grants" {
  for_each = local.databases

  triggers = {
    user     = google_sql_user.service_users["migrator"].name
    database = each.value
    instance = google_sql_database_instance.main.connection_name
  }

  provisioner "local-exec" {
    command = <<-EOT
      gcloud sql connect ${google_sql_database_instance.main.name} \
        --project=${var.project_id} \
        --database=${each.value} \
        --quiet <<SQL
      GRANT ALL PRIVILEGES ON DATABASE ${each.value} TO clario360_migrator;
      GRANT CREATE ON SCHEMA public TO clario360_migrator;
      SQL
    EOT
  }

  depends_on = [
    google_sql_database.databases,
    google_sql_user.service_users,
  ]
}

# -----------------------------------------------------------------------------
# Store passwords in Vault
# -----------------------------------------------------------------------------
resource "vault_generic_secret" "db_passwords" {
  for_each = local.service_db_map

  path = "secret/clario360/${var.environment}/database/${each.key}"

  data_json = jsonencode({
    username = "clario360_${each.key}"
    password = random_password.db_passwords[each.key].result
    host     = google_sql_database_instance.main.private_ip_address
    port     = 5432
    database = each.value == "all" ? "clario360_iam" : each.value
    url = format(
      "postgres://clario360_%s:%s@%s:5432/%s?sslmode=%s",
      each.key,
      random_password.db_passwords[each.key].result,
      google_sql_database_instance.main.private_ip_address,
      each.value == "all" ? "clario360_iam" : each.value,
      var.environment == "dev" ? "disable" : "require"
    )
  })
}
