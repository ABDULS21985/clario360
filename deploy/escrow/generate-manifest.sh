#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# Clario 360 — Escrow Manifest Generator
# ═══════════════════════════════════════════════════════════════════════════════
# Generates a machine-readable JSON manifest for the escrow deposit.
# Called by create-escrow-package.sh — can also be run standalone.
#
# Usage: ./generate-manifest.sh <version> <date> <file_count> <line_count> \
#            <migration_count> <image_count> <test_count> <tf_module_count>
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

VERSION="${1:?Usage: generate-manifest.sh <version> <date> <file_count> ...}"
DATE="${2:?}"
FILE_COUNT="${3:-0}"
LINE_COUNT="${4:-0}"
MIGRATION_COUNT="${5:-0}"
IMAGE_COUNT="${6:-0}"
TEST_COUNT="${7:-0}"
TF_MODULE_COUNT="${8:-0}"

cat << EOF
{
  "package": {
    "name": "clario360-enterprise-ai-platform",
    "version": "${VERSION}",
    "date": "${DATE}",
    "format": "tar.gz",
    "checksum_algorithm": "SHA-256"
  },
  "contents": {
    "source_code": {
      "languages": ["go", "typescript", "sql", "hcl", "yaml", "bash"],
      "go_version": "1.22",
      "node_version": "20",
      "total_files": ${FILE_COUNT},
      "total_lines": ${LINE_COUNT}
    },
    "services": [
      {"name": "api-gateway", "type": "go", "path": "backend/cmd/api-gateway", "description": "API gateway with rate limiting, circuit breaker, WebSocket proxy"},
      {"name": "iam-service", "type": "go", "path": "backend/cmd/iam-service", "description": "Identity and access management with RS256 JWT"},
      {"name": "event-bus", "type": "go", "path": "backend/cmd/event-bus", "description": "Event streaming and message bus"},
      {"name": "workflow-engine", "type": "go", "path": "backend/cmd/workflow-engine", "description": "Workflow orchestration and human task management"},
      {"name": "audit-service", "type": "go", "path": "backend/cmd/audit-service", "description": "Immutable audit logging with hash chain"},
      {"name": "cyber-service", "type": "go", "path": "backend/cmd/cyber-service", "description": "Cybersecurity risk management and vulnerability tracking"},
      {"name": "data-service", "type": "go", "path": "backend/cmd/data-service", "description": "Data management and analytics"},
      {"name": "acta-service", "type": "go", "path": "backend/cmd/acta-service", "description": "Contract and asset tracking"},
      {"name": "lex-service", "type": "go", "path": "backend/cmd/lex-service", "description": "Legal and compliance management"},
      {"name": "visus-service", "type": "go", "path": "backend/cmd/visus-service", "description": "Visualization and reporting"},
      {"name": "file-service", "type": "go", "path": "backend/cmd/file-service", "description": "File storage with encryption and virus scanning"},
      {"name": "notification-service", "type": "go", "path": "backend/cmd/notification-service", "description": "Multi-channel notifications with WebSocket support"},
      {"name": "migrator", "type": "go", "path": "backend/cmd/migrator", "description": "Database schema migration runner"},
      {"name": "frontend", "type": "typescript", "path": "frontend", "description": "Next.js 14 dashboard with Tailwind and shadcn/ui"}
    ],
    "databases": [
      {"name": "platform_core", "description": "Core platform: users, tenants, roles, permissions"},
      {"name": "cyber_db", "description": "Cybersecurity: vulnerabilities, risks, assets, controls"},
      {"name": "data_db", "description": "Data management: datasets, quality, lineage"},
      {"name": "acta_db", "description": "Asset tracking: contracts, vendors, licenses"},
      {"name": "audit_db", "description": "Audit logs: immutable event chain"},
      {"name": "notification_db", "description": "Notifications: channels, preferences, delivery status"},
      {"name": "lex_db", "description": "Legal: regulations, compliance mappings, obligations"},
      {"name": "visus_db", "description": "Visualization: dashboards, reports, widgets"}
    ],
    "migrations": ${MIGRATION_COUNT},
    "container_images": ${IMAGE_COUNT},
    "helm_charts": 1,
    "terraform_modules": ${TF_MODULE_COUNT},
    "api_endpoints": "~300",
    "test_files": ${TEST_COUNT}
  },
  "build_requirements": {
    "go": "1.22+",
    "nodejs": "20+",
    "docker": "24+",
    "kubernetes": "1.28+",
    "postgresql": "16+",
    "redis": "7+",
    "kafka": "3.7+"
  },
  "deployment_targets": {
    "cloud": "Google Cloud Platform (GKE)",
    "on_premises": "Any Kubernetes 1.28+ cluster",
    "air_gapped": "Supported via offline deployment scripts"
  },
  "compliance": {
    "frameworks": ["ISO 27001", "NCA ECC", "SAMA CSF", "NDMO"],
    "data_residency": "Saudi Arabia (ME-CENTRAL2)",
    "encryption": "AES-256-GCM at rest, TLS 1.3 in transit"
  },
  "verification": {
    "integrity_script": "verification/verify-integrity.sh",
    "build_script": "verification/verify-build.sh",
    "checksum_file": "SHA256SUMS",
    "expected_result": "all checks pass"
  },
  "escrow_terms": {
    "rfp_reference": "RFP §14 — Source Code Escrow",
    "deposit_schedule": "Quarterly (E1–E4)",
    "release_conditions": [
      "Vendor ceases business operations",
      "Vendor fails to maintain platform per SLA",
      "Vendor files for bankruptcy or insolvency",
      "Material breach of support obligations (30-day cure period)"
    ],
    "independence_guarantee": "Package enables full build, deploy, and operate capability without vendor contact or internet access"
  }
}
EOF
