# Clario 360 — Operational Runbooks

| Field | Value |
|-------|-------|
| Last Updated | 2026-03-06 |
| Owner | Platform Team |
| Review Frequency | Quarterly |

## Overview

This runbook library provides step-by-step procedures for operating the Clario 360 platform. Every runbook contains **exact commands** — no pseudocode or "refer to external document" references. Each procedure has been validated against the production environment.

## How to Use These Runbooks

1. **Identify the issue** using alerts, monitoring dashboards, or user reports
2. **Find the matching runbook** using the index below or the category directories
3. **Follow every step in order** — do not skip diagnosis steps
4. **Verify resolution** using the verification commands at the end of each runbook
5. **Complete post-incident tasks** including updating the runbook if new patterns are discovered

## Prerequisites

All runbooks assume the operator has:

- `kubectl` configured with access to the `clario360` namespace
- `psql` client installed (PostgreSQL 15+)
- `jq` installed for JSON processing
- `curl` installed for HTTP requests
- Access to Grafana dashboards at `https://grafana.clario360.io`
- Access to Vault at `https://vault.clario360.io`
- Appropriate K8s RBAC permissions (at minimum `clario360-operator` role)

## Environment Variables

Set these before running runbook commands:

```bash
export NAMESPACE=clario360
export PG_HOST=postgresql.clario360.svc.cluster.local
export PG_USER=clario360_admin
export API_URL=https://api.clario360.io
export GRAFANA_URL=https://grafana.clario360.io
export VAULT_ADDR=https://vault.clario360.io
```

## Runbook Index

### Incident Response (IR)

| ID | Title | Severity | Description |
|----|-------|----------|-------------|
| [IR-001](incident-response/IR-001-service-outage.md) | Service Outage | P1 — Critical | Single service outage diagnosis and recovery |
| [IR-002](incident-response/IR-002-database-failure.md) | Database Failure | P1 — Critical | Database connectivity loss |
| [IR-003](incident-response/IR-003-kafka-failure.md) | Kafka Failure | P1 — Critical | Kafka broker failure |
| [IR-004](incident-response/IR-004-redis-failure.md) | Redis Failure | P2 — High | Redis cache failure |
| [IR-005](incident-response/IR-005-certificate-expiry.md) | Certificate Expiry | P2 — High | TLS certificate expiration |
| [IR-006](incident-response/IR-006-disk-full.md) | Disk Full | P2 — High | Disk space exhaustion |
| [IR-007](incident-response/IR-007-memory-exhaustion.md) | Memory Exhaustion | P2 — High | OOMKilled pods |
| [IR-008](incident-response/IR-008-security-breach.md) | Security Breach | P1 — Critical | Suspected security incident |
| [IR-009](incident-response/IR-009-data-corruption.md) | Data Corruption | P1 — Critical | Data integrity issue |
| [IR-010](incident-response/IR-010-dns-failure.md) | DNS Failure | P1 — Critical | DNS resolution failure |

### Operations (OP)

| ID | Title | Frequency | Description |
|----|-------|-----------|-------------|
| [OP-001](operations/OP-001-daily-checks.md) | Daily Checks | Daily | Daily operational health checks |
| [OP-002](operations/OP-002-weekly-maintenance.md) | Weekly Maintenance | Weekly | Weekly maintenance procedures |
| [OP-003](operations/OP-003-monthly-review.md) | Monthly Review | Monthly | Monthly operational review |
| [OP-004](operations/OP-004-backup-verification.md) | Backup Verification | Weekly | Backup integrity verification |
| [OP-005](operations/OP-005-certificate-renewal.md) | Certificate Renewal | As needed | TLS certificate rotation |
| [OP-006](operations/OP-006-secret-rotation.md) | Secret Rotation | Quarterly | Vault secret rotation |
| [OP-007](operations/OP-007-database-maintenance.md) | Database Maintenance | Weekly/Monthly | PostgreSQL VACUUM, REINDEX, partitions |
| [OP-008](operations/OP-008-kafka-maintenance.md) | Kafka Maintenance | Weekly | Topic management, consumer lag |
| [OP-009](operations/OP-009-log-management.md) | Log Management | Weekly | Log rotation, retention, archival |
| [OP-010](operations/OP-010-user-management.md) | User Management | As needed | Admin user operations |

### Scaling (SC)

| ID | Title | Description |
|----|-------|-------------|
| [SC-001](scaling/SC-001-horizontal-scaling.md) | Horizontal Scaling | Scale out services |
| [SC-002](scaling/SC-002-database-scaling.md) | Database Scaling | Read replicas, connection pooling |
| [SC-003](scaling/SC-003-kafka-scaling.md) | Kafka Scaling | Add brokers, rebalance partitions |
| [SC-004](scaling/SC-004-node-pool-scaling.md) | Node Pool Scaling | Add K8s nodes |
| [SC-005](scaling/SC-005-capacity-planning.md) | Capacity Planning | Capacity planning guide |

### Troubleshooting (TS)

| ID | Title | Description |
|----|-------|-------------|
| [TS-001](troubleshooting/TS-001-slow-api-responses.md) | Slow API Responses | API latency investigation |
| [TS-002](troubleshooting/TS-002-failed-pipelines.md) | Failed Pipelines | Data pipeline failure investigation |
| [TS-003](troubleshooting/TS-003-missing-events.md) | Missing Events | Kafka event loss investigation |
| [TS-004](troubleshooting/TS-004-auth-failures.md) | Auth Failures | Authentication issue debugging |
| [TS-005](troubleshooting/TS-005-websocket-disconnects.md) | WebSocket Disconnects | WebSocket connectivity issues |
| [TS-006](troubleshooting/TS-006-search-not-returning.md) | Search Not Returning | Full-text search problems |
| [TS-007](troubleshooting/TS-007-high-cpu-usage.md) | High CPU Usage | CPU investigation |
| [TS-008](troubleshooting/TS-008-connection-pool-exhaustion.md) | Connection Pool Exhaustion | Database connection issues |
| [TS-009](troubleshooting/TS-009-audit-chain-broken.md) | Audit Chain Broken | Audit hash chain integrity failure |
| [TS-010](troubleshooting/TS-010-cross-suite-events-not-flowing.md) | Cross-Suite Events Not Flowing | Event integration problems |

### Deployment (DP)

| ID | Title | Description |
|----|-------|-------------|
| [DP-001](deployment/DP-001-new-release.md) | New Release | Deploy a new version |
| [DP-002](deployment/DP-002-rollback.md) | Rollback | Rollback to previous version |
| [DP-003](deployment/DP-003-hotfix.md) | Hotfix | Emergency hotfix procedure |
| [DP-004](deployment/DP-004-database-migration.md) | Database Migration | Run database migrations |
| [DP-005](deployment/DP-005-feature-flags.md) | Feature Flags | Feature flag management |

### Automation

| Script | Description |
|--------|-------------|
| [daily-check.sh](scripts/daily-check.sh) | Automated daily health check (OP-001) |
| [daily-health-check-cronjob.yaml](scripts/daily-health-check-cronjob.yaml) | K8s CronJob for daily checks |

## Clario 360 Services Reference

| Service | Port | Health Endpoint | Readiness Endpoint |
|---------|------|-----------------|-------------------|
| api-gateway | 8080 | `/healthz` | `/readyz` |
| iam-service | 8080 | `/healthz` | `/readyz` |
| audit-service | 8080 | `/healthz` | `/readyz` |
| workflow-engine | 8080 | `/healthz` | `/readyz` |
| notification-service | 8080 | `/healthz` | `/readyz` |
| cyber-service | 8080 | `/healthz` | `/readyz` |
| data-service | 8080 | `/healthz` | `/readyz` |
| acta-service | 8080 | `/healthz` | `/readyz` |
| lex-service | 8080 | `/healthz` | `/readyz` |
| visus-service | 8080 | `/healthz` | `/readyz` |

## Escalation Matrix

| Severity | Response Time | Escalation Path |
|----------|---------------|-----------------|
| P1 — Critical | 15 minutes | On-call → Platform Lead → CTO |
| P2 — High | 1 hour | On-call → Platform Lead |
| P3 — Medium | 4 hours | Platform Team |
| P4 — Low | Next business day | Platform Team |

## Grafana Dashboards

| Dashboard | URL Path | Description |
|-----------|----------|-------------|
| Cluster Overview | `/d/cluster-overview` | K8s node and pod health |
| API Performance | `/d/api-performance` | Request rates, latencies, error rates |
| Database Performance | `/d/db-performance` | Query duration, connections, replication |
| Kafka Overview | `/d/kafka-overview` | Broker health, consumer lag, throughput |
| Redis Overview | `/d/redis-overview` | Memory usage, hit rate, connections |
| Service Health | `/d/service-health` | Per-service health and metrics |
| Security Overview | `/d/security-overview` | Auth failures, rate limiting, WAF events |
