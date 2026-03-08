#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Terraform Apply
# Applies a saved plan for the specified environment.
# Production requires explicit confirmation.
#
# Usage: ./apply.sh <dev|staging|production>
# =============================================================================

set -euo pipefail

ENV="${1:?Usage: apply.sh <dev|staging|production>}"

if [[ ! "$ENV" =~ ^(dev|staging|production)$ ]]; then
  echo "Error: Invalid environment '${ENV}'. Must be one of: dev, staging, production"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_DIR="${SCRIPT_DIR}/../environments/${ENV}"

if [[ ! -f "${ENV_DIR}/tfplan" ]]; then
  echo "Error: No plan file found at ${ENV_DIR}/tfplan"
  echo "Run './plan.sh ${ENV}' first to generate a plan."
  exit 1
fi

cd "${ENV_DIR}"

if [[ "$ENV" == "production" ]]; then
  echo "============================================="
  echo "  WARNING: PRODUCTION DEPLOYMENT"
  echo "============================================="
  echo ""
  echo "You are about to apply infrastructure changes"
  echo "to the PRODUCTION environment."
  echo ""
  read -rp "Type 'production' to confirm: " CONFIRM
  if [[ "$CONFIRM" != "production" ]]; then
    echo "Aborted."
    exit 1
  fi
  echo ""
fi

echo "=== Applying Terraform: ${ENV} ==="
terraform apply tfplan

echo ""
echo "Applied successfully to ${ENV}."

# Clean up plan file
rm -f tfplan
echo "Plan file removed."
