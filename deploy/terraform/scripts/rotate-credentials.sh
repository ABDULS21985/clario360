#!/usr/bin/env bash
# =============================================================================
# Clario 360 — Credential Rotation
# Rotates database passwords and API keys via Vault.
# Generates new passwords, updates Cloud SQL users, and stores in Vault.
#
# Usage: ./rotate-credentials.sh <dev|staging|production> [service]
#   service: optional, rotates only the specified service (e.g., "iam", "cyber")
#            if omitted, rotates all service credentials
# =============================================================================

set -euo pipefail

ENV="${1:?Usage: rotate-credentials.sh <dev|staging|production> [service]}"
SERVICE="${2:-all}"

if [[ ! "$ENV" =~ ^(dev|staging|production)$ ]]; then
  echo "Error: Invalid environment '${ENV}'. Must be one of: dev, staging, production"
  exit 1
fi

if [[ "$ENV" == "production" ]]; then
  echo "============================================="
  echo "  WARNING: PRODUCTION CREDENTIAL ROTATION"
  echo "============================================="
  echo ""
  echo "You are about to rotate credentials in PRODUCTION."
  echo "Ensure all services will be restarted after rotation."
  echo ""
  read -rp "Type 'production' to confirm: " CONFIRM
  if [[ "$CONFIRM" != "production" ]]; then
    echo "Aborted."
    exit 1
  fi
fi

INSTANCE="clario360-${ENV}-pg"
PROJECT="clario360-${ENV}"

SERVICES=("iam" "platform" "cyber" "data" "acta" "lex" "visus" "migrator")

if [[ "$SERVICE" != "all" ]]; then
  if [[ ! " ${SERVICES[*]} " =~ " ${SERVICE} " ]]; then
    echo "Error: Unknown service '${SERVICE}'. Valid: ${SERVICES[*]}"
    exit 1
  fi
  SERVICES=("$SERVICE")
fi

echo "=== Credential Rotation: ${ENV} ==="
echo "Services: ${SERVICES[*]}"
echo ""

for svc in "${SERVICES[@]}"; do
  echo "Rotating credentials for: clario360_${svc}"

  # Generate new password
  NEW_PASSWORD=$(openssl rand -base64 32 | tr -dc 'A-Za-z0-9!@#$%' | head -c 32)

  # Update Cloud SQL user password
  echo "  Updating Cloud SQL password..."
  gcloud sql users set-password "clario360_${svc}" \
    --instance="${INSTANCE}" \
    --project="${PROJECT}" \
    --password="${NEW_PASSWORD}" \
    --quiet

  # Determine database name
  case "$svc" in
    migrator) DB_NAME="clario360_iam" ;;
    *) DB_NAME="clario360_${svc}" ;;
  esac

  # Get instance private IP
  PRIVATE_IP=$(gcloud sql instances describe "${INSTANCE}" \
    --project="${PROJECT}" \
    --format='value(ipAddresses[0].ipAddress)')

  SSL_MODE="require"
  if [[ "$ENV" == "dev" ]]; then
    SSL_MODE="disable"
  fi

  # Store in Vault
  echo "  Updating Vault secret..."
  vault kv put "secret/clario360/${ENV}/database/${svc}" \
    username="clario360_${svc}" \
    password="${NEW_PASSWORD}" \
    host="${PRIVATE_IP}" \
    port="5432" \
    database="${DB_NAME}" \
    url="postgres://clario360_${svc}:${NEW_PASSWORD}@${PRIVATE_IP}:5432/${DB_NAME}?sslmode=${SSL_MODE}"

  echo "  Done."
  echo ""
done

echo "=== Credential rotation complete ==="
echo ""
echo "IMPORTANT: Restart affected services to pick up new credentials."
echo "Services using Vault Agent Injector will auto-rotate within the TTL window."
