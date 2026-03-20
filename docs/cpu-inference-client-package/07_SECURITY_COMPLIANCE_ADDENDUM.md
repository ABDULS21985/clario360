# Security & Compliance Addendum — CPU Inference Infrastructure

| Attribute    | Value                                                    |
|--------------|----------------------------------------------------------|
| **For**      | Compliance Officers, Security Architects, Legal, Audit   |
| **Date**     | 2026-03-14                                               |
| **Status**   | Phase 1 Delivered                                        |
| **Read time**| 8 minutes                                                |

---

## Purpose

This addendum documents the security controls, data protection measures, and compliance alignment of Clario360's CPU-based inference and benchmarking infrastructure. It supplements the existing platform security documentation with controls specific to the compute layer.

---

## Table of Contents

1. [Security Architecture](#1-security-architecture)
2. [Data Classification & Protection](#2-data-classification--protection)
3. [Access Control](#3-access-control)
4. [Multi-Tenant Isolation](#4-multi-tenant-isolation)
5. [Network Security](#5-network-security)
6. [Cryptographic Controls](#6-cryptographic-controls)
7. [Audit & Logging](#7-audit--logging)
8. [Vulnerability Management](#8-vulnerability-management)
9. [Compliance Framework Mapping](#9-compliance-framework-mapping)
10. [Third-Party Dependencies](#10-third-party-dependencies)
11. [Incident Response Considerations](#11-incident-response-considerations)

---

## 1. Security Architecture

### Defence-in-Depth Model

The compute infrastructure follows the same defence-in-depth model as the core Clario360 platform:

```
Layer 1: Network Boundary
├── API Gateway with rate limiting and circuit breaker
├── TLS termination (TLS 1.3)
└── No direct access to inference servers from external networks

Layer 2: Authentication & Authorization
├── RS256 JWT tokens with short expiry
├── Per-request tenant context extraction
└── Role-based permission gating on admin pages

Layer 3: Application Security
├── Strict JSON decoding (rejects unknown fields)
├── Input validation on all write endpoints
├── Parameterised SQL queries (no string concatenation)
└── Context-scoped timeouts on all external calls

Layer 4: Data Isolation
├── PostgreSQL Row-Level Security on all 4 tables
├── Tenant ID in every query (enforced at repository layer)
└── Cascading deletes tied to tenant lifecycle

Layer 5: Infrastructure
├── Non-root container execution
├── Read-only model file mounts
├── Minimal container images (no build tools in runtime)
└── Health probes for automated recovery
```

### Trust Boundaries

```
┌─────────────────────────────────────────────────────────┐
│  TRUSTED: Clario360 Internal                            │
│                                                          │
│  ┌───────────┐   ┌──────────────┐   ┌───────────────┐  │
│  │ Frontend  │──▶│ API Gateway  │──▶│ AI Governance  │  │
│  │           │   │ + Auth       │   │ Module         │  │
│  └───────────┘   └──────────────┘   └───────┬───────┘  │
│                                              │          │
│  ┌───────────────────────────────────────────▼───────┐  │
│  │  Database Layer (RLS enforced)                    │  │
│  └───────────────────────────────────────────────────┘  │
│                                                          │
├──────────────────────── Trust Boundary ──────────────────┤
│                                                          │
│  SEMI-TRUSTED: Inference Servers                         │
│                                                          │
│  ┌─────────────────┐  ┌─────────────────┐               │
│  │ llama.cpp (CPU) │  │ vLLM (GPU)      │               │
│  │ Internal network│  │ Internal network│               │
│  └─────────────────┘  └─────────────────┘               │
│                                                          │
│  Inference servers are treated as semi-trusted:          │
│  • Responses are not blindly trusted                     │
│  • Server URLs are validated on registration             │
│  • API keys are stored encrypted                         │
│  • Timeouts prevent hanging connections                  │
│                                                          │
├──────────────────────── Trust Boundary ──────────────────┤
│                                                          │
│  UNTRUSTED: External Networks                            │
│  (Not applicable in air-gapped deployments)              │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Data Classification & Protection

### Data Elements

| Data Element             | Classification | Storage             | Encryption at Rest | Encryption in Transit |
|--------------------------|----------------|----------------------|--------------------|-----------------------|
| Inference server names   | Internal       | PostgreSQL           | Yes (TDE/volume)   | Yes (TLS)            |
| Server base URLs         | Internal       | PostgreSQL           | Yes                | Yes                  |
| Server API keys          | Confidential   | PostgreSQL           | Yes                | Yes                  |
| Benchmark prompts        | Varies*        | PostgreSQL (JSONB)   | Yes                | Yes                  |
| Benchmark responses      | Varies*        | PostgreSQL (JSONB)   | Yes                | Yes                  |
| Latency/throughput metrics| Internal      | PostgreSQL           | Yes                | Yes                  |
| Cost model pricing       | Internal       | PostgreSQL           | Yes                | Yes                  |
| Raw iteration results    | Varies*        | PostgreSQL (JSONB)   | Yes                | Yes                  |

*Benchmark prompts and responses inherit the classification of the data they contain. If testing with production-like security data, they should be classified accordingly.*

### Data Retention

| Data Type              | Retention Policy                                      |
|------------------------|-------------------------------------------------------|
| Inference servers      | Retained until explicitly decommissioned; decommissioned records kept for audit trail |
| Benchmark suites       | Retained until explicitly deleted                     |
| Benchmark runs         | Retained indefinitely (historical analysis)           |
| Raw iteration results  | Stored as JSONB in benchmark runs; follows run retention |
| Cost models            | Retained until explicitly removed                     |

### Data Minimisation

| Principle                      | Implementation                                      |
|--------------------------------|-----------------------------------------------------|
| No PII in benchmarks           | Prompt datasets should not contain personally identifiable information |
| Response truncation            | Raw responses limited to 1 MB per iteration         |
| Metric aggregation             | Only statistical aggregates (p50/p95/p99) are surfaced; raw data available but not displayed by default |
| API key masking                | Server API keys omitted from list responses where marked with `omitempty` |

---

## 3. Access Control

### Authentication

| Mechanism              | Details                                              |
|------------------------|------------------------------------------------------|
| Token type             | RS256 JWT signed by IAM service                     |
| Token lifetime         | Configurable (default: 15 minutes access, 7 days refresh) |
| Key management         | RSA key pair managed by IAM service                  |
| Session handling        | Access tokens in memory (never in cookies/localStorage); refresh via httpOnly cookie |

### Authorisation

| Operation                    | Required                                        |
|------------------------------|-------------------------------------------------|
| Read operations (list, get)  | Valid JWT with tenant context                   |
| Write operations (create, update, delete) | Valid JWT + authenticated user ID  |
| Admin UI access              | `users:read` permission (via PermissionRedirect)|

### Principle of Least Privilege

| Role                 | Can Do                                              | Cannot Do                         |
|----------------------|-----------------------------------------------------|-----------------------------------|
| Platform Admin       | Full CRUD on servers, suites, runs, cost models     | Access other tenants' data        |
| Security Analyst     | View servers, suites, run results                   | Create/delete servers             |
| Read-only Viewer     | View dashboards and results                         | Create or modify any resources    |

---

## 4. Multi-Tenant Isolation

### Database-Level Isolation

All four compute tables enforce PostgreSQL Row-Level Security:

| Table                    | RLS Policy                                                  |
|--------------------------|-------------------------------------------------------------|
| `ai_inference_servers`   | `USING (tenant_id = current_setting('app.current_tenant_id')::uuid)` |
| `ai_benchmark_suites`    | `USING (tenant_id = current_setting('app.current_tenant_id')::uuid)` |
| `ai_benchmark_runs`      | `USING (tenant_id = current_setting('app.current_tenant_id')::uuid)` |
| `ai_compute_cost_models` | `USING (tenant_id = current_setting('app.current_tenant_id')::uuid)` |

### Isolation Guarantees

| Guarantee                              | Mechanism                                        |
|----------------------------------------|--------------------------------------------------|
| Tenant A cannot read Tenant B's data   | RLS filters all SELECT queries by tenant_id      |
| Tenant A cannot modify Tenant B's data | RLS filters all UPDATE/DELETE queries             |
| Cross-tenant enumeration impossible    | Unique indexes scoped to tenant_id               |
| Tenant deletion cascades cleanly       | `ON DELETE CASCADE` from tenants table            |
| No shared inference servers            | Each server is tenant-scoped (not global)         |

### Application-Level Enforcement

Beyond database RLS, the application layer also enforces isolation:

| Layer       | Enforcement                                                |
|-------------|-------------------------------------------------------------|
| Handler     | Extracts `tenant_id` from JWT; rejects if missing          |
| Service     | Passes `tenant_id` to all repository calls                 |
| Repository  | Sets `app.current_tenant_id` session variable before queries|

This dual enforcement (application + database) provides defence-in-depth against tenant data leakage.

---

## 5. Network Security

### Standard Deployment

| Communication Path                  | Protocol | Authentication          | Encryption |
|-------------------------------------|----------|-------------------------|------------|
| Client → Frontend                   | HTTPS    | Session cookies         | TLS 1.3    |
| Frontend → API Gateway              | HTTPS    | Bearer JWT              | TLS 1.3    |
| API Gateway → AI Governance Module  | In-process | N/A (same binary)    | N/A        |
| AI Governance → PostgreSQL          | TCP      | Username/password       | TLS optional|
| AI Governance → Inference Server    | HTTP(S)  | Bearer API key (optional)| TLS optional|

### Air-Gapped Deployment

| Communication Path                  | Protocol | Authentication          | Encryption |
|-------------------------------------|----------|-------------------------|------------|
| All paths above                     | Same     | Same                    | Same       |
| Outbound internet                   | **Blocked** | N/A                  | N/A        |
| Inbound from external               | **Blocked** | N/A                  | N/A        |

### Inference Server Network Recommendations

| Recommendation                                   | Rationale                                    |
|--------------------------------------------------|----------------------------------------------|
| Run inference servers on a dedicated VLAN/subnet | Minimise blast radius of compromise          |
| Restrict inference server access to backend only | Prevent direct client access                 |
| Use TLS between backend and inference servers    | Protect prompts and responses in transit     |
| Firewall inference servers from internet         | No legitimate reason for external access     |

---

## 6. Cryptographic Controls

### In Transit

| Path                               | Protocol    | Minimum Version |
|------------------------------------|-------------|-----------------|
| All external-facing endpoints      | TLS         | 1.2 (1.3 preferred) |
| Internal service-to-service        | TLS or mTLS | 1.2             |
| Database connections               | TLS         | 1.2             |

### At Rest

| Data Store               | Encryption Method                            |
|--------------------------|----------------------------------------------|
| PostgreSQL               | Volume-level encryption (cloud-managed or LUKS) |
| Model files (GGUF)       | Volume-level encryption                      |
| Backup storage           | AES-256 encrypted backups                    |

### Key Management

| Key Type                 | Management                                    |
|--------------------------|-----------------------------------------------|
| JWT signing keys         | RSA 2048+ bit, managed by IAM service        |
| Database credentials     | Environment variables or secrets manager      |
| Inference server API keys| Stored in database, protected by RLS          |
| TLS certificates         | Certificate manager or manual provisioning    |

---

## 7. Audit & Logging

### Audit Events

| Event                           | Logged Data                                          |
|---------------------------------|------------------------------------------------------|
| Server registered               | User ID, tenant ID, server name, backend type        |
| Server decommissioned           | User ID, tenant ID, server ID                        |
| Benchmark suite created         | User ID, tenant ID, suite name, model slug           |
| Benchmark run started           | User ID, tenant ID, suite ID, server ID              |
| Benchmark run completed/failed  | Run ID, status, duration, error message (if failed)  |
| Cost model created              | User ID, tenant ID, backend type, pricing            |
| Comparison executed             | User ID, run IDs, recommendation result              |

### Log Format

All logs use structured JSON via zerolog:

```json
{
  "level": "info",
  "time": "2026-03-14T10:30:00Z",
  "service": "ai_benchmark",
  "handler": "ai_benchmark",
  "tenant_id": "aaaaaaaa-0000-0000-0000-000000000001",
  "user_id": "bbbbbbbb-1111-1111-1111-111111111111",
  "event": "benchmark_run_completed",
  "run_id": "cccccccc-2222-2222-2222-222222222222",
  "duration_seconds": 300,
  "status": "completed"
}
```

### Log Retention

| Log Type              | Retention        | Storage                |
|-----------------------|------------------|------------------------|
| Application logs      | 90 days          | Log aggregation system |
| Audit events          | 1 year minimum   | Immutable log storage  |
| Prometheus metrics    | 30 days          | Prometheus TSDB        |

---

## 8. Vulnerability Management

### Container Security

| Control                          | Implementation                               |
|----------------------------------|----------------------------------------------|
| Base image                       | Ubuntu 24.04 LTS (latest security patches)  |
| Non-root execution               | Container runs as `llama` user (UID 1000)    |
| Minimal attack surface           | Runtime image has only: binary, curl, ca-certs|
| No build tools in runtime        | Multi-stage build strips compilers/sources   |
| Image scanning                   | Recommended: Trivy or Snyk in CI pipeline    |
| Read-only model mount            | Model directory mounted `readOnly: true`     |

### Dependency Management

| Component            | Update Strategy                                     |
|----------------------|-----------------------------------------------------|
| llama.cpp            | Pin to specific version tag; update quarterly       |
| Go dependencies      | Dependabot/Renovate for automated PR creation       |
| Frontend (npm)       | Dependabot for automated security updates           |
| Container base image | Rebuild monthly with latest OS patches              |

### Known Risk Areas

| Risk Area                        | Mitigation                                        |
|----------------------------------|---------------------------------------------------|
| Model file integrity             | SHA-256 verification on deployment                |
| Inference server compromise      | Treat as semi-trusted; validate responses         |
| Prompt injection via benchmarks  | Benchmark prompts are operator-controlled data    |
| API key exposure in logs         | API keys never logged; omitted from JSON responses|
| Resource exhaustion (CPU/memory) | Kubernetes resource limits; max_concurrent cap     |

---

## 9. Compliance Framework Mapping

### SOC 2 Type II

| Trust Service Criteria | Control                                        | Implementation                                |
|------------------------|-------------------------------------------------|-----------------------------------------------|
| CC5.1                  | Logical access controls                        | JWT auth + RBAC + RLS                         |
| CC5.2                  | Authentication mechanisms                      | RS256 JWT with configurable expiry            |
| CC6.1                  | Data classification                            | Data elements classified (see Section 2)      |
| CC6.6                  | System boundaries                              | Trust boundaries defined; RLS enforced        |
| CC6.7                  | Data transmission protection                   | TLS 1.2+ on all paths                        |
| CC7.1                  | System monitoring                              | Prometheus metrics + structured logging       |
| CC7.2                  | Anomaly detection                              | Alert rules for server health and latency     |
| CC8.1                  | Change management                              | Version-controlled infrastructure as code     |

### NIST 800-53 (Rev 5)

| Control Family | Control    | Implementation                                       |
|----------------|------------|------------------------------------------------------|
| AC             | AC-3       | Role-based access; RLS tenant isolation              |
| AC             | AC-6       | Least privilege; read-only for non-admin roles       |
| AU             | AU-2       | Audit events for all state-changing operations       |
| AU             | AU-3       | Structured JSON logs with user/tenant/event context  |
| IA             | IA-2       | Multi-factor via platform IAM (not compute-specific) |
| SC             | SC-7       | Network boundary protection; VLAN isolation          |
| SC             | SC-8       | TLS encryption in transit                            |
| SC             | SC-28      | Encryption at rest for all data stores               |
| SI             | SI-4       | Prometheus monitoring; health probes                 |
| CM             | CM-2       | Helm charts define baseline configuration            |

### FedRAMP

| Control                     | Implementation                                         |
|-----------------------------|---------------------------------------------------------|
| Boundary protection (SC-7)  | Air-gapped mode; no external egress                   |
| Data sovereignty            | All data remains in deployment boundary               |
| Encryption (SC-13)          | TLS 1.2+ in transit; volume encryption at rest        |
| Continuous monitoring       | Prometheus metrics with alerting rules                |
| Access control (AC-2)       | JWT-based; tenant-scoped; role-gated                  |

### GDPR

| Article    | Requirement                    | Implementation                                  |
|------------|--------------------------------|-------------------------------------------------|
| Art. 5     | Data minimisation              | No PII in benchmarks; metric aggregation        |
| Art. 25    | Privacy by design              | RLS isolation; encrypted storage                |
| Art. 32    | Security of processing         | Defence-in-depth; TLS; access controls          |
| Art. 44    | Cross-border transfers         | Air-gapped mode prevents any data transfer      |

---

## 10. Third-Party Dependencies

### Open-Source Components

| Component          | License    | Purpose                            | Security Posture        |
|--------------------|------------|-------------------------------------|-------------------------|
| llama.cpp          | MIT        | CPU inference runtime              | Active community; rapid patches |
| PostgreSQL         | PostgreSQL | Data storage                       | Mature; well-audited    |
| Go standard library| BSD-3     | Application runtime                | Google-maintained       |
| chi (router)       | MIT        | HTTP routing                       | Minimal; well-tested   |
| zerolog            | MIT        | Structured logging                 | No external deps        |
| Prometheus client  | Apache 2.0 | Metrics exposition                 | CNCF project           |

### No Proprietary AI Dependencies

| What's NOT Required        | Why It Matters                                     |
|----------------------------|----------------------------------------------------|
| OpenAI API key             | CPU inference is self-contained                    |
| Anthropic API key          | No cloud LLM provider required                    |
| NVIDIA CUDA                | CPU-only inference; no GPU drivers                 |
| Commercial model licenses  | Open-source models (Llama, Mistral, BitNet)       |

---

## 11. Incident Response Considerations

### Compute-Specific Incident Types

| Incident Type                    | Detection                       | Response                                    |
|----------------------------------|---------------------------------|---------------------------------------------|
| Inference server compromise      | Unexpected responses; health check failure | Isolate server; decommission; forensics |
| Model file tampering             | SHA-256 hash mismatch           | Halt inference; re-deploy from trusted source|
| Tenant data leakage              | Audit log analysis              | Revoke tokens; investigate RLS bypass       |
| Resource exhaustion (DoS)        | CPU/memory alerts               | Rate limit; reduce concurrency; scale out   |
| API key exposure                 | Log monitoring; secret scanning | Rotate key; audit access logs               |
| Benchmark data exfiltration      | Network monitoring              | Block egress; investigate data classification|

### Response Priorities

| Priority | Action                                                        |
|----------|---------------------------------------------------------------|
| P1       | Isolate compromised inference servers from the network       |
| P2       | Verify model file integrity across all servers               |
| P3       | Audit RLS enforcement for tenant boundary integrity          |
| P4       | Review and rotate any exposed API keys                       |
| P5       | Preserve logs for forensic analysis                          |

---

*For deployment security in air-gapped environments, see the [Air-Gapped Deployment Guide](04_AIRGAPPED_DEPLOYMENT_GUIDE.md). For technical implementation details, see the [Technical Architecture](../architecture/CPU_INFERENCE_ARCHITECTURE.md).*
