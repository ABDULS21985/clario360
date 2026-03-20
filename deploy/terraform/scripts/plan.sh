#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Terraform Plan
# Validates and generates a plan for the specified environment.
#
# Usage: ./plan.sh <dev|staging|production>
# =============================================================================

set -euo pipefail

ENV="${1:?Usage: plan.sh <dev|staging|production>}"

if [[ ! "$ENV" =~ ^(dev|staging|production)$ ]]; then
  echo "Error: Invalid environment '${ENV}'. Must be one of: dev, staging, production"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_DIR="${SCRIPT_DIR}/../environments/${ENV}"

if [[ ! -d "$ENV_DIR" ]]; then
  echo "Error: Environment directory not found: ${ENV_DIR}"
  exit 1
fi

echo "=== Terraform Plan: ${ENV} ==="
echo ""

cd "${ENV_DIR}"

echo "Initializing Terraform..."
terraform init -backend-config="prefix=${ENV}"

echo ""
echo "Validating configuration..."
terraform validate

echo ""
echo "Generating plan..."
terraform plan -var-file=terraform.tfvars -out=tfplan

echo ""
echo "Plan saved to: ${ENV_DIR}/tfplan"
echo "Review the plan above, then run: ./apply.sh ${ENV}"
