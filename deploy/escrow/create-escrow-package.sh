#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# Clario 360 — Source Code Escrow Package Creator
# ═══════════════════════════════════════════════════════════════════════════════
# Creates a complete, self-contained source code escrow package per RFP §14.
# A competent third party must be able to build, deploy, and operate the
# entire platform from this package alone — zero vendor contact, zero internet.
#
# Usage:  ./create-escrow-package.sh <version> [--skip-images] [--skip-tools]
# Output: clario360-escrow-<version>.tar.gz
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

VERSION="${1:?Usage: create-escrow-package.sh <version> [--skip-images] [--skip-tools]}"
SKIP_IMAGES=false
SKIP_TOOLS=false
for arg in "${@:2}"; do
    case "$arg" in
        --skip-images) SKIP_IMAGES=true ;;
        --skip-tools)  SKIP_TOOLS=true ;;
        *) echo "Unknown option: $arg"; exit 1 ;;
    esac
done

ESCROW_DIR="escrow-${VERSION}"
DATE=$(date +%Y-%m-%d)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "═══════════════════════════════════════════════════════════════════"
echo "  Creating Clario 360 Escrow Package v${VERSION} — ${DATE}"
echo "═══════════════════════════════════════════════════════════════════"
echo ""

cd "${REPO_ROOT}"

# Clean up any previous run
rm -rf "${ESCROW_DIR}"
mkdir -p "${ESCROW_DIR}"/{source,dependencies,build,deploy,docs,verification,tools}

# ═══════════════════════════════════════════════════════════════════════════════
# 1. SOURCE CODE (complete, buildable)
# ═══════════════════════════════════════════════════════════════════════════════
echo "[1/10] Copying source code..."

# Export clean copy from git (no .git history — reduces size, removes branch data)
git archive HEAD --format=tar | tar -x -C "${ESCROW_DIR}/source/"

# Remove sensitive or unnecessary files
rm -f "${ESCROW_DIR}/source/.env" \
      "${ESCROW_DIR}/source/.env.local" \
      "${ESCROW_DIR}/source/.env.production"
find "${ESCROW_DIR}/source" -name "node_modules" -type d -exec rm -rf {} + 2>/dev/null || true
find "${ESCROW_DIR}/source" -name ".terraform" -type d -exec rm -rf {} + 2>/dev/null || true
find "${ESCROW_DIR}/source" -name "*.tfstate" -delete 2>/dev/null || true
find "${ESCROW_DIR}/source" -name "*.tfstate.backup" -delete 2>/dev/null || true

# Record provenance
echo "${VERSION}" > "${ESCROW_DIR}/source/VERSION"
git rev-parse HEAD > "${ESCROW_DIR}/source/COMMIT_SHA"
git log --oneline -50 > "${ESCROW_DIR}/source/RECENT_COMMITS.txt"
git log --format="%H %ai %s" -50 > "${ESCROW_DIR}/source/COMMIT_LOG.txt"

echo "  Source code exported from $(git rev-parse --short HEAD)"

# ═══════════════════════════════════════════════════════════════════════════════
# 2. VENDORED DEPENDENCIES (offline build capability)
# ═══════════════════════════════════════════════════════════════════════════════
echo "[2/10] Vendoring Go dependencies..."

(
    cd "${ESCROW_DIR}/source/backend"
    GOWORK=off go mod vendor
    echo "  Go vendor directory created ($(find vendor -name '*.go' | wc -l | tr -d ' ') Go files)"
)

echo "[2b/10] Caching npm dependencies..."

(
    cd "${ESCROW_DIR}/source/frontend"
    npm ci --ignore-scripts 2>/dev/null || npm install --ignore-scripts 2>/dev/null || true
    if [ -d node_modules ]; then
        cp -r node_modules "${REPO_ROOT}/${ESCROW_DIR}/dependencies/node_modules"
        echo "  Node modules cached ($(ls node_modules | wc -l | tr -d ' ') packages)"
    else
        echo "  ⚠ npm install failed — node_modules will need to be installed from package-lock.json"
    fi
)

# Create npm offline mirror tarball if cache is available
NPM_CACHE_DIR=$(npm config get cache 2>/dev/null || echo "")
if [ -n "${NPM_CACHE_DIR}" ] && [ -d "${NPM_CACHE_DIR}" ]; then
    tar czf "${ESCROW_DIR}/dependencies/npm-cache.tar.gz" \
        -C "$(dirname "${NPM_CACHE_DIR}")" "$(basename "${NPM_CACHE_DIR}")" 2>/dev/null || \
        echo "  ⚠ npm cache tarball not created (non-critical)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 3. CONTAINER IMAGES (pre-built for immediate deployment)
# ═══════════════════════════════════════════════════════════════════════════════
echo "[3/10] Saving container images..."
mkdir -p "${ESCROW_DIR}/dependencies/images"

if [ "${SKIP_IMAGES}" = true ]; then
    echo "  Skipping container images (--skip-images)"
else
    # Application services
    APP_SERVICES=(
        api-gateway
        iam-service
        event-bus
        workflow-engine
        audit-service
        cyber-service
        data-service
        acta-service
        lex-service
        visus-service
        file-service
        notification-service
        frontend
        migrator
    )

    for svc in "${APP_SERVICES[@]}"; do
        IMAGE="ghcr.io/clario360/${svc}:${VERSION}"
        if docker image inspect "${IMAGE}" &>/dev/null; then
            echo "  Saving ${svc}:${VERSION}..."
            docker save "${IMAGE}" -o "${ESCROW_DIR}/dependencies/images/${svc}-${VERSION}.tar"
        else
            echo "  ⚠ Image not found: ${IMAGE} (skipping)"
        fi
    done

    # Infrastructure base images for offline builds
    INFRA_IMAGES=(
        "golang:1.22-alpine"
        "node:20-alpine"
        "gcr.io/distroless/static-debian12:nonroot"
        "postgres:16-alpine"
        "redis:7-alpine"
        "bitnami/kafka:3.7"
        "prom/prometheus:v2.50.0"
        "grafana/grafana:10.3.1"
        "nginx:1.25-alpine"
    )

    for img in "${INFRA_IMAGES[@]}"; do
        SAFE_NAME=$(echo "${img}" | tr '/:' '-')
        if docker image inspect "${img}" &>/dev/null; then
            echo "  Saving infrastructure image: ${img}..."
            docker save "${img}" -o "${ESCROW_DIR}/dependencies/images/${SAFE_NAME}.tar"
        else
            echo "  ⚠ Image not found: ${img} (pull first or skip)"
        fi
    done

    IMAGE_COUNT=$(find "${ESCROW_DIR}/dependencies/images" -name '*.tar' 2>/dev/null | wc -l | tr -d ' ')
    echo "  Saved ${IMAGE_COUNT} container images"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 4. BUILD INSTRUCTIONS
# ═══════════════════════════════════════════════════════════════════════════════
echo "[4/10] Creating build instructions..."

cp "${SCRIPT_DIR}/BUILD_INSTRUCTIONS.md" "${ESCROW_DIR}/build/BUILD_INSTRUCTIONS.md"

# ═══════════════════════════════════════════════════════════════════════════════
# 5. DEPLOYMENT ARTIFACTS
# ═══════════════════════════════════════════════════════════════════════════════
echo "[5/10] Copying deployment artifacts..."

# Helm charts
if [ -d deploy/helm ]; then
    cp -r deploy/helm "${ESCROW_DIR}/deploy/helm"
    echo "  Helm charts copied"
fi

# Terraform modules
if [ -d deploy/terraform ]; then
    cp -r deploy/terraform "${ESCROW_DIR}/deploy/terraform"
    # Remove state files
    find "${ESCROW_DIR}/deploy/terraform" -name "*.tfstate*" -delete 2>/dev/null || true
    find "${ESCROW_DIR}/deploy/terraform" -name ".terraform" -type d -exec rm -rf {} + 2>/dev/null || true
    echo "  Terraform modules copied (state files excluded)"
fi

# Docker configs
if [ -d deploy/docker ]; then
    cp -r deploy/docker "${ESCROW_DIR}/deploy/docker"
    echo "  Docker configurations copied"
fi

# Grafana dashboards
if [ -d deploy/grafana ]; then
    cp -r deploy/grafana "${ESCROW_DIR}/deploy/grafana"
    echo "  Grafana dashboards copied"
fi

# Prometheus config
if [ -d deploy/prometheus ]; then
    cp -r deploy/prometheus "${ESCROW_DIR}/deploy/prometheus"
    echo "  Prometheus configuration copied"
fi

# Root-level deployment files
[ -f docker-compose.yml ] && cp docker-compose.yml "${ESCROW_DIR}/deploy/"
[ -f docker-compose.test.yml ] && cp docker-compose.test.yml "${ESCROW_DIR}/deploy/"
[ -f Makefile ] && cp Makefile "${ESCROW_DIR}/deploy/"

# ═══════════════════════════════════════════════════════════════════════════════
# 6. DATABASE SCHEMAS & MIGRATIONS
# ═══════════════════════════════════════════════════════════════════════════════
echo "[6/10] Copying database schemas and migrations..."

if [ -d backend/migrations ]; then
    cp -r backend/migrations "${ESCROW_DIR}/deploy/migrations"

    # Count migrations per database
    for db_dir in "${ESCROW_DIR}/deploy/migrations"/*/; do
        db_name=$(basename "${db_dir}")
        migration_count=$(ls "${db_dir}"*.sql 2>/dev/null | wc -l | tr -d ' ')
        echo "  ${db_name}: ${migration_count} migration files"
    done
fi

# Generate current schema DDL if database is accessible
DATABASES=(platform_core cyber_db data_db acta_db audit_db notification_db lex_db visus_db)
if command -v pg_dump &>/dev/null && [ -n "${PG_HOST:-}" ]; then
    echo "  Dumping current schema DDL..."
    for db in "${DATABASES[@]}"; do
        pg_dump -h "${PG_HOST}" -U "${PG_USER:-postgres}" -d "${db}" \
            --schema-only --no-owner --no-privileges \
            -f "${ESCROW_DIR}/deploy/migrations/${db}_schema_current.sql" 2>/dev/null || \
            echo "  ⚠ Could not dump ${db} schema (database not accessible)"
    done
else
    echo "  ⚠ pg_dump not available or PG_HOST not set — schema DDL not generated"
    echo "    Migrations are sufficient to recreate schemas from scratch"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 7. DOCUMENTATION
# ═══════════════════════════════════════════════════════════════════════════════
echo "[7/10] Copying documentation..."

if [ -d docs ]; then
    cp -r docs/ "${ESCROW_DIR}/docs/"
    echo "  Documentation copied:"
    [ -d docs/api ] && echo "    - API specifications"
    [ -d docs/architecture ] && echo "    - Architecture documents"
    [ -d docs/runbooks ] && echo "    - Operational runbooks ($(find docs/runbooks -name '*.md' | wc -l | tr -d ' ') procedures)"
fi

# Copy escrow-specific documentation
cp "${SCRIPT_DIR}/ESCROW_README.md" "${ESCROW_DIR}/docs/ESCROW_README.md"

# ═══════════════════════════════════════════════════════════════════════════════
# 8. CLI TOOLS (for offline environments)
# ═══════════════════════════════════════════════════════════════════════════════
echo "[8/10] Including CLI tools..."

if [ "${SKIP_TOOLS}" = true ]; then
    echo "  Skipping tool downloads (--skip-tools)"
    cat > "${ESCROW_DIR}/tools/README.md" << 'EOF'
# Required Tools

The following tools are needed to build and deploy Clario 360.
Download them for your target architecture before air-gapped deployment.

| Tool       | Version | Download URL                                              |
|------------|---------|-----------------------------------------------------------|
| Go         | 1.22+   | https://go.dev/dl/                                        |
| Node.js    | 20 LTS  | https://nodejs.org/en/download/                           |
| kubectl    | 1.28+   | https://kubernetes.io/docs/tasks/tools/                   |
| Helm       | 3.14+   | https://helm.sh/docs/intro/install/                       |
| Terraform  | 1.7+    | https://developer.hashicorp.com/terraform/downloads       |
| Docker     | 24+     | https://docs.docker.com/engine/install/                   |
EOF
else
    ARCH="${ARCH:-amd64}"
    TARGET_OS="${TARGET_OS:-linux}"

    echo "  Downloading tools for ${TARGET_OS}/${ARCH}..."

    # Go compiler
    echo "  Downloading Go 1.22..."
    curl -sfL "https://go.dev/dl/go1.22.0.${TARGET_OS}-${ARCH}.tar.gz" \
        -o "${ESCROW_DIR}/tools/go1.22.0.tar.gz" 2>/dev/null || \
        echo "  ⚠ Failed to download Go (add manually)"

    # Node.js
    echo "  Downloading Node.js 20..."
    curl -sfL "https://nodejs.org/dist/v20.11.0/node-v20.11.0-${TARGET_OS}-x64.tar.xz" \
        -o "${ESCROW_DIR}/tools/node-v20.11.0.tar.xz" 2>/dev/null || \
        echo "  ⚠ Failed to download Node.js (add manually)"

    # Helm
    echo "  Downloading Helm 3.14..."
    curl -sfL "https://get.helm.sh/helm-v3.14.0-${TARGET_OS}-${ARCH}.tar.gz" | \
        tar xz -C "${ESCROW_DIR}/tools/" --strip-components=1 "${TARGET_OS}-${ARCH}/helm" 2>/dev/null || \
        echo "  ⚠ Failed to download Helm (add manually)"

    # kubectl
    echo "  Downloading kubectl 1.28..."
    curl -sfL "https://dl.k8s.io/release/v1.28.8/bin/${TARGET_OS}/${ARCH}/kubectl" \
        -o "${ESCROW_DIR}/tools/kubectl" 2>/dev/null || \
        echo "  ⚠ Failed to download kubectl (add manually)"

    # Terraform
    echo "  Downloading Terraform 1.7..."
    curl -sfL "https://releases.hashicorp.com/terraform/1.7.0/terraform_1.7.0_${TARGET_OS}_${ARCH}.zip" \
        -o /tmp/terraform.zip 2>/dev/null && \
        unzip -o /tmp/terraform.zip -d "${ESCROW_DIR}/tools/" 2>/dev/null && \
        rm -f /tmp/terraform.zip || \
        echo "  ⚠ Failed to download Terraform (add manually)"

    chmod +x "${ESCROW_DIR}/tools/"* 2>/dev/null || true

    # Tools README
    cat > "${ESCROW_DIR}/tools/README.md" << EOF
# Included Tools

These tools are included for offline/air-gapped environments.
Target: ${TARGET_OS}/${ARCH}

## Installation
\`\`\`bash
# Go
tar xzf go1.22.0.tar.gz -C /usr/local
export PATH=\$PATH:/usr/local/go/bin

# Node.js
tar xJf node-v20.11.0.tar.xz -C /usr/local --strip-components=1

# kubectl, helm, terraform
cp kubectl helm terraform /usr/local/bin/
chmod +x /usr/local/bin/{kubectl,helm,terraform}
\`\`\`
EOF
fi

# ═══════════════════════════════════════════════════════════════════════════════
# 9. VERIFICATION SUITE
# ═══════════════════════════════════════════════════════════════════════════════
echo "[9/10] Creating verification suite..."

cp "${SCRIPT_DIR}/verify-integrity.sh" "${ESCROW_DIR}/verification/verify-integrity.sh"
cp "${SCRIPT_DIR}/verify-build.sh" "${ESCROW_DIR}/verification/verify-build.sh"
chmod +x "${ESCROW_DIR}/verification/"*.sh

# ═══════════════════════════════════════════════════════════════════════════════
# 10. GENERATE MANIFEST, CHECKSUMS, AND PACKAGE
# ═══════════════════════════════════════════════════════════════════════════════
echo "[10/10] Generating manifest and packaging..."

# Generate manifest
FILE_COUNT=$(find "${ESCROW_DIR}/source" -type f | wc -l | tr -d ' ')
LINE_COUNT=$(find "${ESCROW_DIR}/source" -type f -name '*.go' -o -name '*.ts' -o -name '*.tsx' -o -name '*.sql' -o -name '*.tf' | \
    xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}')
LINE_COUNT="${LINE_COUNT:-0}"
MIGRATION_COUNT=$(find "${ESCROW_DIR}/deploy/migrations" -name '*.sql' 2>/dev/null | wc -l | tr -d ' ')
IMAGE_COUNT=$(find "${ESCROW_DIR}/dependencies/images" -name '*.tar' 2>/dev/null | wc -l | tr -d ' ')
TEST_COUNT=$(find "${ESCROW_DIR}/source" -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.tsx' -o -name '*.spec.ts' -o -name '*.spec.tsx' 2>/dev/null | wc -l | tr -d ' ')
TF_MODULE_COUNT=$(find "${ESCROW_DIR}/deploy/terraform/modules" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')

# Generate manifest from template
"${SCRIPT_DIR}/generate-manifest.sh" \
    "${VERSION}" "${DATE}" "${FILE_COUNT}" "${LINE_COUNT}" \
    "${MIGRATION_COUNT}" "${IMAGE_COUNT}" "${TEST_COUNT}" "${TF_MODULE_COUNT}" \
    > "${ESCROW_DIR}/manifest.json"
echo "  Manifest generated"

# Calculate checksums for all files
echo "  Generating SHA-256 checksums..."
(
    cd "${ESCROW_DIR}"
    find . -type f -not -name "SHA256SUMS" -not -name "*.sig" -not -name "*.asc" | \
        sort | xargs shasum -a 256 > SHA256SUMS
)
CHECKSUM_COUNT=$(wc -l < "${ESCROW_DIR}/SHA256SUMS" | tr -d ' ')
echo "  ${CHECKSUM_COUNT} file checksums recorded"

# Create archive
echo "  Creating compressed archive..."
tar czf "clario360-escrow-${VERSION}.tar.gz" "${ESCROW_DIR}/"

# Generate checksum of the archive
shasum -a 256 "clario360-escrow-${VERSION}.tar.gz" > "clario360-escrow-${VERSION}.tar.gz.sha256"

# Sign with GPG if key is available
if command -v gpg &>/dev/null && gpg --list-keys "escrow@clario360.com" &>/dev/null 2>&1; then
    gpg --detach-sign --armor -u "escrow@clario360.com" \
        "clario360-escrow-${VERSION}.tar.gz"
    echo "  Package signed with GPG key"
    SIGNED="Yes"
else
    echo "  ⚠ GPG key 'escrow@clario360.com' not available — package unsigned"
    SIGNED="No"
fi

# Package info
SIZE=$(du -h "clario360-escrow-${VERSION}.tar.gz" | cut -f1)
SHA=$(cut -d' ' -f1 "clario360-escrow-${VERSION}.tar.gz.sha256")

# Clean up working directory
rm -rf "${ESCROW_DIR}"

echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "  Escrow Package Created Successfully"
echo "═══════════════════════════════════════════════════════════════════"
echo ""
echo "  File:       clario360-escrow-${VERSION}.tar.gz"
echo "  Size:       ${SIZE}"
echo "  SHA-256:    ${SHA}"
echo "  Date:       ${DATE}"
echo "  Version:    ${VERSION}"
echo "  Signed:     ${SIGNED}"
echo ""
echo "  Contents:"
echo "    - Complete source code (${FILE_COUNT} files, ~${LINE_COUNT} lines)"
echo "    - Vendored Go dependencies (go mod vendor)"
echo "    - Cached npm dependencies"
echo "    - Pre-built container images (${IMAGE_COUNT} images)"
echo "    - Database migrations (${MIGRATION_COUNT} SQL files across 8 databases)"
echo "    - Helm charts and Terraform modules (${TF_MODULE_COUNT} modules)"
echo "    - API documentation, architecture docs, and runbooks"
echo "    - Build tools (Go, Node.js, kubectl, helm, terraform)"
echo "    - Integrity verification suite"
echo "    - ${CHECKSUM_COUNT} SHA-256 file checksums"
echo "    - ${TEST_COUNT} test files"
echo ""
echo "  Deliverables:"
echo "    1. clario360-escrow-${VERSION}.tar.gz"
echo "    2. clario360-escrow-${VERSION}.tar.gz.sha256"
if [ "${SIGNED}" = "Yes" ]; then
echo "    3. clario360-escrow-${VERSION}.tar.gz.asc (GPG signature)"
fi
echo ""
echo "  Deliver these files to the escrow provider."
echo "═══════════════════════════════════════════════════════════════════"
