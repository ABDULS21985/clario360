# CPU-Based Inference & BitNet Readiness Architecture

| Attribute       | Value                                                             |
|-----------------|-------------------------------------------------------------------|
| **Document**    | CPU-Based Inference & BitNet Readiness — Technical Architecture   |
| **Version**     | 1.0                                                               |
| **Date**        | 2026-03-13                                                        |
| **Status**      | Phase 1 Complete (Benchmarking Framework)                         |
| **Scope**       | Backend, Frontend, Database, Infrastructure                       |
| **Depends on**  | AI Governance Module, vCISO LLM Subsystem                        |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview](#2-architecture-overview)
3. [Compute Backend Types](#3-compute-backend-types)
4. [Database Schema](#4-database-schema)
5. [Backend Architecture](#5-backend-architecture)
6. [API Reference](#6-api-reference)
7. [Frontend Architecture](#7-frontend-architecture)
8. [Infrastructure & Deployment](#8-infrastructure--deployment)
9. [Cost Comparison Framework](#9-cost-comparison-framework)
10. [Observability & Metrics](#10-observability--metrics)
11. [Testing Coverage](#11-testing-coverage)
12. [Roadmap: Phase 2 & 3](#12-roadmap-phase-2--3)

---

## 1. Executive Summary

### Problem Statement

Clario360's AI subsystem runs 18 production models covering threat scoring, anomaly detection, compliance classification, and agentic vCISO capabilities. All inference currently executes through:

- **Inline Go functions** — rule-based and statistical models embedded in the platform binary
- **Cloud API calls** — OpenAI, Anthropic, and Azure endpoints for LLM workloads

This architecture creates three critical limitations:

| Limitation                | Impact                                                          |
|---------------------------|-----------------------------------------------------------------|
| **GPU vendor lock-in**    | GPU-based inference (vLLM, cloud APIs) costs $2–$8/hr per GPU  |
| **No air-gapped support** | Regulated clients cannot use cloud LLM APIs                    |
| **No cost visibility**    | No infrastructure to measure CPU vs GPU cost-performance        |

### Solution

Phase 1 introduces a **multi-backend compute abstraction** with a complete benchmarking framework:

- **8 compute backend types** covering CPU, GPU, and hybrid inference runtimes
- **Inference server registry** to manage and monitor any OpenAI-compatible endpoint
- **Benchmark runner** with warmup, concurrency control, streaming TTFT measurement, and retry
- **Side-by-side comparison** engine with automated CPU-viability recommendations
- **Cost model framework** for per-backend pricing and monthly savings estimation

### Key Differentiators

1. **BitNet 1-bit readiness** — First-class support for 1.58-bit ternary models (`{-1, 0, 1}` weights) that run on pure CPU via integer addition, eliminating GPU requirements entirely
2. **OpenAI-compatible protocol everywhere** — All backends (vLLM, llama.cpp, BitNet, ONNX) share the same `/v1/chat/completions` endpoint, so the existing 13-phase LLM pipeline works identically across all compute backends
3. **Shadow mode comparison** (Phase 2) — Reuses the existing shadow execution infrastructure to run CPU and GPU versions side-by-side with no new A/B testing code

---

## 2. Architecture Overview

### System Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Clario360 Platform                              │
│                                                                         │
│  ┌──────────────┐    ┌──────────────────┐    ┌───────────────────────┐  │
│  │   Frontend    │    │   API Gateway     │    │   Prometheus/Grafana  │  │
│  │  (Next.js)    │───▶│   (chi router)    │    │   (Metrics)           │  │
│  │              │    │                   │    │                       │  │
│  │ • Compute pg │    │ • JWT Auth        │    │ • ai_benchmark_*      │  │
│  │ • Benchmarks │    │ • Rate Limiting   │    │ • ai_inference_*      │  │
│  │ • Suite detail│    │ • Tenant Routing  │    │ • ai_compute_*        │  │
│  └──────┬───────┘    └────────┬──────────┘    └───────────────────────┘  │
│         │                     │                                          │
│         ▼                     ▼                                          │
│  ┌───────────────────────────────────────────────────────────────────┐   │
│  │                    AI Governance Module                            │   │
│  │                                                                   │   │
│  │  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────┐  │   │
│  │  │BenchmarkHandler │  │BenchmarkService   │  │Benchmark Runner │  │   │
│  │  │                 │──▶│                  │──▶│                 │  │   │
│  │  │ 17 API endpoints│  │ • Orchestration   │  │ • Warmup phase  │  │   │
│  │  │ • Servers CRUD  │  │ • Comparison      │  │ • Measured iter │  │   │
│  │  │ • Suites CRUD   │  │ • Cost estimation │  │ • Concurrency   │  │   │
│  │  │ • Runs          │  │ • Metrics emit    │  │ • SSE streaming │  │   │
│  │  │ • Cost models   │  │                   │  │ • Retry+backoff │  │   │
│  │  └─────────────────┘  └────────┬──────────┘  └────────┬────────┘  │   │
│  │                                │                       │           │   │
│  │  ┌─────────────────┐  ┌───────▼──────────┐            │           │   │
│  │  │InferenceServer  │  │Benchmark         │            │           │   │
│  │  │  Repository     │  │  Repository      │            │           │   │
│  │  │                 │  │                   │            │           │   │
│  │  │ • CRUD + health │  │ • Suites CRUD     │            │           │   │
│  │  │ • Stats queries │  │ • Runs CRUD       │            │           │   │
│  │  │ • RLS isolation │  │ • Cost models     │            │           │   │
│  │  └────────┬────────┘  └────────┬──────────┘            │           │   │
│  │           │                    │                        │           │   │
│  └───────────┼────────────────────┼────────────────────────┼───────────┘   │
│              ▼                    ▼                        ▼               │
│  ┌────────────────────────────────────────┐  ┌─────────────────────────┐  │
│  │         PostgreSQL (RLS)               │  │  Inference Servers      │  │
│  │                                        │  │                         │  │
│  │  • ai_inference_servers                │  │  ┌───────────────────┐  │  │
│  │  • ai_benchmark_suites                 │  │  │ llama.cpp (CPU)   │  │  │
│  │  • ai_benchmark_runs                   │  │  │ /v1/chat/complete │  │  │
│  │  • ai_compute_cost_models              │  │  └───────────────────┘  │  │
│  │                                        │  │  ┌───────────────────┐  │  │
│  │  Row-Level Security:                   │  │  │ BitNet (CPU)      │  │  │
│  │  tenant_id = app.current_tenant_id     │  │  │ /v1/chat/complete │  │  │
│  └────────────────────────────────────────┘  │  └───────────────────┘  │  │
│                                              │  ┌───────────────────┐  │  │
│                                              │  │ vLLM (GPU)        │  │  │
│                                              │  │ /v1/chat/complete │  │  │
│                                              │  └───────────────────┘  │  │
│                                              └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Benchmark Execution Flow

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│  Client   │────▶│   Handler    │────▶│    Service       │────▶│   Runner     │
│  (POST    │     │  RunBenchmark│     │  RunBenchmark()  │     │  Execute()   │
│   /run)   │     └──────────────┘     │                  │     │              │
└──────────┘                           │ 1. Load suite    │     │ 1. Warmup    │
                                       │ 2. Load server   │     │    phase     │
                                       │ 3. Create run    │     │ 2. Measured  │
                                       │    (status:      │     │    iterations│
                                       │     running)     │     │ 3. Aggregate │
                                       │ 4. Execute ──────┼────▶│    results   │
                                       │ 5. Store results │     │              │
                                       │ 6. Emit metrics  │     │ Returns:     │
                                       └──────────────────┘     │ p50/p95/p99  │
                                                                │ tokens/sec   │
                                              ┌─────────────────│ TTFT metrics │
                                              ▼                 │ fail/retry   │
                                       ┌──────────────┐        └──────────────┘
                                       │  Inference    │               │
                                       │  Server       │               │
                                       │              │◀──────────────┘
                                       │ POST /v1/chat│   N concurrent
                                       │ /completions │   requests with
                                       └──────────────┘   semaphore
```

### Multi-Tenant Isolation

All four database tables enforce Row-Level Security (RLS) using PostgreSQL's `current_setting('app.current_tenant_id')`. Each API request sets the tenant context via the JWT-extracted `tenant_id`, ensuring complete data isolation between organisations.

---

## 3. Compute Backend Types

The platform supports **8 compute backend types**, covering the full spectrum from embedded Go functions to GPU-accelerated inference:

| Backend Type    | Runtime         | Hardware | Protocol                    | Use Case                              |
|-----------------|-----------------|----------|-----------------------------|---------------------------------------|
| `inline_go`     | Go binary       | CPU      | In-process function call    | Rule-based & statistical models       |
| `vllm_gpu`      | vLLM            | GPU      | `/v1/chat/completions`      | High-throughput production LLM        |
| `vllm_cpu`      | vLLM (CPU mode) | CPU      | `/v1/chat/completions`      | CPU fallback for vLLM models          |
| `llamacpp_cpu`  | llama.cpp       | CPU      | `/v1/chat/completions`      | Quantised GGUF models on CPU          |
| `llamacpp_gpu`  | llama.cpp       | GPU      | `/v1/chat/completions`      | GGUF models with GPU acceleration     |
| `bitnet_cpu`    | llama.cpp       | CPU      | `/v1/chat/completions`      | 1-bit ternary models (BitNet b1.58)   |
| `onnx_cpu`      | ONNX Runtime    | CPU      | `/v1/chat/completions`      | Optimised ONNX models on CPU          |
| `onnx_gpu`      | ONNX Runtime    | GPU      | `/v1/chat/completions`      | ONNX models with CUDA/TensorRT        |

### OpenAI-Compatible Protocol

All non-inline backends communicate via the **OpenAI Chat Completions API**:

```
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer <optional-api-key>

{
  "model": "llama-3.1-8b-instruct-q4_0",
  "messages": [
    {"role": "system", "content": "You are a security analyst."},
    {"role": "user", "content": "Analyze this network flow..."}
  ],
  "max_tokens": 1024,
  "temperature": 0.1,
  "stream": true,
  "stream_options": {"include_usage": true}
}
```

This protocol is natively supported by **vLLM**, **llama.cpp server**, and **BitNet** (served via llama.cpp), meaning the existing LLM pipeline (tools, grounding, PII filtering, audit logging) works identically across all backends with zero modification.

### LLM Provider Implementations

Two new providers implement the existing `LLMProvider` interface:

**LlamaCppProvider** (`backend/internal/cyber/vciso/llm/provider/llamacpp_provider.go`):
- Wraps an OpenAI-compatible llama.cpp server
- Embeds `OpenAIProvider` (code reuse — identical HTTP protocol)
- Cost: ~$0.00000025/prompt token + $0.000001/completion token (~10x cheaper than GPU)
- Max context: 32,768 tokens
- Default timeout: 60s

**BitNetProvider** (`backend/internal/cyber/vciso/llm/provider/bitnet_provider.go`):
- 1.58-bit quantisation: `{-1, 0, 1}` ternary weights
- Pure CPU inference via integer addition — no matrix multiplication
- Cost: ~$0.0000014/token (combined prompt+completion)
- Max context: 4,096 tokens (current BitNet model limitation)
- Default timeout: 120s (slower inference, longer timeout)

---

## 4. Database Schema

**Migration**: `backend/migrations/platform_core/000010_compute_benchmark_schema.up.sql`

### Entity-Relationship Diagram

```
┌──────────────────────┐
│     tenants           │
│  (id UUID PK)         │
└──────────┬────────────┘
           │ 1:N
     ┌─────┼──────────────────────────────────┐
     │     │                                   │
     ▼     ▼                                   ▼
┌─────────────────────┐  ┌────────────────────────────┐  ┌──────────────────────┐
│ ai_inference_servers │  │ ai_benchmark_suites        │  │ ai_compute_cost_     │
│                      │  │                            │  │ models               │
│ id          UUID PK  │  │ id             UUID PK     │  │                      │
│ tenant_id   UUID FK  │  │ tenant_id      UUID FK     │  │ id         UUID PK   │
│ name        TEXT     │  │ name           TEXT         │  │ tenant_id  UUID FK   │
│ backend_type TEXT    │  │ model_slug     TEXT         │  │ name       TEXT      │
│ base_url    TEXT     │  │ prompt_dataset JSONB        │  │ backend_type TEXT    │
│ health_ep   TEXT     │  │ warmup_count   INT          │  │ instance_type TEXT   │
│ model_name  TEXT     │  │ iteration_count INT         │  │ hourly_cost FLOAT   │
│ status      TEXT     │  │ concurrency    INT          │  │ cpu_cores  INT      │
│ cpu_cores   INT      │  │ timeout_seconds INT         │  │ memory_gb  INT      │
│ memory_mb   INT      │  │ stream_enabled BOOL         │  │ gpu_type   TEXT     │
│ gpu_type    TEXT     │  │ max_retries    INT          │  │ gpu_count  INT      │
│ gpu_count   INT      │  │ created_by     UUID         │  │ max_tok/s  FLOAT    │
│ max_conc    INT      │  │ created_at     TIMESTAMPTZ  │  │ created_at TSTAMPTZ │
│ stream_cap  BOOL     │  └──────────┬─────────────────┘  └──────────────────────┘
│ metadata    JSONB    │             │ 1:N
│ created_at  TSTAMPTZ │             │
│ updated_at  TSTAMPTZ │             ▼
└──────────┬───────────┘  ┌──────────────────────────────────┐
           │              │ ai_benchmark_runs                 │
           │   FK         │                                   │
           └──────────────│ id                 UUID PK        │
                          │ tenant_id          UUID FK        │
                          │ suite_id           UUID FK ───────┘
                          │ server_id          UUID FK ───────┐
                          │ backend_type       TEXT            │
                          │ model_name         TEXT            │
                          │ status             TEXT            │
                          │                                   │
                          │ ── Latency (6 fields) ──          │
                          │ p50/p95/p99/avg/min/max_latency   │
                          │                                   │
                          │ ── Throughput (6 fields) ──       │
                          │ tokens_per_sec, requests_per_sec  │
                          │ total_tokens, total_requests      │
                          │ failed_requests, retried_requests │
                          │                                   │
                          │ ── TTFT (3 fields, stream only) ──│
                          │ p50/p95/avg_ttft_ms               │
                          │                                   │
                          │ ── Quality (5 fields) ──          │
                          │ avg_perplexity, bleu_score         │
                          │ rouge_l, semantic_sim, factual_acc │
                          │                                   │
                          │ ── Resource (4 fields) ──         │
                          │ peak/avg_cpu_percent               │
                          │ peak/avg_memory_mb                 │
                          │                                   │
                          │ ── Cost (2 fields) ──             │
                          │ estimated_hourly_cost_usd          │
                          │ cost_per_1k_tokens_usd             │
                          │                                   │
                          │ ── Lifecycle (5 fields) ──        │
                          │ started_at, completed_at           │
                          │ duration_seconds, error_message    │
                          │ raw_results (JSONB)                │
                          └──────────────────────────────────┘
```

### Indexes

| Table                  | Index                                     | Purpose                          |
|------------------------|-------------------------------------------|----------------------------------|
| `ai_inference_servers` | `(tenant_id, name) WHERE status<>'decommissioned'` | Unique active server names |
| `ai_inference_servers` | `(tenant_id, status)`                     | Filter by status                 |
| `ai_benchmark_suites`  | `(tenant_id, created_at DESC)`            | List suites by recency           |
| `ai_benchmark_runs`    | `(suite_id, created_at DESC)`             | List runs for a suite            |
| `ai_benchmark_runs`    | `(server_id, created_at DESC)`            | List runs for a server           |
| `ai_benchmark_runs`    | `(tenant_id, status, created_at DESC)`    | Filter runs by status            |
| `ai_compute_cost_models` | `(tenant_id, backend_type)`             | Lookup cost by backend           |

### Artifact Type Extension

The `ai_model_versions` table's `artifact_type` constraint was extended to support new model formats:

```sql
CHECK (artifact_type IN (
    'go_function', 'rule_set', 'statistical_config', 'template_config',
    'serialized_model', 'gguf_model', 'bitnet_model', 'onnx_model'
))
```

---

## 5. Backend Architecture

### Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Handler Layer                         │
│  benchmark_handler.go — HTTP request/response handling       │
│  17 endpoints mapped in routes.go                            │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                       DTO Layer                              │
│  benchmark_dto.go — Request/response structs                 │
│  Strict JSON decoding (unknown fields rejected)              │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                      Service Layer                           │
│  benchmark_service.go — Business logic & orchestration       │
│  • Server CRUD with validation                               │
│  • Suite CRUD with prompt dataset parsing                     │
│  • Run execution (delegates to Runner)                       │
│  • Run comparison with automated recommendations             │
│  • Cost estimation (CPU vs GPU monthly savings)              │
└────────────────────────┬────────────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ InferenceSvr │ │ Benchmark    │ │ Benchmark    │
│ Repository   │ │ Repository   │ │ Runner       │
│              │ │              │ │              │
│ PostgreSQL   │ │ PostgreSQL   │ │ HTTP Client  │
│ + RLS        │ │ + RLS        │ │ OpenAI API   │
└──────────────┘ └──────────────┘ └──────────────┘
```

### Model Layer

**File**: `backend/internal/aigovernance/model/compute.go`

| Struct                | Fields | Purpose                                     |
|-----------------------|--------|---------------------------------------------|
| `InferenceServer`     | 18     | Registered inference endpoint with hardware specs |
| `BenchmarkSuite`      | 14     | Reusable test configuration with prompts    |
| `BenchmarkRun`        | 41     | Complete results from one benchmark execution |
| `ComputeCostModel`    | 12     | Pricing data per backend/instance type      |
| `BenchmarkComparison` | 6      | Side-by-side analysis with recommendation   |

### Benchmark Runner

**File**: `backend/internal/aigovernance/benchmark/runner.go`

The runner is a stateless HTTP client that executes benchmarks against any OpenAI-compatible server:

```
Execute(ctx, RunConfig) → *AggregatedResults

1. Warmup Phase
   └─ Run warmup_count iterations (results discarded)
   └─ Purpose: warm JIT, page caches, model loading

2. Measured Phase
   └─ Launch iteration_count goroutines
   └─ Semaphore limits to concurrency concurrent requests
   └─ Each iteration:
       ├─ Select prompt (round-robin from dataset)
       ├─ callWithRetry(prompt)
       │   ├─ If stream=false: POST → parse JSON response
       │   └─ If stream=true:  POST → parse SSE stream → measure TTFT
       ├─ On transient error: exponential backoff (500ms → 10s cap)
       └─ Record IterationResult {latency, tokens, content, ttft, retries}

3. Aggregation
   └─ Sort latencies → compute p50/p95/p99
   └─ Sort TTFTs → compute p50/p95/avg (streaming only)
   └─ Calculate tokens/sec, requests/sec
   └─ Count failed/retried requests
```

**Retry configuration**:

| Parameter      | Default    | Description                                  |
|----------------|------------|----------------------------------------------|
| MaxRetries     | 3          | Maximum retry attempts per request            |
| InitialDelay   | 500ms      | Delay before first retry                      |
| MaxDelay       | 10s        | Upper bound on backoff delay                  |
| BackoffFactor  | 2.0        | Exponential multiplier                        |
| RetryableHTTP  | 429, 500, 502, 503, 504 | HTTP codes that trigger retry   |

**Streaming SSE support**:

When `stream=true`, the runner parses `data: {...}` SSE frames and records:
- **TTFT** (Time to First Token): duration from request start to first content-bearing chunk
- Token usage from the final chunk's `usage` field (OpenAI-compatible `stream_options`)
- Full response content reconstructed from delta chunks

### Comparison Engine

The service layer's `CompareRuns()` method:

1. Loads multiple completed benchmark runs by ID
2. Computes deltas between the first two runs:
   - `CostDeltaMonthlyUSD` = hourly cost difference × 730 hours
   - `LatencyDeltaPct` = p95 latency difference as percentage
   - `QualityDeltaPct` = semantic similarity difference as percentage
3. Generates automated recommendation:

| Recommendation    | Condition                                          |
|-------------------|----------------------------------------------------|
| `cpu_viable`      | CPU latency < 3× GPU AND quality drop < 10%       |
| `gpu_required`    | CPU latency ≥ 3× GPU                               |
| `needs_more_data` | Results inconclusive                                |

---

## 6. API Reference

**Base path**: `/api/ai`

### Inference Servers (6 endpoints)

| Method   | Path                              | Handler              | Description                    |
|----------|-----------------------------------|----------------------|--------------------------------|
| `POST`   | `/inference-servers`              | `CreateServer`       | Register new inference server  |
| `GET`    | `/inference-servers`              | `ListServers`        | List servers (paginated)       |
| `GET`    | `/inference-servers/{serverId}`   | `GetServer`          | Get server by ID               |
| `PUT`    | `/inference-servers/{serverId}`   | `UpdateServer`       | Update server configuration    |
| `PUT`    | `/inference-servers/{serverId}/status` | `UpdateServerStatus` | Change server status      |
| `DELETE` | `/inference-servers/{serverId}`   | `DeleteServer`       | Decommission server            |

### Benchmark Suites (5 endpoints)

| Method   | Path                                   | Handler          | Description                    |
|----------|----------------------------------------|------------------|--------------------------------|
| `POST`   | `/benchmarks/suites`                   | `CreateSuite`    | Create benchmark suite         |
| `GET`    | `/benchmarks/suites`                   | `ListSuites`     | List suites (paginated)        |
| `GET`    | `/benchmarks/suites/{suiteId}`         | `GetSuite`       | Get suite by ID                |
| `PUT`    | `/benchmarks/suites/{suiteId}`         | `UpdateSuite`    | Update suite configuration     |
| `POST`   | `/benchmarks/suites/{suiteId}/run`     | `RunBenchmark`   | Execute benchmark on a server  |

### Benchmark Runs (3 endpoints)

| Method   | Path                                   | Handler          | Description                    |
|----------|----------------------------------------|------------------|--------------------------------|
| `GET`    | `/benchmarks/runs`                     | `ListRuns`       | List runs (paginated, filterable by suite_id) |
| `GET`    | `/benchmarks/runs/{runId}`             | `GetRun`         | Get run with full results      |
| `POST`   | `/benchmarks/runs/compare`             | `CompareRuns`    | Compare 2–10 runs side-by-side |

### Cost Models (3 endpoints)

| Method   | Path                                   | Handler              | Description                     |
|----------|----------------------------------------|----------------------|---------------------------------|
| `POST`   | `/compute-costs`                       | `CreateCostModel`    | Define cost model for backend   |
| `GET`    | `/compute-costs`                       | `ListCostModels`     | List all cost models            |
| `POST`   | `/compute-costs/estimate`              | `EstimateCostSavings`| Compare CPU vs GPU monthly cost |

### Example: Run Benchmark

```bash
# Execute benchmark suite against a llama.cpp CPU server
curl -X POST http://localhost:8080/api/ai/benchmarks/suites/{suiteId}/run \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"server_id": "abc-123"}'

# Response (202 Accepted):
{
  "data": {
    "id": "run-456",
    "suite_id": "suite-789",
    "server_id": "abc-123",
    "backend_type": "llamacpp_cpu",
    "status": "completed",
    "p50_latency_ms": 245.3,
    "p95_latency_ms": 512.7,
    "p99_latency_ms": 890.1,
    "tokens_per_second": 42.5,
    "total_tokens": 12750,
    "total_requests": 100,
    "failed_requests": 0,
    "duration_seconds": 300,
    "estimated_hourly_cost_usd": 0.12
  }
}
```

### Example: Compare CPU vs GPU Runs

```bash
curl -X POST http://localhost:8080/api/ai/benchmarks/runs/compare \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"run_ids": ["cpu-run-id", "gpu-run-id"]}'

# Response:
{
  "data": {
    "runs": [...],
    "cost_delta_monthly_usd": -2190.00,
    "latency_delta_percent": 45.2,
    "quality_delta_percent": -2.1,
    "recommendation": "cpu_viable",
    "recommendation_reason": "CPU latency is within 3x of GPU with acceptable quality."
  }
}
```

---

## 7. Frontend Architecture

### Page Structure

```
/admin/ai-governance/
├── compute/              ← Inference Server Management
│   └── page.tsx
├── benchmarks/           ← Benchmark Dashboard
│   ├── page.tsx          ← Suites tab + Run Results tab
│   └── [suiteId]/
│       └── page.tsx      ← Suite Detail with run history
```

### Compute Infrastructure Page

**File**: `frontend/src/app/(dashboard)/admin/ai-governance/compute/page.tsx`

```
┌─────────────────────────────────────────────────────────┐
│  Compute Infrastructure                    [Add Server] │
│  Manage inference servers for CPU and GPU   [Refresh]   │
├──────────────┬──────────┬──────────┬───────────────────┤
│ Total Servers│ Healthy  │CPU Back- │ GPU Backends      │
│     5        │    3     │  ends 3  │     2             │
├──────────────┴──────────┴──────────┴───────────────────┤
│                                                         │
│  Inference Servers                                      │
│  ┌──────────────────┬──────────┬─────────────┬────────┐│
│  │ Name             │ Backend  │ Model       │ Status ││
│  ├──────────────────┼──────────┼─────────────┼────────┤│
│  │ llamacpp-cpu-01  │llama.cpp │ llama-3.1   │healthy ││
│  │ localhost:8081/v1│  CPU     │ Q4_0        │        ││
│  ├──────────────────┼──────────┼─────────────┼────────┤│
│  │ vllm-gpu-prod    │vLLM GPU  │ mistral-7b  │healthy ││
│  │ gpu-01:8000/v1   │          │             │        ││
│  └──────────────────┴──────────┴─────────────┴────────┘│
└─────────────────────────────────────────────────────────┘
```

**Features**:
- 4 KPI cards: Total Servers, Healthy, CPU Backends, GPU Backends
- Data table with columns: Name (+ URL), Backend type, Model (+ quantization), Status badge, Resources, Actions
- Register Server dialog: name, backend type selector (8 options), base URL, health endpoint, model name, quantization, CPU/memory/GPU specs
- Decommission confirmation dialog with destructive action
- Refresh button reloads both table and KPI data

### Benchmarks Page

**File**: `frontend/src/app/(dashboard)/admin/ai-governance/benchmarks/page.tsx`

```
┌─────────────────────────────────────────────────────────┐
│  Inference Benchmarks                    [New Suite]     │
│  Measure and compare CPU vs GPU inference latency       │
├──────────────┬──────────┬──────────┬───────────────────┤
│ Benchmark    │Total Runs│Completed │ Avg Latency       │
│ Suites: 3    │   12     │    10    │   312ms           │
├──────────────┴──────────┴──────────┴───────────────────┤
│  [Suites] [Run Results]                                 │
│                                                         │
│  ┌──────────────────────┬────────┬──────┬──────┬──────┐│
│  │ Suite Name           │ Model  │Config│ Runs │      ││
│  ├──────────────────────┼────────┼──────┼──────┼──────┤│
│  │ threat-scorer-cpu    │threat- │5 warm│  4   │[Run] ││
│  │ -bench               │scorer  │100it │      │      ││
│  │                      │        │1 conc│      │      ││
│  └──────────────────────┴────────┴──────┴──────┴──────┘│
└─────────────────────────────────────────────────────────┘
```

**Features**:
- Tab-based layout: Suites (default) / Run Results
- Suites tab: clickable suite names navigate to detail page, per-row Run button
- Run Results tab: all runs across suites with latency/throughput columns
- Create Suite dialog: name, description, model slug, warmup, iterations, concurrency, timeout
- Run Benchmark dialog: select target server from registered servers

### Suite Detail Page

**File**: `frontend/src/app/(dashboard)/admin/ai-governance/benchmarks/[suiteId]/page.tsx`

**Features**:
- Suite configuration summary at top
- Run history table with status badges, latency metrics, throughput
- Multi-select checkboxes for run comparison
- "Compare Selected" button triggers side-by-side analysis
- Comparison results panel showing: cost delta, latency delta, quality delta, recommendation

### TypeScript Types

**File**: `frontend/src/types/ai-governance.ts` (lines 420–543)

All compute types are defined alongside the existing AI governance types:
- `ComputeBackendType` — 8-variant union type
- `InferenceServerStatus` — 5-variant union type
- `BenchmarkRunStatus` — 5-variant union type
- `AIInferenceServer` — 18 fields
- `AIBenchmarkSuite` — 13 fields
- `AIBenchmarkRun` — 36 fields
- `AIBenchmarkComparison` — 6 fields with typed recommendation
- `AIComputeCostModel` — 12 fields

---

## 8. Infrastructure & Deployment

### Docker: llama.cpp Server

**File**: `deploy/docker/Dockerfile.llamacpp`

Multi-stage build optimised for CPU inference:

```
Stage 1: Builder (ubuntu:24.04)
├── Install build-essential, cmake, git
├── Clone llama.cpp (configurable version via LLAMACPP_VERSION arg)
└── Build with CPU optimisations:
    ├── AVX2 = ON    (256-bit SIMD)
    ├── F16C = ON    (FP16 conversion)
    ├── FMA  = ON    (fused multiply-add)
    └── NATIVE = OFF (portable binary)

Stage 2: Runtime (ubuntu:24.04)
├── Copy llama-server binary
├── Non-root user (llama:llama)
├── Health check: curl /health every 15s
└── Default config:
    ├── CTX_SIZE = 4096
    ├── THREADS = 4
    ├── PARALLEL = 4
    └── BATCH_SIZE = 512
```

**BitNet support**: The same llama.cpp binary serves both standard quantised models (Q4_0, Q5_K_M, etc.) and BitNet 1-bit models. The only difference is the model file — BitNet GGUF files use ternary weights internally.

### Kubernetes: Helm Deployment

**Files**:
- `deploy/helm/clario360/templates/inference/llamacpp-deployment.yaml`
- `deploy/helm/clario360/templates/inference/llamacpp-service.yaml`

```yaml
# Conditionally deployed when inference.llamacpp.enabled = true
# values.yaml configuration:
inference:
  llamacpp:
    enabled: true
    replicaCount: 2
    image:
      repository: clario360-llamacpp
      tag: latest
    contextSize: 4096
    threads: 8
    parallel: 4
    batchSize: 512
    model:
      downloadUrl: "https://models.internal/llama-3.1-8b-Q4_0.gguf"
    persistence:
      enabled: true
      sizeLimit: "10Gi"
    resources:
      requests:
        cpu: "4"
        memory: "8Gi"
      limits:
        cpu: "8"
        memory: "16Gi"
    nodeSelector:
      inference/cpu: "true"
```

**Deployment features**:
- **Init container**: Downloads GGUF model from URL or S3 on first deploy (skips if model already exists)
- **Rolling updates**: maxSurge=1, maxUnavailable=0 for zero-downtime deploys
- **Health probes**: Liveness (30s initial, 15s interval) and Readiness (10s initial, 10s interval)
- **Node selector**: Schedule on nodes labelled for inference workloads
- **Persistent volume**: Optional PVC for model storage across pod restarts
- **Non-root**: `runAsNonRoot: true`, `fsGroup: 1000`

### Air-Gapped Deployment

BitNet models are particularly suitable for air-gapped environments:

| Property            | BitNet Advantage                                   |
|---------------------|----------------------------------------------------|
| Model size          | ~1/16 of FP16 (1-bit vs 16-bit weights)           |
| Hardware            | Commodity CPU only — no GPU, no CUDA               |
| External deps       | Zero — no cloud API calls, no internet access       |
| Pre-loading         | GGUF file loaded from internal registry at deploy   |

---

## 9. Cost Comparison Framework

### Cost Model Structure

Each `ComputeCostModel` defines:

| Field              | Type    | Example (CPU)        | Example (GPU)           |
|--------------------|---------|----------------------|-------------------------|
| `name`             | string  | "CPU-8c-32GB"        | "A100-1xGPU"            |
| `backend_type`     | string  | "llamacpp_cpu"       | "vllm_gpu"              |
| `instance_type`    | string  | "c6i.2xlarge"        | "p4d.24xlarge"          |
| `hourly_cost_usd`  | float   | 0.34                 | 3.40                    |
| `cpu_cores`        | int     | 8                    | 96                      |
| `memory_gb`        | int     | 32                   | 1152                    |
| `gpu_type`         | string  | —                    | "A100"                  |
| `gpu_count`        | int     | 0                    | 8                       |
| `max_tokens/sec`   | float   | 45.0                 | 450.0                   |

### Savings Estimation Algorithm

The `EstimateCostSavings` endpoint compares a CPU run against a GPU run:

```
Input: cpu_run_id, gpu_run_id
Output:
  cpu_monthly_cost     = cpu_hourly × 730 hours
  gpu_monthly_cost     = gpu_hourly × 730 hours
  monthly_savings      = gpu_monthly - cpu_monthly
  savings_percent      = (gpu_hourly - cpu_hourly) / gpu_hourly × 100
  latency_increase_pct = (cpu_p95 - gpu_p95) / gpu_p95 × 100
  cpu_tokens_per_sec   (from run results)
  gpu_tokens_per_sec   (from run results)
```

**Example**: A typical CPU-only deployment saves **$2,200+/month** compared to GPU with a 2–3x latency increase that is acceptable for non-real-time workloads like batch threat analysis, compliance scanning, and report generation.

---

## 10. Observability & Metrics

**File**: `backend/internal/aigovernance/service/metrics.go`

### Prometheus Metrics

| Metric Name                    | Type      | Labels                       | Purpose                              |
|--------------------------------|-----------|------------------------------|--------------------------------------|
| `ai_benchmark_runs_total`      | Counter   | `backend_type`, `status`     | Track benchmark executions           |
| `ai_benchmark_latency_seconds` | Histogram | `backend_type`               | Latency distribution per backend     |
| `ai_inference_server_health`   | Gauge     | `server_name`, `backend_type`| Server health status (1/0)           |
| `ai_compute_cost_per_token`    | Gauge     | `backend_type`               | Estimated cost per 1K tokens         |

### Alerting Rules

Recommended Prometheus alerting rules (Phase 3):

```yaml
- alert: InferenceServerDown
  expr: ai_inference_server_health == 0
  for: 5m
  labels:
    severity: critical

- alert: CPULatencyHigh
  expr: histogram_quantile(0.95, ai_benchmark_latency_seconds{backend_type=~".*cpu.*"}) > 5
  for: 10m
  labels:
    severity: warning

- alert: FallbackRateHigh
  expr: rate(ai_inference_fallback_total[5m]) / rate(ai_inference_requests_total[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
```

---

## 11. Testing Coverage

### Playwright E2E Tests

**23 total tests** across 2 spec files:

#### Compute Tests (`frontend/e2e/compute.spec.ts`) — 9 tests

| Test                               | Coverage                                           |
|------------------------------------|----------------------------------------------------|
| Renders page header and description| Page title and description text                    |
| Displays KPI cards                 | Total Servers, Healthy, CPU Backends, GPU Backends |
| Shows existing servers             | Seeded data visible in table                       |
| Shows server details in row        | Backend type badge, model name, status badge       |
| Opens Register Server dialog       | Dialog opens with all form fields                  |
| Fills and submits form             | Full form submission, server appears in table      |
| Decommissions a server             | Delete confirmation, dialog closes                 |
| Refresh button reloads data        | Data reloads on click                              |
| Cancel closes dialog               | Form data not persisted after cancel               |

#### Benchmark Tests (`frontend/e2e/benchmarks.spec.ts`) — 14 tests

| Test                                | Coverage                                          |
|-------------------------------------|---------------------------------------------------|
| Renders page header and description | Page title and description text                   |
| Displays KPI cards                  | Benchmark Suites, Total Runs, Completed, Avg Latency |
| Shows Suites tab with data          | Seeded suite visible                              |
| Shows suite configuration           | Model slug badge, iteration config in row         |
| Switches to Run Results tab         | Tab navigation, heading visible                   |
| Shows run data in Run Results       | Backend type visible in results                   |
| Opens Create Suite dialog           | Dialog with form fields                           |
| Fills and submits Create Suite      | Full form, dialog closes, suite appears           |
| Opens Run Benchmark dialog          | Dialog from suite row action button               |
| Cancel closes Create Suite dialog   | Form data not persisted                           |
| Navigates to suite detail page      | Click suite name → URL changes to UUID path       |
| Loads suite detail page             | Suite name visible on detail                      |
| Shows benchmark runs on detail page | Run data visible on suite detail                  |

### Test Infrastructure

- **Authentication**: Global setup authenticates via login form, saves storage state to `e2e/.auth/user.json`
- **Unique identifiers**: Timestamp-based names (`pw-suite-${Date.now()}`) prevent test pollution
- **Strict mode safe**: Uses `.first()`, `{ exact: true }`, and row-scoped locators to avoid strict mode violations

---

## 12. Roadmap: Phase 2 & 3

### Phase 2: Shadow Mode Compute Comparison

**Goal**: Deploy CPU inference as shadow version alongside GPU production using existing shadow infrastructure.

| Feature                        | Approach                                             |
|--------------------------------|------------------------------------------------------|
| Compute config on versions     | Add `compute_backend`, `inference_server_id`, `compute_config` to `ai_model_versions` |
| Inference Router               | New provider that routes CPU→GPU with latency-based fallback |
| Shadow comparison              | Reuse existing `ShadowExecutor` + `Comparator` — deploy CPU version in shadow status |
| Frontend                       | Compute config dialog, compute comparison panel, backend badges on model cards |

### Phase 3: Production Routing & Cost Tracking

**Goal**: Route production traffic based on policies; full cost dashboard.

| Feature                        | Approach                                             |
|--------------------------------|------------------------------------------------------|
| Routing policies               | Rule-based routing (short queries→CPU, complex→GPU, off-peak→CPU) |
| Enhanced cost tracking         | Real per-backend cost models, cumulative savings tracking |
| Cost dashboard                 | Savings time series, compute distribution pie chart   |
| Monitoring & alerting          | Inference server down, CPU latency high, fallback rate high alerts |
| Air-gapped support             | BitNet pre-loaded from internal registry, zero external deps |

---

## File Reference

### Backend — New Files

| File | Purpose |
|------|---------|
| `backend/migrations/platform_core/000010_compute_benchmark_schema.up.sql` | Database schema (4 tables + artifact type extension) |
| `backend/internal/aigovernance/model/compute.go` | Model structs and constants |
| `backend/internal/aigovernance/repository/inference_server_repo.go` | Inference server CRUD + stats |
| `backend/internal/aigovernance/repository/benchmark_repo.go` | Benchmark suite/run/cost CRUD |
| `backend/internal/aigovernance/service/benchmark_service.go` | Business logic and orchestration |
| `backend/internal/aigovernance/benchmark/runner.go` | Benchmark execution engine |
| `backend/internal/aigovernance/handler/benchmark_handler.go` | HTTP handlers (17 endpoints) |
| `backend/internal/aigovernance/dto/benchmark_dto.go` | Request/response DTOs |
| `backend/internal/cyber/vciso/llm/provider/llamacpp_provider.go` | llama.cpp LLM provider |
| `backend/internal/cyber/vciso/llm/provider/bitnet_provider.go` | BitNet LLM provider |

### Backend — Modified Files

| File | Changes |
|------|---------|
| `backend/internal/aigovernance/handler/routes.go` | Added compute/benchmark route group |
| `backend/internal/aigovernance/service/metrics.go` | Added 4 benchmark/inference metrics |

### Frontend — Modified Files

| File | Changes |
|------|---------|
| `frontend/src/types/ai-governance.ts` | Added compute types (lines 420–543) |
| `frontend/src/lib/api/enterprise-api.ts` | Added 15 API methods |

### Frontend — New Files

| File | Purpose |
|------|---------|
| `frontend/src/app/(dashboard)/admin/ai-governance/compute/page.tsx` | Compute infrastructure page |
| `frontend/src/app/(dashboard)/admin/ai-governance/benchmarks/page.tsx` | Benchmarks dashboard |
| `frontend/src/app/(dashboard)/admin/ai-governance/benchmarks/[suiteId]/page.tsx` | Suite detail page |

### Infrastructure — New Files

| File | Purpose |
|------|---------|
| `deploy/docker/Dockerfile.llamacpp` | Multi-stage CPU-optimised llama.cpp build |
| `deploy/helm/clario360/templates/inference/llamacpp-deployment.yaml` | K8s deployment with model init container |
| `deploy/helm/clario360/templates/inference/llamacpp-service.yaml` | K8s ClusterIP service |

### Tests — New Files

| File | Purpose |
|------|---------|
| `frontend/e2e/compute.spec.ts` | 9 Playwright E2E tests for compute page |
| `frontend/e2e/benchmarks.spec.ts` | 14 Playwright E2E tests for benchmarks |
| `frontend/playwright.config.ts` | Playwright configuration with auth setup |
| `frontend/e2e/global-setup.ts` | Authentication via login form |
