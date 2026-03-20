# CPU-Based Inference & BitNet Readiness — Solutions Design

| Attribute        | Value                                                          |
|------------------|----------------------------------------------------------------|
| **Document**     | Solutions Design — CPU Inference & BitNet Readiness            |
| **Version**      | 2.0                                                            |
| **Date**         | 2026-03-14                                                     |
| **Status**       | Phase 1 Delivered · Phases 2–3 Planned                         |
| **Audience**     | Client Stakeholders, Security Architects, Product Owners       |
| **Related**      | [Technical Architecture](CPU_INFERENCE_ARCHITECTURE.md)        |

---

## Table of Contents

1. [Business Context & Client Requirements](#1-business-context--client-requirements)
2. [Solution Overview](#2-solution-overview)
3. [High-Level Architecture](#3-high-level-architecture)
4. [Compute Backend Strategy](#4-compute-backend-strategy)
5. [Benchmarking Framework](#5-benchmarking-framework)
6. [Cost-Benefit Analysis Framework](#6-cost-benefit-analysis-framework)
7. [Data Model Design](#7-data-model-design)
8. [Frontend Experience](#8-frontend-experience)
9. [Deployment Architecture](#9-deployment-architecture)
10. [Security & Multi-Tenancy](#10-security--multi-tenancy)
11. [Observability Strategy](#11-observability-strategy)
12. [Quality Assurance](#12-quality-assurance)
13. [Phased Delivery Roadmap](#13-phased-delivery-roadmap)
14. [Risk Assessment & Mitigations](#14-risk-assessment--mitigations)
15. [Decision Log](#15-decision-log)

---

## 1. Business Context & Client Requirements

### The Challenge

Clario360 operates 18 AI models across six product suites — threat scoring, anomaly detection, compliance classification, natural language security analysis, and agentic vCISO reasoning. Today, all inference falls into two categories:

| Current Model           | How It Works                         | Limitation                              |
|-------------------------|--------------------------------------|-----------------------------------------|
| Inline Go functions     | Logic embedded in the platform binary | Cannot run external LLMs                |
| Cloud API calls         | OpenAI, Anthropic, Azure endpoints   | Expensive, requires internet, no control|

This creates a dependency on GPU cloud infrastructure and external AI providers — a growing concern for enterprise and regulated clients.

### Client Requirements

| # | Requirement                                          | Priority  | Driver                                 |
|---|------------------------------------------------------|-----------|----------------------------------------|
| R1| Reduce inference cost by running models on CPU       | **High**  | OpEx reduction — GPU costs $2–8/hr     |
| R2| Support air-gapped deployment with no cloud APIs     | **High**  | Regulated/defence clients              |
| R3| Benchmark CPU vs GPU to make data-driven decisions   | **High**  | Risk mitigation before migration       |
| R4| Support BitNet 1-bit models for maximum efficiency   | **Medium**| Future-proof, marketing differentiator |
| R5| Compare latency, quality, and cost side-by-side      | **High**  | Executive decision support             |
| R6| No disruption to existing AI pipeline                | **High**  | Protect current production stability   |
| R7| Multi-tenant isolation for compute infrastructure    | **High**  | Enterprise multi-tenancy requirement   |
| R8| Automated recommendation on CPU viability            | **Medium**| Reduce manual analysis burden          |

### Success Criteria

| Criterion                                        | Measurable Outcome                             |
|--------------------------------------------------|-------------------------------------------------|
| Benchmarks can compare any two backends          | Run comparison across all 8 backend types       |
| Cost savings are quantified per workload         | Monthly savings estimate with latency trade-off |
| Air-gapped deployment is possible                | CPU inference with zero external dependencies   |
| Existing models continue working unchanged       | Zero regressions in 18 production models        |
| Data is tenant-isolated                          | Row-Level Security on all compute tables        |

---

## 2. Solution Overview

### Approach

Rather than replacing the existing inference pipeline, we introduce a **compute abstraction layer** alongside it. This layer:

1. **Registers** inference servers of any backend type (CPU, GPU, hybrid)
2. **Benchmarks** them with configurable test suites (warmup, iterations, concurrency, streaming)
3. **Compares** results side-by-side with automated viability recommendations
4. **Estimates** cost savings to support executive decision-making

The existing LLM pipeline — including tool orchestration, grounding, PII filtering, and audit logging — remains untouched. All backends speak the same OpenAI-compatible protocol, so switching a model from GPU to CPU is a configuration change, not a code change.

### What Was Delivered (Phase 1)

```
┌──────────────────────────────────────────────────────────────────┐
│                      PHASE 1: DELIVERED                          │
│                                                                  │
│  ✓ Inference Server Registry (8 backend types)                  │
│  ✓ Benchmark Suite Management (create, configure, execute)       │
│  ✓ Benchmark Runner (warmup, concurrency, streaming, retry)      │
│  ✓ Run Comparison Engine (cost, latency, quality deltas)         │
│  ✓ Cost Model Framework (per-backend pricing)                    │
│  ✓ LLM Providers for llama.cpp and BitNet                        │
│  ✓ Full Frontend (3 pages, CRUD dialogs, KPI dashboards)         │
│  ✓ Database Schema (4 tables, RLS, indexes)                      │
│  ✓ Container Infrastructure (Dockerfile, Helm charts)            │
│  ✓ Prometheus Metrics (4 new metric types)                       │
│  ✓ E2E Test Coverage (23 Playwright tests)                       │
│  ✓ 17 REST API Endpoints                                         │
└──────────────────────────────────────────────────────────────────┘
```

### How Requirements Are Met

| Req | Solution Component                                                           | Status       |
|-----|------------------------------------------------------------------------------|--------------|
| R1  | Benchmark framework proves CPU viability; llama.cpp/BitNet providers ready   | ✅ Delivered  |
| R2  | BitNet CPU provider + containerised llama.cpp with pre-loaded GGUF models    | ✅ Delivered  |
| R3  | Full benchmark runner with p50/p95/p99 latency, throughput, TTFT, quality    | ✅ Delivered  |
| R4  | BitNet provider + artifact type support + Dockerfile supports BitNet GGUF    | ✅ Delivered  |
| R5  | Comparison engine with cost/latency/quality deltas + recommendation          | ✅ Delivered  |
| R6  | New providers implement existing LLMProvider interface; pipeline unchanged   | ✅ Delivered  |
| R7  | Row-Level Security on all 4 new tables; tenant-scoped API handlers           | ✅ Delivered  |
| R8  | Automated cpu_viable / gpu_required / needs_more_data recommendations        | ✅ Delivered  |

---

## 3. High-Level Architecture

### System Context

```
                     ┌─────────────────────────┐
                     │    Security Analysts     │
                     │    Platform Admins       │
                     │    DevOps Engineers      │
                     └────────────┬────────────┘
                                  │ HTTPS
                                  ▼
                     ┌─────────────────────────┐
                     │    Clario360 Frontend    │
                     │    (Next.js 14)          │
                     │                          │
                     │  • Compute Management    │
                     │  • Benchmark Dashboard   │
                     │  • Suite Detail & Compare│
                     └────────────┬────────────┘
                                  │ REST API
                                  ▼
┌────────────────────────────────────────────────────────────┐
│                    Clario360 Backend                        │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           AI Governance Module                       │  │
│  │                                                      │  │
│  │  Existing Capabilities:          New Capabilities:   │  │
│  │  • 18 Model Registry            • Server Registry    │  │
│  │  • Version Lifecycle             • Benchmark Suites   │  │
│  │  • Prediction Logging            • Benchmark Runner   │  │
│  │  • Drift Detection               • Run Comparison     │  │
│  │  • Shadow Mode                   • Cost Estimation    │  │
│  │  • Explainability                • CPU/GPU Metrics    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           vCISO LLM Subsystem                        │  │
│  │                                                      │  │
│  │  Existing Providers:             New Providers:       │  │
│  │  • OpenAI                        • LlamaCpp (CPU)     │  │
│  │  • Anthropic                     • BitNet (CPU)       │  │
│  │  • Azure OpenAI                                       │  │
│  │  • Local (self-hosted)                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────┬──────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
     │  PostgreSQL  │  │  llama.cpp  │  │  vLLM       │
     │  (with RLS)  │  │  Server     │  │  Server     │
     │              │  │  (CPU)      │  │  (GPU)      │
     └─────────────┘  └─────────────┘  └─────────────┘
```

### Integration Points

The new compute layer integrates with the existing platform at exactly two points:

| Integration Point              | How                                                              | Impact on Existing Code |
|--------------------------------|------------------------------------------------------------------|-------------------------|
| **AI Governance Module**       | New handler, service, and repository registered in `routes.go`   | 1 file modified         |
| **LLM Provider Registry**     | Two new providers implement the existing `LLMProvider` interface | 0 interfaces changed    |

This minimal integration surface ensures that existing production models, prediction logging, drift detection, and shadow comparisons continue to operate without modification.

---

## 4. Compute Backend Strategy

### The OpenAI-Compatible Protocol Advantage

A critical design decision underpins the entire solution: **all inference backends expose the same API**.

The OpenAI Chat Completions protocol (`POST /v1/chat/completions`) has become the de facto standard for LLM serving. All major open-source serving frameworks support it natively:

```
┌────────────────────┐     ┌──────────────────────────────┐
│                    │     │  /v1/chat/completions         │
│  Clario360         │────▶│                              │
│  Benchmark Runner  │     │  Same request/response for:   │
│  or LLM Pipeline   │     │  • vLLM (GPU)                │
│                    │     │  • llama.cpp (CPU)            │
│                    │     │  • BitNet (CPU)               │
│                    │     │  • ONNX Runtime               │
└────────────────────┘     └──────────────────────────────┘
```

**Why this matters**: The existing 13-phase LLM pipeline (tool resolution, context assembly, grounding, PII redaction, completion, response parsing, audit logging, etc.) calls `provider.Complete()`. The provider handles the HTTP call. Switching from GPU to CPU is simply a matter of pointing the provider at a different URL — no pipeline changes required.

### Backend Selection Matrix

| Workload Profile              | Recommended Backend | Why                                           |
|-------------------------------|---------------------|-----------------------------------------------|
| Real-time user-facing queries | `vllm_gpu`          | Lowest latency, highest throughput            |
| Batch analysis (overnight)    | `llamacpp_cpu`      | 10x cost savings, latency doesn't matter      |
| Air-gapped / classified       | `bitnet_cpu`        | No GPU, no internet, tiny model footprint     |
| Development / testing         | `llamacpp_cpu`      | Cheap, easy to set up, good enough quality    |
| Edge / embedded               | `bitnet_cpu`        | Runs on commodity hardware, minimal resources |
| High-quality production       | `vllm_gpu`          | Best quality preservation at scale            |

### BitNet 1-Bit: The CPU-Only Future

BitNet b1.58 uses **ternary weights** (`{-1, 0, 1}`) instead of 16-bit floating point. This enables:

| Property              | Traditional LLM (FP16)          | BitNet (1.58-bit)                |
|-----------------------|---------------------------------|----------------------------------|
| Weight storage        | 16 bits per weight              | 1.58 bits per weight (~10x smaller) |
| Core operation        | Matrix multiplication (FP16)    | Integer addition (no multiply)   |
| Hardware required     | GPU (CUDA, ROCm)               | CPU only (AVX2 SIMD)            |
| Memory footprint      | ~16 GB for 8B model             | ~1.6 GB for 8B model            |
| Energy consumption    | 200–400W (GPU)                  | 15–30W (CPU)                     |
| External dependencies | CUDA drivers, GPU cloud rental  | None                             |
| Quality trade-off     | Baseline                        | ~5–15% quality reduction         |

The platform's BitNet support means clients can deploy a fully functional AI security assistant on a standard server rack with no GPU hardware, no cloud connectivity, and no external API dependencies.

---

## 5. Benchmarking Framework

### Purpose

Before migrating any production workload from GPU to CPU, clients need empirical evidence that:
1. **Latency** stays within acceptable bounds for the use case
2. **Quality** (accuracy, coherence) doesn't degrade beyond thresholds
3. **Cost savings** justify the migration effort

The benchmarking framework provides this evidence through controlled, repeatable experiments.

### Benchmark Lifecycle

```
┌────────────┐     ┌────────────┐     ┌────────────┐     ┌────────────┐
│   Define   │────▶│   Execute  │────▶│   Analyse  │────▶│   Decide   │
│   Suite    │     │   Run      │     │   Results  │     │            │
│            │     │            │     │            │     │ cpu_viable │
│ • Model    │     │ • Warmup   │     │ • Latency  │     │    or      │
│ • Prompts  │     │ • Measured │     │ • Quality  │     │ gpu_needed │
│ • Config   │     │ • Parallel │     │ • Cost     │     │    or      │
│ • Params   │     │ • Stream   │     │ • Compare  │     │ more_data  │
└────────────┘     └────────────┘     └────────────┘     └────────────┘
```

### Suite Configuration

A benchmark suite defines the "what" and "how" of a test:

| Parameter          | Purpose                                              | Default  |
|--------------------|------------------------------------------------------|----------|
| **Model slug**     | Which model to benchmark                             | Required |
| **Prompt dataset** | Array of test inputs (system + user messages)        | `[]`     |
| **Warmup count**   | Iterations to discard (warm JIT, caches, model load) | 5        |
| **Iteration count**| Measured iterations for statistical analysis         | 100      |
| **Concurrency**    | Parallel requests to stress-test throughput           | 1        |
| **Timeout**        | Per-request timeout                                  | 60s      |
| **Stream enabled** | Use SSE streaming to measure time-to-first-token     | false    |
| **Max retries**    | Retry transient failures with exponential backoff     | 3        |

### What Gets Measured

Each benchmark run produces a comprehensive metrics profile across five dimensions:

```
                         Benchmark Run Results
    ┌──────────────────────────────────────────────────────────┐
    │                                                          │
    │  LATENCY               THROUGHPUT          STREAMING     │
    │  ───────               ──────────          ─────────     │
    │  p50, p95, p99         Tokens/sec          TTFT p50      │
    │  Average               Requests/sec        TTFT p95      │
    │  Min, Max              Total tokens        TTFT avg      │
    │                        Total requests                    │
    │                        Failed / Retried                  │
    │                                                          │
    │  QUALITY               RESOURCES           COST          │
    │  ───────               ─────────           ────          │
    │  Perplexity            Peak CPU %          $/hour        │
    │  BLEU score            Peak Memory MB      $/1K tokens   │
    │  ROUGE-L               Avg CPU %                         │
    │  Semantic similarity   Avg Memory MB                     │
    │  Factual accuracy                                        │
    │                                                          │
    └──────────────────────────────────────────────────────────┘
```

### Execution Engine Design

The benchmark runner is designed for accuracy and resilience:

| Design Aspect          | Approach                                                      |
|------------------------|---------------------------------------------------------------|
| **Warmup isolation**   | Warmup results are fully discarded — only measured phase counts |
| **Concurrency control**| Semaphore-based — exactly N requests in flight at any time    |
| **Retry resilience**   | Exponential backoff (500ms → 10s cap) for transient failures  |
| **Retryable errors**   | HTTP 429, 500, 502, 503, 504 and network-level errors         |
| **Streaming TTFT**     | SSE parsing records wall-clock time to first content chunk    |
| **Statistical rigour** | Interpolated percentiles on sorted latency arrays             |
| **Context cancellation**| Respects parent context for graceful shutdown                |

### Comparison & Recommendation

When comparing two or more runs, the system computes:

| Metric                  | Calculation                                         |
|-------------------------|-----------------------------------------------------|
| **Cost delta**          | (Run A hourly cost − Run B hourly cost) × 730 hours |
| **Latency delta**       | Percentage difference in p95 latency                |
| **Quality delta**       | Percentage difference in semantic similarity        |

And generates an automated recommendation:

| Recommendation     | When                                                    |
|--------------------|---------------------------------------------------------|
| **CPU Viable**     | CPU latency is within 3x of GPU with < 10% quality loss |
| **GPU Required**   | CPU latency exceeds 3x of GPU                            |
| **Needs More Data**| Results are inconclusive — run more iterations           |

This provides executives and architects with a clear, data-driven signal for migration decisions.

---

## 6. Cost-Benefit Analysis Framework

### Cost Model Design

The platform maintains per-backend cost models that map infrastructure configurations to hourly pricing:

```
┌────────────────────────────────────────────────────────────┐
│                    Cost Model                               │
│                                                             │
│  Name: "CPU-8c-32GB"        Name: "A100-1xGPU"            │
│  Backend: llamacpp_cpu       Backend: vllm_gpu              │
│  Instance: c6i.2xlarge       Instance: p4d.24xlarge         │
│  Hourly: $0.34               Hourly: $3.40                  │
│  CPU: 8 cores                CPU: 96 cores                  │
│  Memory: 32 GB               Memory: 1,152 GB               │
│  GPU: none                   GPU: 8× A100                   │
│  Throughput: ~45 tok/s       Throughput: ~450 tok/s          │
│                                                             │
│  Monthly: $248                Monthly: $2,482               │
│                                                             │
│             Monthly Savings: $2,234 (90%)                   │
│             Latency Trade-off: ~2.5× slower                 │
└────────────────────────────────────────────────────────────┘
```

### Savings Estimation

The cost estimation endpoint takes a CPU benchmark run and a GPU benchmark run and produces:

| Output                    | Description                                          |
|---------------------------|------------------------------------------------------|
| CPU monthly cost          | CPU hourly rate × 730 hours                          |
| GPU monthly cost          | GPU hourly rate × 730 hours                          |
| Monthly savings           | GPU cost − CPU cost                                  |
| Savings percentage        | Savings as a fraction of GPU cost                    |
| Latency increase          | Percentage increase in p95 latency (CPU vs GPU)      |
| Throughput comparison     | Tokens/second for each backend                       |

### Typical Savings Scenarios

| Workload                        | GPU Cost/mo | CPU Cost/mo | Savings  | Latency Impact |
|---------------------------------|-------------|-------------|----------|----------------|
| Batch threat analysis (8hr/day) | $816        | $82         | $734/mo  | Acceptable     |
| Compliance scanning (24/7)      | $2,482      | $248        | $2,234/mo| Acceptable     |
| Interactive vCISO chat          | $2,482      | $248        | —        | Too slow       |
| Report generation (nightly)     | $408        | $41         | $367/mo  | Not relevant   |

The key insight: **not all workloads need real-time latency**. Batch processing, overnight analysis, report generation, and compliance scanning can run on CPU with 90% cost savings and no user-visible impact.

---

## 7. Data Model Design

### Entity Relationships

```
                        Tenant
                          │
          ┌───────────────┼───────────────┐
          │               │               │
    Inference          Benchmark        Cost
    Servers            Suites           Models
          │               │
          │               │
          └───────┬───────┘
                  │
            Benchmark
              Runs
```

### Core Entities

**Inference Server** — A registered inference endpoint with hardware specifications.

| Attribute       | Description                                               |
|-----------------|-----------------------------------------------------------|
| Identity        | Name, backend type, base URL, health endpoint             |
| Hardware specs  | CPU cores, memory, GPU type/count                         |
| Configuration   | Model name, quantization, max concurrency, stream capable |
| Lifecycle       | Status: provisioning → healthy → degraded → decommissioned |

**Benchmark Suite** — A reusable, parameterised test definition.

| Attribute       | Description                                               |
|-----------------|-----------------------------------------------------------|
| Target          | Model slug to benchmark                                   |
| Test data       | Array of prompt pairs (system + user messages)            |
| Execution       | Warmup, iterations, concurrency, timeout, retry policy    |
| Streaming       | Whether to measure time-to-first-token via SSE            |

**Benchmark Run** — The complete results of executing a suite against a server.

| Attribute       | Description                                               |
|-----------------|-----------------------------------------------------------|
| Context         | Which suite, which server, which backend type             |
| Latency         | p50, p95, p99, average, min, max (milliseconds)          |
| Throughput      | Tokens/second, requests/second, total tokens/requests     |
| Reliability     | Failed requests, retried requests                         |
| Streaming       | TTFT p50, p95, average (when streaming is used)           |
| Quality         | Perplexity, BLEU, ROUGE-L, semantic similarity, accuracy |
| Resources       | Peak and average CPU %, peak and average memory           |
| Cost            | Estimated hourly cost, cost per 1K tokens                 |
| Lifecycle       | pending → running → completed / failed / cancelled        |
| Raw data        | Full per-iteration results stored as JSON                 |

**Cost Model** — Pricing data for a specific infrastructure configuration.

| Attribute       | Description                                               |
|-----------------|-----------------------------------------------------------|
| Identity        | Name, backend type, cloud instance type                   |
| Pricing         | Hourly cost in USD                                        |
| Hardware        | CPU cores, memory GB, GPU type/count                      |
| Performance     | Maximum tokens per second                                 |

### Data Isolation

All four entities are tenant-scoped using PostgreSQL Row-Level Security. Each table has:
- A `tenant_id` foreign key to the `tenants` table
- An RLS policy that filters rows by the current session's tenant ID
- Cascading deletes tied to the parent tenant

This ensures that Tenant A cannot see, query, or modify Tenant B's inference servers, benchmarks, or cost models — even at the database level.

---

## 8. Frontend Experience

### Navigation

The compute infrastructure pages are integrated into the existing AI Governance admin section:

```
Admin
└── AI Governance
    ├── Model Registry        (existing)
    ├── Predictions           (existing)
    ├── Compute Infrastructure  ← NEW
    └── Benchmarks              ← NEW
        └── Suite Detail        ← NEW (dynamic route)
```

### Compute Infrastructure Page

**Purpose**: Register, monitor, and manage inference server endpoints.

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Compute Infrastructure                  [Add Server] [↻]  │
│  Manage inference servers for CPU and GPU model serving     │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │  Total   │ │ Healthy  │ │   CPU    │ │   GPU    │      │
│  │ Servers  │ │          │ │ Backends │ │ Backends │      │
│  │    5     │ │    3     │ │    3     │ │    2     │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Name            │Backend    │Model        │Status   │   │
│  ├─────────────────┼───────────┼─────────────┼─────────┤   │
│  │ llamacpp-cpu-01 │llama.cpp  │llama-3.1-8b │✓ healthy│   │
│  │ bitnet-cpu-dev  │BitNet CPU │bitnet-3b    │✓ healthy│   │
│  │ vllm-gpu-prod   │vLLM GPU   │mistral-7b   │✓ healthy│   │
│  │ onnx-test       │ONNX CPU   │—            │◌ provis.│   │
│  └─────────────────┴───────────┴─────────────┴─────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key interactions**:
- **Add Server**: Dialog to register a new endpoint with backend type selection (8 options), connection details, and hardware specs
- **Decommission**: Confirmation dialog that marks a server as decommissioned (soft delete — historical data preserved)
- **Refresh**: Reloads both KPI cards and server table
- **Status management**: Mark servers healthy/degraded/offline

### Benchmarks Page

**Purpose**: Create benchmark suites, execute them, and view results across all runs.

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  Inference Benchmarks                       [New Suite]     │
│  Measure and compare CPU vs GPU inference latency           │
│                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │Benchmark │ │  Total   │ │Completed │ │   Avg    │      │
│  │ Suites   │ │  Runs    │ │          │ │ Latency  │      │
│  │    3     │ │   12     │ │   10     │ │  312ms   │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│                                                             │
│  [Suites]  [Run Results]                                    │
│                                                             │
│  ┌──────────────────┬────────┬───────────┬──────┬────────┐ │
│  │ Suite Name       │ Model  │ Config    │ Runs │ Action │ │
│  ├──────────────────┼────────┼───────────┼──────┼────────┤ │
│  │ threat-scorer-   │threat- │ 5 warmup  │  4   │ [Run]  │ │
│  │ cpu-bench        │scorer  │ 100 iter  │      │        │ │
│  │                  │        │ 1 conc.   │      │        │ │
│  └──────────────────┴────────┴───────────┴──────┴────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key interactions**:
- **Tab navigation**: Switch between Suites and Run Results views
- **New Suite**: Dialog with model slug, prompt configuration, and execution parameters
- **Run**: Select a target inference server and execute the benchmark
- **Click suite name**: Navigate to detailed suite view with full run history

### Suite Detail Page

**Purpose**: Deep-dive into a specific benchmark suite with run history and comparison.

```
┌─────────────────────────────────────────────────────────────┐
│  ← Back to Benchmarks                                       │
│                                                             │
│  threat-scorer-cpu-bench                                    │
│  Suite configuration and execution history                  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Run History                     [Compare Selected]  │   │
│  │                                                     │   │
│  │ ☐ │ Backend     │ Status    │ p95 Latency │ tok/s  │   │
│  │ ☑ │ llamacpp_cpu│ completed │   512ms     │  42.5  │   │
│  │ ☑ │ vllm_gpu   │ completed │   89ms      │  380.2 │   │
│  │ ☐ │ bitnet_cpu │ completed │   1,204ms   │  18.3  │   │
│  │ ☐ │ llamacpp_cpu│ failed    │   —         │  —     │   │
│  │                                                     │   │
│  │ ── Comparison Results ──────────────────────────    │   │
│  │ Cost Delta:    -$2,190/month (CPU saves 90%)        │   │
│  │ Latency Delta: +475% (CPU is 5.75× slower)          │   │
│  │ Quality Delta: -2.1% (minimal quality loss)          │   │
│  │                                                     │   │
│  │ Recommendation: ⚠ GPU Required                      │   │
│  │ CPU latency is 5.75× GPU — not suitable for         │   │
│  │ real-time production use.                            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Key interactions**:
- **Multi-select runs**: Checkbox selection for side-by-side comparison
- **Compare Selected**: Triggers comparison API with selected run IDs
- **Comparison panel**: Shows cost delta, latency delta, quality delta, and recommendation

---

## 9. Deployment Architecture

### Container Strategy

The solution uses a single container image that serves both standard quantised models and BitNet 1-bit models:

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Dockerfile.llamacpp (Multi-Stage)                       │
│                                                          │
│  Stage 1: Build                                          │
│  ├── Ubuntu 24.04 base                                   │
│  ├── Clone llama.cpp (configurable branch/tag)           │
│  └── Build with CPU optimisations (AVX2, F16C, FMA)     │
│                                                          │
│  Stage 2: Runtime                                        │
│  ├── Minimal Ubuntu 24.04 (curl for healthcheck only)   │
│  ├── Non-root user (llama:llama)                         │
│  ├── Model directory at /models                          │
│  ├── Configurable via environment variables              │
│  └── Health check: GET /health every 15s                 │
│                                                          │
│  Supports: Q4_0, Q5_K_M, Q8_0, BitNet 1-bit GGUF       │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Kubernetes Deployment

```
┌──────────────────────────────────────────────────────────┐
│  Kubernetes Cluster                                       │
│                                                          │
│  ┌──────────────────────────────────────────────┐        │
│  │  Clario360 Namespace                          │        │
│  │                                                │        │
│  │  ┌─────────────┐    ┌─────────────────────┐   │        │
│  │  │ API Server  │    │ llama.cpp Deployment │   │        │
│  │  │ (existing)  │───▶│                     │   │        │
│  │  │             │    │ Replicas: 2          │   │        │
│  │  └─────────────┘    │ Strategy: Rolling    │   │        │
│  │                     │                     │   │        │
│  │                     │ Init Container:      │   │        │
│  │                     │   Model download     │   │        │
│  │                     │   (curl from S3/URL) │   │        │
│  │                     │                     │   │        │
│  │                     │ Main Container:      │   │        │
│  │                     │   llama-server       │   │        │
│  │                     │   Port 8080          │   │        │
│  │                     │   Liveness probe     │   │        │
│  │                     │   Readiness probe    │   │        │
│  │                     │                     │   │        │
│  │                     │ Volume:              │   │        │
│  │                     │   PVC or emptyDir    │   │        │
│  │                     │   for model storage  │   │        │
│  │                     └─────────┬───────────┘   │        │
│  │                               │               │        │
│  │                     ┌─────────▼───────────┐   │        │
│  │                     │ ClusterIP Service   │   │        │
│  │                     │ Port 8080           │   │        │
│  │                     └─────────────────────┘   │        │
│  └────────────────────────────────────────────────┘        │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

**Deployment features**:

| Feature                 | Implementation                                          |
|-------------------------|---------------------------------------------------------|
| Model provisioning      | Init container downloads GGUF model from configurable URL |
| Zero-downtime deploys   | Rolling update with maxSurge=1, maxUnavailable=0        |
| Health monitoring        | Liveness probe (30s initial) + readiness probe (10s initial) |
| Resource management      | Configurable CPU/memory requests and limits             |
| Node scheduling          | Optional nodeSelector for inference-dedicated nodes     |
| Security                 | Non-root user, fsGroup 1000                             |
| Persistence              | Optional PVC for model storage across pod restarts      |
| Conditional deployment   | Only deployed when `inference.llamacpp.enabled = true`  |

### Air-Gapped Deployment Topology

```
┌─────────────────────────────────────────────────────────┐
│  Air-Gapped Environment (No Internet)                    │
│                                                          │
│  ┌─────────────┐    ┌──────────────────┐                │
│  │ Clario360   │    │ BitNet CPU Server │                │
│  │ Platform    │───▶│                  │                │
│  │             │    │ Pre-loaded GGUF  │                │
│  │ All 18 AI   │    │ model from       │                │
│  │ models run  │    │ internal registry│                │
│  │ inline      │    │                  │                │
│  └─────────────┘    │ No GPU needed    │                │
│                     │ No internet      │                │
│                     │ 8 CPU cores      │                │
│                     │ 16 GB RAM        │                │
│                     └──────────────────┘                │
│                                                          │
│  Total external dependencies: ZERO                       │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 10. Security & Multi-Tenancy

### Tenant Isolation

| Layer          | Mechanism                                                    |
|----------------|--------------------------------------------------------------|
| **API**        | JWT `tenant_id` extracted by middleware; handlers reject cross-tenant access |
| **Database**   | Row-Level Security policies on all 4 tables filter by `app.current_tenant_id` |
| **Indexes**    | All query indexes include `tenant_id` as leading column      |
| **Cascade**    | `ON DELETE CASCADE` from `tenants` table removes all data on tenant deletion |

### Authentication & Authorisation

| Aspect                 | Approach                                                |
|------------------------|---------------------------------------------------------|
| Authentication         | RS256 JWT tokens issued by IAM service                  |
| API key storage        | Inference server API keys stored in database (encrypted at rest) |
| Write operations       | Require authenticated user (`userID` extracted from JWT) |
| Read operations        | Require tenant context only                             |
| Permission gating      | Frontend pages wrapped in `PermissionRedirect`          |

### Data Sensitivity

| Data Type              | Sensitivity | Protection                                  |
|------------------------|-------------|---------------------------------------------|
| Inference server URLs  | Medium      | Tenant-isolated, not exposed in logs        |
| API keys               | High        | Stored in DB, omitted from JSON responses where marked |
| Benchmark prompts      | Medium      | Tenant-scoped, stored as JSONB              |
| Run results            | Low         | Statistical aggregates, no PII              |
| Raw iteration results  | Medium      | May contain model responses — tenant-isolated |

---

## 11. Observability Strategy

### Metrics

Four new Prometheus metrics provide visibility into the compute layer:

| Metric                           | Type      | Dimensions                | Alerting Use                      |
|----------------------------------|-----------|---------------------------|-----------------------------------|
| Benchmark runs completed         | Counter   | Backend type, status      | Track execution volume and failures |
| Benchmark latency distribution   | Histogram | Backend type              | Detect latency regressions        |
| Inference server health          | Gauge     | Server name, backend type | Alert on server outages           |
| Cost per token                   | Gauge     | Backend type              | Track cost trends over time       |

### Recommended Alerts

| Alert                    | Condition                                    | Severity |
|--------------------------|----------------------------------------------|----------|
| Server unreachable       | Health gauge = 0 for 5 minutes               | Critical |
| CPU latency regression   | p95 latency exceeds historical baseline by 2× | Warning  |
| High benchmark failure   | > 20% of benchmark requests failing           | Warning  |
| Cost anomaly             | Cost per token increases > 50% week-over-week | Info     |

### Logging

All benchmark operations emit structured JSON logs via zerolog:
- Suite creation/deletion events
- Run start/complete/fail events with timing
- Server registration/decommission events
- Comparison results with recommendations

---

## 12. Quality Assurance

### End-to-End Test Coverage

The solution includes **23 Playwright E2E tests** covering the complete user workflow:

| Area                     | Tests | Coverage                                                |
|--------------------------|-------|---------------------------------------------------------|
| Compute page rendering   | 4     | Header, KPI cards, server table, server row details     |
| Server CRUD operations   | 3     | Register dialog, form submission, decommission          |
| Compute interactions     | 2     | Refresh button, cancel dialog                           |
| Benchmark page rendering | 4     | Header, KPI cards, suite data, run results tab          |
| Suite CRUD operations    | 3     | Create dialog, form submission, cancel                  |
| Benchmark interactions   | 2     | Run benchmark dialog, navigation to detail              |
| Suite detail page        | 2     | Configuration display, run history                      |
| Auth setup               | 1     | Login authentication for all tests                      |
| **Total**                | **23**|                                                         |

### Test Design Principles

| Principle                | Implementation                                          |
|--------------------------|----------------------------------------------------------|
| Data independence        | Unique names with timestamps prevent cross-test pollution |
| Strict mode compliance   | Exact text matching, role selectors, row-scoped locators |
| Auth reuse               | Login once, share storage state across all tests         |
| Real backend             | Tests run against live API — not mocked                  |

---

## 13. Phased Delivery Roadmap

### Phase 1: Benchmarking Framework — DELIVERED

Everything needed to register servers, run benchmarks, compare results, and estimate costs.

### Phase 2: Shadow Mode Compute Comparison

**Goal**: Deploy CPU inference as a shadow version alongside GPU production and compare them using the existing shadow infrastructure.

| Deliverable                       | Description                                             |
|-----------------------------------|---------------------------------------------------------|
| Compute config on model versions  | Link versions to specific backends and inference servers |
| Inference router with fallback    | Route CPU → GPU when CPU latency exceeds threshold      |
| Shadow comparison integration     | Shadow version with CPU backend runs alongside GPU prod |
| Frontend: compute config dialog   | Configure compute backend when creating model versions  |
| Frontend: compute comparison panel| Side-by-side CPU vs GPU metrics on shadow comparison    |

**Key advantage**: No new A/B testing code is needed. The existing shadow executor, comparator, and comparison service work identically — the only difference is the shadow version uses a CPU backend.

### Phase 3: Production Routing & Cost Dashboard

**Goal**: Route production traffic based on workload policies; full cost visibility.

| Deliverable                       | Description                                             |
|-----------------------------------|---------------------------------------------------------|
| Routing policy engine             | Rule-based routing: short queries→CPU, complex→GPU, off-peak→CPU |
| Enhanced cost tracking            | Cumulative cost per backend, per model, per time period |
| Cost savings dashboard            | Time-series savings chart, distribution pie, ROI metrics |
| Production monitoring             | Inference server health grid, fallback rate tracking    |
| Air-gapped deployment support     | BitNet models pre-loaded from internal registry         |

---

## 14. Risk Assessment & Mitigations

| Risk                                  | Likelihood | Impact | Mitigation                                              |
|---------------------------------------|------------|--------|---------------------------------------------------------|
| CPU latency too high for real-time    | High       | Medium | Benchmark first; only migrate batch/async workloads     |
| BitNet quality degradation            | Medium     | High   | Quality metrics in benchmark; automated viability check |
| Model format incompatibility          | Low        | Medium | GGUF is universal format for llama.cpp/BitNet           |
| Inference server instability          | Medium     | High   | Health probes, auto-restart, fallback to GPU (Phase 2)  |
| Tenant data leakage                   | Low        | Critical| RLS on all tables; tenant_id in all queries             |
| Benchmark results not representative  | Medium     | Medium | Configurable warmup, iterations, real prompt datasets   |
| Air-gapped model updates              | Low        | Low    | Model files loaded from internal registry at deploy     |

---

## 15. Decision Log

| # | Decision                                      | Rationale                                                        | Date       |
|---|-----------------------------------------------|------------------------------------------------------------------|------------|
| D1| Use OpenAI-compatible protocol for all backends | Universal standard; vLLM, llama.cpp, BitNet all support it natively | 2026-03-08 |
| D2| Single Dockerfile for both quantised and BitNet | Same llama.cpp binary serves both; simpler operations            | 2026-03-08 |
| D3| Reuse existing shadow infrastructure for Phase 2 | Avoids building new A/B testing; shadow executor already works   | 2026-03-08 |
| D4| Benchmark runner is synchronous (not async job) | Simpler implementation; benchmarks typically complete in < 5min  | 2026-03-09 |
| D5| Soft-delete servers (decommission, not remove) | Preserves historical benchmark runs that reference the server    | 2026-03-09 |
| D6| Relaxed prompt_dataset validation on create    | Allows creating suites first, adding prompts later               | 2026-03-12 |
| D7| All list endpoints use WritePaginated envelope | Consistent with existing AI governance API contract              | 2026-03-12 |
| D8| Comparison recommendation uses 3x latency threshold | Balances cost savings against user experience degradation   | 2026-03-09 |
| D9| Store raw iteration results as JSONB            | Enables future re-analysis without re-running benchmarks        | 2026-03-09 |
| D10| Tenant-scoped inference servers (not global)  | Different tenants may use different infrastructure               | 2026-03-08 |
