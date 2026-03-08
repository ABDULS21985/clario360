#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# Clario 360 — Escrow Package Build Verification
# ═══════════════════════════════════════════════════════════════════════════════
# Verifies that the escrow package can actually be built from source.
# Run AFTER verify-integrity.sh passes. Requires Go and Node.js installed.
#
# Usage: cd escrow-<version> && ./verification/verify-build.sh
# Exit:  0 = all builds succeed, N = number of failures
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESCROW_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${ESCROW_ROOT}"

PASS=0
FAIL=0
SKIP=0

# Colors
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    GREEN='' RED='' YELLOW='' BLUE='' NC=''
fi

check() {
    local description="$1"
    local command="$2"
    printf "  ${BLUE}…${NC} %s" "$description"
    if (eval "$command") &>/dev/null; then
        printf "\r  ${GREEN}✓${NC} %s\n" "$description"
        PASS=$((PASS + 1))
    else
        printf "\r  ${RED}✗${NC} %s\n" "$description"
        FAIL=$((FAIL + 1))
    fi
}

skip() {
    local description="$1"
    local reason="$2"
    printf "  ${YELLOW}⊘${NC} %s — %s\n" "$description" "$reason"
    SKIP=$((SKIP + 1))
}

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "  Clario 360 — Escrow Build Verification"
echo "═══════════════════════════════════════════════════════════════════"
echo ""
echo "  Package root: ${ESCROW_ROOT}"
echo "  Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 1. TOOL AVAILABILITY
# ─────────────────────────────────────────────────────────────────────────────
echo "1. Build Tool Availability"

HAS_GO=false
HAS_NODE=false
HAS_DOCKER=false
HAS_MAKE=false

if command -v go &>/dev/null; then
    HAS_GO=true
    printf "  ${GREEN}✓${NC} Go: %s\n" "$(go version 2>&1)"
else
    printf "  ${YELLOW}⊘${NC} Go not found — install from tools/go1.22.0.tar.gz\n"
fi

if command -v node &>/dev/null; then
    HAS_NODE=true
    printf "  ${GREEN}✓${NC} Node.js: %s\n" "$(node --version 2>&1)"
else
    printf "  ${YELLOW}⊘${NC} Node.js not found — install from tools/node-v20.11.0.tar.xz\n"
fi

if command -v docker &>/dev/null; then
    HAS_DOCKER=true
    printf "  ${GREEN}✓${NC} Docker: %s\n" "$(docker --version 2>&1)"
else
    printf "  ${YELLOW}⊘${NC} Docker not found\n"
fi

if command -v make &>/dev/null; then
    HAS_MAKE=true
    printf "  ${GREEN}✓${NC} Make available\n"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 2. GO COMPILATION (using vendored dependencies)
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "2. Go Service Compilation (offline, -mod=vendor)"

SERVICES=(api-gateway iam-service event-bus workflow-engine audit-service
          cyber-service data-service acta-service lex-service visus-service
          file-service notification-service migrator)

if [ "${HAS_GO}" = true ]; then
    BUILD_DIR=$(mktemp -d)
    trap "rm -rf ${BUILD_DIR}" EXIT

    for svc in "${SERVICES[@]}"; do
        if [ -d "source/backend/cmd/${svc}" ]; then
            check "Compile ${svc}" \
                "cd source/backend && GOWORK=off go build -mod=vendor -o '${BUILD_DIR}/${svc}' ./cmd/${svc}/"
        else
            skip "Compile ${svc}" "source not found"
        fi
    done

    # Verify binaries were created
    BUILT_COUNT=$(ls "${BUILD_DIR}" 2>/dev/null | wc -l | tr -d ' ')
    printf "  ℹ Successfully compiled: %s/%s services\n" "${BUILT_COUNT}" "${#SERVICES[@]}"
else
    for svc in "${SERVICES[@]}"; do
        skip "Compile ${svc}" "Go not available"
    done
fi

# ─────────────────────────────────────────────────────────────────────────────
# 3. GO TESTS (using vendored dependencies)
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "3. Go Unit Tests (sample — vendor mode)"

if [ "${HAS_GO}" = true ]; then
    # Run tests on core packages only (full test suite may need infrastructure)
    TESTABLE_PKGS=(
        "./pkg/..."
        "./internal/auth/..."
    )
    for pkg in "${TESTABLE_PKGS[@]}"; do
        check "Test ${pkg}" \
            "cd source/backend && GOWORK=off go test -mod=vendor -short -count=1 ${pkg} 2>&1"
    done
else
    skip "Go unit tests" "Go not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 4. GO VET (static analysis)
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "4. Go Vet (static analysis)"

if [ "${HAS_GO}" = true ]; then
    check "go vet all packages" \
        "cd source/backend && GOWORK=off go vet -mod=vendor ./... 2>&1"
else
    skip "go vet" "Go not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 5. FRONTEND BUILD
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "5. Frontend Build"

if [ "${HAS_NODE}" = true ]; then
    # Copy vendored node_modules if available
    if [ -d dependencies/node_modules ] && [ ! -d source/frontend/node_modules ]; then
        echo "  ℹ Copying vendored node_modules..."
        cp -r dependencies/node_modules source/frontend/
    fi

    if [ -d source/frontend/node_modules ]; then
        check "TypeScript compilation" \
            "cd source/frontend && npx tsc --noEmit 2>&1"
        check "Next.js production build" \
            "cd source/frontend && npx next build 2>&1"
    else
        skip "Frontend build" "node_modules not available (run npm ci first)"
    fi
else
    skip "Frontend build" "Node.js not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 6. DOCKER IMAGE BUILD (if Docker available)
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "6. Container Image Build (sample)"

if [ "${HAS_DOCKER}" = true ]; then
    # Test one service build to verify Dockerfiles work
    if [ -f deploy/docker/Dockerfile.backend ]; then
        check "Docker build api-gateway (test)" \
            "cd source && docker build -f ../deploy/docker/Dockerfile.backend \
                --build-arg BUILD_SERVICE=api-gateway \
                -t clario360/escrow-test:verify . 2>&1 && \
             docker rmi clario360/escrow-test:verify 2>&1"
    else
        skip "Docker build" "Dockerfile.backend not found"
    fi
else
    skip "Docker image build" "Docker not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 7. MIGRATION SYNTAX CHECK
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "7. Migration File Validation"

DATABASES=(platform_core cyber_db data_db acta_db audit_db notification_db lex_db visus_db)
for db in "${DATABASES[@]}"; do
    if [ -d "deploy/migrations/${db}" ]; then
        UP_COUNT=$(ls deploy/migrations/${db}/*.up.sql 2>/dev/null | wc -l | tr -d ' ')
        DOWN_COUNT=$(ls deploy/migrations/${db}/*.down.sql 2>/dev/null | wc -l | tr -d ' ')
        check "${db}: up/down migrations balanced (${UP_COUNT}/${DOWN_COUNT})" \
            "[ '${UP_COUNT}' -eq '${DOWN_COUNT}' ] && [ '${UP_COUNT}' -gt 0 ]"
    fi
done

# ─────────────────────────────────────────────────────────────────────────────
# 8. HELM CHART VALIDATION
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "8. Helm Chart Validation"

if command -v helm &>/dev/null || [ -f tools/helm ]; then
    HELM_CMD="helm"
    [ -f tools/helm ] && HELM_CMD="./tools/helm"
    check "Helm chart lint" \
        "${HELM_CMD} lint deploy/helm/clario360/ 2>&1"
    check "Helm template render" \
        "${HELM_CMD} template test deploy/helm/clario360/ 2>&1"
else
    skip "Helm chart validation" "helm not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# 9. TERRAFORM VALIDATION
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "9. Terraform Validation"

if command -v terraform &>/dev/null || [ -f tools/terraform ]; then
    TF_CMD="terraform"
    [ -f tools/terraform ] && TF_CMD="./tools/terraform"
    for env_dir in deploy/terraform/environments/*/; do
        env_name=$(basename "${env_dir}")
        check "Terraform validate: ${env_name}" \
            "cd '${env_dir}' && ${TF_CMD} init -backend=false 2>&1 && ${TF_CMD} validate 2>&1"
    done
else
    skip "Terraform validation" "terraform not available"
fi

# ─────────────────────────────────────────────────────────────────────────────
# RESULTS
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo ""
printf "  Results:  ${GREEN}%d passed${NC}" "${PASS}"
if [ "${SKIP}" -gt 0 ]; then
    printf ",  ${YELLOW}%d skipped${NC}" "${SKIP}"
fi
if [ "${FAIL}" -gt 0 ]; then
    printf ",  ${RED}%d failed${NC}" "${FAIL}"
fi
echo ""
echo ""

if [ "${FAIL}" -eq 0 ]; then
    if [ "${SKIP}" -gt 0 ]; then
        printf "  STATUS: ${GREEN}BUILD VERIFICATION PASSED ✓${NC} (with %d skipped checks)\n" "${SKIP}"
        echo "  Install missing tools and re-run for full verification."
    else
        printf "  STATUS: ${GREEN}BUILD VERIFICATION PASSED ✓${NC}\n"
    fi
else
    printf "  STATUS: ${RED}BUILD VERIFICATION FAILED ✗${NC}\n"
    echo "  ${FAIL} build check(s) failed — escrow package may not be buildable"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════════"

exit "${FAIL}"
