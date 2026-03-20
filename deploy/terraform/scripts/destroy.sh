#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Terraform Destroy
# Destroys infrastructure for the specified environment.
# BLOCKS production destruction via script — must use Terraform directly.
#
# Usage: ./destroy.sh <dev|staging>
# =============================================================================

set -euo pipefail

ENV="${1:?Usage: destroy.sh <dev|staging>}"

if [[ "$ENV" == "production" ]]; then
  echo "============================================="
  echo "  BLOCKED: Cannot destroy production"
  echo "============================================="
  echo ""
  echo "Production infrastructure cannot be destroyed via this script."
  echo "If you truly need to destroy production resources, use Terraform"
  echo "directly with manual approval and proper change management."
  exit 1
fi

if [[ ! "$ENV" =~ ^(dev|staging)$ ]]; then
  echo "Error: Invalid environment '${ENV}'. Must be one of: dev, staging"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_DIR="${SCRIPT_DIR}/../environments/${ENV}"

if [[ ! -d "$ENV_DIR" ]]; then
  echo "Error: Environment directory not found: ${ENV_DIR}"
  exit 1
fi

cd "${ENV_DIR}"

echo "=== Terraform Destroy: ${ENV} ==="
echo ""
echo "This will DESTROY all infrastructure in the ${ENV} environment."
echo ""
read -rp "Type '${ENV}' to confirm destruction: " CONFIRM
if [[ "$CONFIRM" != "$ENV" ]]; then
  echo "Aborted."
  exit 1
fi

terraform init -backend-config="prefix=${ENV}"
terraform destroy -var-file=terraform.tfvars

echo ""
echo "Infrastructure destroyed for ${ENV}."
