#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Initialize Terraform Remote State Backend
# Creates the GCS bucket for Terraform state with versioning and lifecycle.
# Idempotent: skips if bucket already exists.
#
# Usage: ./init-backend.sh <project-id>
# =============================================================================

set -euo pipefail

PROJECT_ID="${1:?Usage: init-backend.sh <project-id>}"
BUCKET="clario360-terraform-state"
REGION="me-central1"

echo "Initializing Terraform state backend..."
echo "  Project: ${PROJECT_ID}"
echo "  Bucket:  gs://${BUCKET}"
echo "  Region:  ${REGION}"

if ! gsutil ls "gs://${BUCKET}" &>/dev/null; then
  echo "Creating state bucket..."
  gsutil mb -p "${PROJECT_ID}" -l "${REGION}" -b on "gs://${BUCKET}"

  echo "Enabling versioning..."
  gsutil versioning set on "gs://${BUCKET}"

  echo "Setting lifecycle policy (keep 10 versions)..."
  gsutil lifecycle set /dev/stdin "gs://${BUCKET}" <<'EOF'
{
  "rule": [
    {
      "action": { "type": "Delete" },
      "condition": { "numNewerVersions": 10 }
    }
  ]
}
EOF

  echo "State bucket created: gs://${BUCKET}"
else
  echo "State bucket already exists: gs://${BUCKET}"
fi

echo ""
echo "Enable required APIs..."
gcloud services enable --project="${PROJECT_ID}" \
  compute.googleapis.com \
  container.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  dns.googleapis.com \
  cloudkms.googleapis.com \
  servicenetworking.googleapis.com \
  cloudresourcemanager.googleapis.com \
  iam.googleapis.com \
  monitoring.googleapis.com \
  logging.googleapis.com \
  cloudtrace.googleapis.com \
  binaryauthorization.googleapis.com \
  containeranalysis.googleapis.com \
  cloudscheduler.googleapis.com

echo "All required APIs enabled."
echo ""
echo "Next steps:"
echo "  cd environments/<env>"
echo "  terraform init"
echo "  terraform plan -var-file=terraform.tfvars"
