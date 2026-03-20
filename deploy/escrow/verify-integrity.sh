#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# Clario 360 — Escrow Package Integrity Verification
# ═══════════════════════════════════════════════════════════════════════════════
# Verifies the completeness and integrity of an escrow package.
# Run this FIRST after receiving an escrow deposit to confirm all components
# are present and uncorrupted.
#
# Usage: cd escrow-<version> && ./verification/verify-integrity.sh
# Exit:  0 = all checks pass, N = number of failures
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# Navigate to escrow root (parent of verification/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESCROW_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ESCROW_ROOT}"

PASS=0
FAIL=0
WARN=0

# Colors (if terminal supports them)
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    GREEN='' RED='' YELLOW='' NC=''
fi

check() {
    local description="$1"
    local command="$2"
    if eval "$command" &>/dev/null; then
        printf "  ${GREEN}✓${NC} %s\n" "$description"
        PASS=$((PASS + 1))
    else
        printf "  ${RED}✗${NC} %s\n" "$description"
        FAIL=$((FAIL + 1))
    fi
}

warn_check() {
    local description="$1"
    local command="$2"
    if eval "$command" &>/dev/null; then
        printf "  ${GREEN}✓${NC} %s\n" "$description"
        PASS=$((PASS + 1))
    else
        printf "  ${YELLOW}⚠${NC} %s (non-critical)\n" "$description"
        WARN=$((WARN + 1))
    fi
}

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "  Clario 360 — Escrow Package Integrity Verification"
echo "═══════════════════════════════════════════════════════════════════"
echo ""
echo "  Package root: ${ESCROW_ROOT}"
echo "  Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 1. FILE CHECKSUMS
# ─────────────────────────────────────────────────────────────────────────────
echo "1. File Checksums"
if [ -f SHA256SUMS ]; then
    TOTAL_FILES=$(wc -l < SHA256SUMS | tr -d ' ')
    # Use shasum on macOS, sha256sum on Linux
    if command -v shasum &>/dev/null; then
        CHECKSUM_CMD="shasum -a 256 -c SHA256SUMS"
    else
        CHECKSUM_CMD="sha256sum -c SHA256SUMS"
    fi
    FAILED_CHECKSUMS=$(eval "$CHECKSUM_CMD" 2>/dev/null | grep -c "FAILED" || true)
    if [ "${FAILED_CHECKSUMS}" -eq 0 ]; then
        printf "  ${GREEN}✓${NC} All %s file checksums verified\n" "${TOTAL_FILES}"
        PASS=$((PASS + 1))
    else
        printf "  ${RED}✗${NC} %s of %s checksums FAILED\n" "${FAILED_CHECKSUMS}" "${TOTAL_FILES}"
        FAIL=$((FAIL + 1))
    fi
else
    printf "  ${RED}✗${NC} SHA256SUMS file missing — cannot verify integrity\n"
    FAIL=$((FAIL + 1))
fi

# ─────────────────────────────────────────────────────────────────────────────
# 2. MANIFEST
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "2. Package Manifest"
check "manifest.json exists" "[ -f manifest.json ]"
if [ -f manifest.json ]; then
    check "manifest.json is valid JSON" "python3 -m json.tool manifest.json >/dev/null 2>&1 || jq . manifest.json >/dev/null 2>&1"
    if command -v jq &>/dev/null; then
        VERSION=$(jq -r '.package.version' manifest.json 2>/dev/null || echo "unknown")
        printf "  ℹ Package version: %s\n" "${VERSION}"
    fi
fi

# ─────────────────────────────────────────────────────────────────────────────
# 3. SOURCE CODE
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "3. Source Code Integrity"
check "Go module file exists" "[ -f source/backend/go.mod ]"
check "Go sum file exists" "[ -f source/backend/go.sum ]"
check "Go vendor directory exists" "[ -d source/backend/vendor ]"
check "Go vendor modules.txt exists" "[ -f source/backend/vendor/modules.txt ]"
check "Frontend package.json exists" "[ -f source/frontend/package.json ]"
check "Frontend package-lock.json exists" "[ -f source/frontend/package-lock.json ]"
check "VERSION file exists" "[ -f source/VERSION ]"
check "COMMIT_SHA file exists" "[ -f source/COMMIT_SHA ]"

# Check all services have main.go
SERVICES=(api-gateway iam-service event-bus workflow-engine audit-service
          cyber-service data-service acta-service lex-service visus-service
          file-service notification-service migrator)

for svc in "${SERVICES[@]}"; do
    check "Service ${svc} main.go exists" "[ -f source/backend/cmd/${svc}/main.go ]"
done

# Check internal packages
check "Internal packages exist" "[ -d source/backend/internal ]"
check "Package directory exists" "[ -d source/backend/pkg ]"

# ─────────────────────────────────────────────────────────────────────────────
# 4. DEPENDENCIES
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "4. Vendored Dependencies"
check "Go vendor has source files" "test \$(find source/backend/vendor -name '*.go' 2>/dev/null | wc -l) -gt 0"

GO_VENDOR_COUNT=$(find source/backend/vendor -name '*.go' 2>/dev/null | wc -l | tr -d ' ')
printf "  ℹ Go vendor: %s files\n" "${GO_VENDOR_COUNT}"

check "Node modules cached" \
    "[ -d dependencies/node_modules ] || [ -f dependencies/npm-cache.tar.gz ]"

if [ -d dependencies/node_modules ]; then
    NPM_PKG_COUNT=$(ls dependencies/node_modules 2>/dev/null | wc -l | tr -d ' ')
    printf "  ℹ Node modules: %s packages\n" "${NPM_PKG_COUNT}"
fi

warn_check "Container images present" \
    "ls dependencies/images/*.tar 2>/dev/null | head -1 | grep -q ."

if ls dependencies/images/*.tar &>/dev/null; then
    IMG_COUNT=$(ls dependencies/images/*.tar | wc -l | tr -d ' ')
    printf "  ℹ Container images: %s archives\n" "${IMG_COUNT}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 5. DEPLOYMENT ARTIFACTS
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "5. Deployment Artifacts"
check "Helm Chart.yaml exists" "[ -f deploy/helm/clario360/Chart.yaml ]"
check "Helm values.yaml exists" "[ -f deploy/helm/clario360/values.yaml ]"
check "Helm production values exist" "[ -f deploy/helm/clario360/values-production.yaml ]"
check "Terraform modules directory exists" "[ -d deploy/terraform/modules ]"
check "Terraform production environment exists" "[ -d deploy/terraform/environments/production ]"
check "Docker backend Dockerfile exists" "[ -f deploy/docker/Dockerfile.backend ]"
check "Docker frontend Dockerfile exists" "[ -f deploy/docker/Dockerfile.frontend ]"
check "Docker migrator Dockerfile exists" "[ -f deploy/docker/Dockerfile.migrator ]"
check "docker-compose.yml exists" "[ -f deploy/docker-compose.yml ]"
check "Makefile exists" "[ -f deploy/Makefile ]"

# Terraform modules
if [ -d deploy/terraform/modules ]; then
    TF_MODULES=$(ls -d deploy/terraform/modules/*/ 2>/dev/null | wc -l | tr -d ' ')
    printf "  ℹ Terraform modules: %s\n" "${TF_MODULES}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 6. DATABASE MIGRATIONS
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "6. Database Migrations"
check "Migrations directory exists" "[ -d deploy/migrations ]"

DATABASES=(platform_core cyber_db data_db acta_db audit_db notification_db lex_db visus_db)
for db in "${DATABASES[@]}"; do
    check "Migrations for ${db}" "[ -d deploy/migrations/${db} ] && ls deploy/migrations/${db}/*.sql &>/dev/null"
done

TOTAL_MIGRATIONS=$(find deploy/migrations -name '*.sql' 2>/dev/null | wc -l | tr -d ' ')
printf "  ℹ Total migration files: %s\n" "${TOTAL_MIGRATIONS}"

# ─────────────────────────────────────────────────────────────────────────────
# 7. DOCUMENTATION
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "7. Documentation"
check "Escrow README exists" "[ -f docs/ESCROW_README.md ]"
warn_check "API specification exists" "[ -f docs/api/openapi.yaml ] || [ -f docs/api/openapi.json ]"
check "Architecture docs exist" "[ -d docs/architecture ]"
check "Runbooks exist" "[ -d docs/runbooks ]"
check "Build instructions exist" "[ -f build/BUILD_INSTRUCTIONS.md ]"

if [ -d docs/runbooks ]; then
    RUNBOOK_COUNT=$(find docs/runbooks -name '*.md' 2>/dev/null | wc -l | tr -d ' ')
    printf "  ℹ Runbooks: %s procedures\n" "${RUNBOOK_COUNT}"
fi

if [ -d docs/architecture ]; then
    ARCH_COUNT=$(find docs/architecture -name '*.md' 2>/dev/null | wc -l | tr -d ' ')
    printf "  ℹ Architecture docs: %s files\n" "${ARCH_COUNT}"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 8. TOOLS
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "8. Build Tools"
check "Tools directory exists" "[ -d tools ]"
warn_check "Go compiler included" "[ -f tools/go1.22.0.tar.gz ]"
warn_check "Node.js included" "[ -f tools/node-v20.11.0.tar.xz ]"
warn_check "kubectl included" "[ -f tools/kubectl ]"
warn_check "helm included" "[ -f tools/helm ]"
warn_check "terraform included" "[ -f tools/terraform ]"

# ─────────────────────────────────────────────────────────────────────────────
# 9. SENSITIVE FILE CHECK
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "9. Security Check (no sensitive files)"
check "No .env files (outside vendor)" \
    "! find . -path '*/vendor/*' -prune -o -path '*/node_modules/*' -prune -o \( -name '.env' -o -name '.env.local' -o -name '.env.production' \) -print 2>/dev/null | grep -q ."
check "No private keys" \
    "! find . -path '*/vendor/*' -prune -o -path '*/node_modules/*' -prune -o \( -name '*.pem' -o -name '*.key' -o -name 'id_rsa*' \) -print 2>/dev/null | grep -q ."
check "No terraform state" \
    "! find . \( -name '*.tfstate' -o -name '*.tfstate.backup' \) -print 2>/dev/null | grep -q ."
check "No credentials files (outside vendor)" \
    "! find . -path '*/vendor/*' -prune -o -path '*/node_modules/*' -prune -o \( -name 'credentials.json' -o -name 'service-account*.json' \) -print 2>/dev/null | grep -q ."

# ─────────────────────────────────────────────────────────────────────────────
# RESULTS
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo ""
printf "  Results:  ${GREEN}%d passed${NC}" "${PASS}"
if [ "${WARN}" -gt 0 ]; then
    printf ",  ${YELLOW}%d warnings${NC}" "${WARN}"
fi
if [ "${FAIL}" -gt 0 ]; then
    printf ",  ${RED}%d failed${NC}" "${FAIL}"
fi
echo ""
echo ""

if [ "${FAIL}" -eq 0 ]; then
    printf "  STATUS: ${GREEN}VERIFICATION PASSED ✓${NC}\n"
    if [ "${WARN}" -gt 0 ]; then
        echo "  (${WARN} non-critical warnings — review above)"
    fi
else
    printf "  STATUS: ${RED}VERIFICATION FAILED ✗${NC}\n"
    echo "  ${FAIL} critical check(s) failed — escrow package may be incomplete"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════════"

exit "${FAIL}"
