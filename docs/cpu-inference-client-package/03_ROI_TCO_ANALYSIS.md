# ROI & Total Cost of Ownership Analysis — CPU vs GPU Inference

| Attribute    | Value                                           |
|--------------|-------------------------------------------------|
| **For**      | CFO, CTO, Procurement, Finance                 |
| **Date**     | 2026-03-14                                      |
| **Status**   | Phase 1 Delivered                               |
| **Read time**| 8 minutes                                       |

---

## Purpose

This document provides a financial analysis of migrating AI inference workloads from GPU to CPU infrastructure using Clario360's benchmarking framework. It covers infrastructure cost modelling, total cost of ownership projections, break-even analysis, and ROI calculations.

---

## Table of Contents

1. [Cost Model Overview](#1-cost-model-overview)
2. [Infrastructure Cost Comparison](#2-infrastructure-cost-comparison)
3. [Workload Migration Analysis](#3-workload-migration-analysis)
4. [Total Cost of Ownership (3-Year)](#4-total-cost-of-ownership-3-year)
5. [Break-Even Analysis](#5-break-even-analysis)
6. [ROI Calculation](#6-roi-calculation)
7. [Risk-Adjusted Projections](#7-risk-adjusted-projections)
8. [Recommendations](#8-recommendations)

---

## 1. Cost Model Overview

Clario360 supports per-backend cost modelling that maps real infrastructure costs to inference workloads. This enables accurate, data-driven comparisons between compute strategies.

### Pricing Components

| Component              | GPU (Cloud)                | GPU (On-Prem)              | CPU (Cloud)              | CPU (On-Prem)            |
|------------------------|----------------------------|----------------------------|--------------------------|--------------------------|
| Instance/server cost   | $3.06–$32.77/hr            | Capital purchase           | $0.17–$1.53/hr           | Capital purchase         |
| GPU rental             | Included in instance       | $10K–$40K per GPU          | Not applicable           | Not applicable           |
| Storage                | $0.08/GB/mo (EBS)          | SAN/NAS amortised          | $0.08/GB/mo              | SAN/NAS amortised        |
| Network egress         | $0.09/GB                   | Included                   | $0.09/GB                 | Included                 |
| Software licensing     | None (open source)         | None                       | None                     | None                     |
| Operations/management  | Standard DevOps            | Standard DevOps            | Standard DevOps          | Standard DevOps          |

### Reference Instance Types

| Backend        | Instance Type     | vCPU | RAM (GB) | GPU             | Hourly Cost | Monthly Cost |
|----------------|-------------------|------|----------|-----------------|-------------|--------------|
| vLLM GPU       | p4d.24xlarge      | 96   | 1,152    | 8× A100 (40GB)  | $32.77      | $23,922      |
| vLLM GPU       | g5.xlarge         | 4    | 16       | 1× A10G (24GB)  | $1.006      | $734         |
| vLLM GPU       | g5.2xlarge        | 8    | 32       | 1× A10G (24GB)  | $1.212      | $885         |
| llama.cpp CPU  | c6i.2xlarge       | 8    | 16       | None             | $0.34       | $248         |
| llama.cpp CPU  | c6i.4xlarge       | 16   | 32       | None             | $0.68       | $496         |
| llama.cpp CPU  | c7i.8xlarge       | 32   | 64       | None             | $1.428      | $1,042       |
| BitNet CPU     | c6i.xlarge        | 4    | 8        | None             | $0.17       | $124         |
| BitNet CPU     | c6i.2xlarge       | 8    | 16       | None             | $0.34       | $248         |

*Prices based on AWS US-East-1 on-demand pricing. Reserved/spot instances reduce costs further.*

---

## 2. Infrastructure Cost Comparison

### Single-Server Comparison

For a single inference endpoint running 24/7:

| Configuration              | Hardware               | Monthly Cost | Tokens/sec | Cost per 1M Tokens |
|----------------------------|------------------------|--------------|------------|---------------------|
| GPU (small)                | g5.xlarge, 1× A10G     | $734         | 200–400    | $0.85               |
| GPU (large)                | p4d.24xlarge, 8× A100  | $23,922      | 2,000–5,000| $0.22               |
| CPU (standard)             | c6i.2xlarge, 8 cores   | $248         | 30–60      | $1.90               |
| CPU (high-perf)            | c7i.8xlarge, 32 cores  | $1,042       | 100–180    | $0.54               |
| BitNet CPU                 | c6i.xlarge, 4 cores    | $124         | 15–30      | $1.90               |

### Key Insight

CPU inference has a **higher per-token cost** but a **dramatically lower infrastructure cost**. For workloads that don't need maximum throughput — batch processing, async analysis, off-hours computation — the raw infrastructure savings dominate.

---

## 3. Workload Migration Analysis

### Clario360 AI Workload Classification

| Workload Category          | Current Backend | Latency Needs | Migration Candidate | Potential Savings |
|----------------------------|-----------------|---------------|---------------------|-------------------|
| **Batch threat analysis**  | Cloud API       | Non-real-time | Yes — CPU           | High              |
| **Compliance scanning**    | Cloud API       | Non-real-time | Yes — CPU           | High              |
| **Report generation**      | Cloud API       | Non-real-time | Yes — CPU           | High              |
| **vCISO chat (interactive)**| Cloud API      | Real-time     | Partial — GPU stays | Medium            |
| **Anomaly detection**      | Inline Go       | Real-time     | No — already CPU    | None              |
| **Threat scoring**         | Inline Go       | Real-time     | No — already CPU    | None              |
| **Alert triage**           | Cloud API       | Near-real-time| Maybe — benchmark   | Medium            |
| **Root cause analysis**    | Cloud API       | Async         | Yes — CPU           | High              |

### Migration Priority Matrix

```
                        HIGH SAVINGS
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
          │  Batch Threat   │  Alert Triage   │
          │  Compliance     │  (benchmark     │
   LOW    │  Reports        │   to confirm)   │   HIGH
 RISK ────│  Root Cause     │                 │──── RISK
          │  Analysis       │                 │
          │                 │                 │
          │  MIGRATE FIRST  │  BENCHMARK FIRST│
          │                 │                 │
          ├─────────────────┼─────────────────┤
          │                 │                 │
          │  Anomaly Det.   │  vCISO Chat     │
          │  Threat Scoring │  (keep GPU)     │
          │  (already CPU)  │                 │
          │                 │                 │
          │  NO ACTION      │  KEEP GPU       │
          │                 │                 │
          └─────────────────┼─────────────────┘
                            │
                        LOW SAVINGS
```

---

## 4. Total Cost of Ownership (3-Year)

### Scenario: Migrate 3 Batch Workloads to CPU

**Assumptions**:
- 3 batch workloads currently using cloud API (estimated equivalent: g5.xlarge GPU)
- Migrate to c6i.2xlarge CPU instances
- 1 GPU server retained for interactive vCISO chat
- Year 2–3 pricing assumes reserved instances (30% discount)

| Cost Category             | Year 1          | Year 2          | Year 3          | 3-Year Total    |
|---------------------------|-----------------|-----------------|-----------------|-----------------|
| **GPU-Only (Current)**    |                 |                 |                 |                 |
| 4× g5.xlarge instances    | $35,232         | $35,232         | $35,232         | $105,696        |
| Operations & monitoring   | $12,000         | $12,000         | $12,000         | $36,000         |
| **GPU-Only Total**        | **$47,232**     | **$47,232**     | **$47,232**     | **$141,696**    |
|                           |                 |                 |                 |                 |
| **Hybrid (CPU + GPU)**    |                 |                 |                 |                 |
| 3× c6i.2xlarge CPU        | $8,928          | $6,250          | $6,250          | $21,428         |
| 1× g5.xlarge GPU          | $8,808          | $6,166          | $6,166          | $21,140         |
| Migration effort (one-time)| $5,000         | —               | —               | $5,000          |
| Benchmarking & validation | $2,000          | —               | —               | $2,000          |
| Operations & monitoring   | $12,000         | $12,000         | $12,000         | $36,000         |
| **Hybrid Total**          | **$36,736**     | **$24,416**     | **$24,416**     | **$85,568**     |
|                           |                 |                 |                 |                 |
| **Annual Savings**        | **$10,496**     | **$22,816**     | **$22,816**     | **$56,128**     |
| **Cumulative Savings**    | $10,496         | $33,312         | $56,128         |                 |

### 3-Year TCO Reduction: 40%

```
  TCO ($K)
  50 ┤
     │ ████████████████████████  GPU-Only: $47.2K
  40 ┤ ████████████████████████
     │ ████████████████████████  ┌─────────┐
  30 ┤ ████████████████████████  │ Hybrid  │
     │ ████████████████████████  │ $36.7K  │  ┌─────────┐  ┌─────────┐
  20 ┤ ████████████████████████  │ Year 1  │  │ $24.4K  │  │ $24.4K  │
     │ ████████████████████████  │         │  │ Year 2  │  │ Year 3  │
  10 ┤ ████████████████████████  │         │  │         │  │         │
     │ ████████████████████████  │         │  │         │  │         │
   0 ┼────────────────────────┴──┴─────────┴──┴─────────┴──┴─────────┘
          GPU-Only                      Hybrid (CPU + GPU)
```

---

## 5. Break-Even Analysis

### One-Time Migration Costs

| Item                              | Estimated Cost |
|-----------------------------------|----------------|
| Benchmark suite creation & testing| $2,000         |
| Infrastructure provisioning       | $1,000         |
| Application configuration         | $1,000         |
| Validation & sign-off             | $1,000         |
| **Total migration cost**          | **$5,000**     |

### Monthly Savings After Migration

| Workload                     | Monthly GPU Cost | Monthly CPU Cost | Monthly Savings |
|------------------------------|------------------|------------------|-----------------|
| Batch threat analysis        | $734             | $248             | $486            |
| Compliance scanning          | $734             | $248             | $486            |
| Report generation            | $734             | $248             | $486            |
| **Total monthly savings**    |                  |                  | **$1,458**      |

### Break-Even Point

```
Migration cost: $5,000
Monthly savings: $1,458/month

Break-even: 5,000 ÷ 1,458 = 3.4 months
```

**The migration pays for itself within 4 months.**

---

## 6. ROI Calculation

### Year 1 ROI

```
Investment:     $7,000 (migration + benchmarking)
Annual savings: $17,496 (12 × $1,458)
Net benefit:    $10,496

ROI = (Net Benefit / Investment) × 100
ROI = ($10,496 / $7,000) × 100
ROI = 150%
```

### 3-Year ROI

```
Investment:       $7,000 (one-time)
3-year savings:   $56,128
Net benefit:      $49,128

3-Year ROI = ($49,128 / $7,000) × 100
3-Year ROI = 702%
```

---

## 7. Risk-Adjusted Projections

### Conservative Scenario (Pessimistic)

| Assumption                          | Adjustment                              |
|-------------------------------------|-----------------------------------------|
| Only 2 of 3 workloads migrate       | 33% reduction in savings               |
| CPU needs higher-spec instances     | 50% increase in CPU costs              |
| Migration takes longer              | 50% increase in migration costs        |

| Metric            | Conservative | Base Case |
|--------------------|-------------|-----------|
| Year 1 savings     | $4,148      | $10,496   |
| 3-year savings     | $28,564     | $56,128   |
| Break-even         | 7 months    | 4 months  |
| 3-year ROI         | 271%        | 702%      |

### Aggressive Scenario (Optimistic)

| Assumption                          | Adjustment                              |
|-------------------------------------|-----------------------------------------|
| All 4 LLM workloads migrate to CPU | 33% increase in savings                |
| Reserved instance pricing           | 30% reduction in infrastructure costs  |
| Spot instances for batch workloads  | 60% reduction in batch processing costs|

| Metric            | Aggressive  | Base Case |
|--------------------|------------|-----------|
| Year 1 savings     | $19,296    | $10,496   |
| 3-year savings     | $82,416    | $56,128   |
| Break-even         | 2 months   | 4 months  |
| 3-year ROI         | 1,077%     | 702%      |

---

## 8. Recommendations

### Immediate Actions

1. **Run benchmarks** on the 3 batch workloads identified as migration candidates using the [Benchmark Playbook](02_BENCHMARK_PLAYBOOK.md)
2. **Create cost models** in Clario360 for your specific infrastructure (cloud instances or on-prem servers)
3. **Validate quality** — ensure CPU inference meets accuracy thresholds for each workload

### Migration Sequence

| Priority | Workload               | Timeline  | Expected Monthly Savings |
|----------|------------------------|-----------|--------------------------|
| 1        | Report generation      | Month 1   | $486                     |
| 2        | Compliance scanning    | Month 2   | $486                     |
| 3        | Batch threat analysis  | Month 3   | $486                     |
| 4        | Alert triage           | Month 4+  | Benchmark first          |

### Scaling Projections

| Scale               | Endpoints | Monthly GPU Cost | Monthly CPU Cost | Monthly Savings |
|----------------------|-----------|------------------|------------------|-----------------|
| Small (3 workloads)  | 4 servers | $2,936           | $1,478           | $1,458          |
| Medium (6 workloads) | 8 servers | $5,872           | $2,232           | $3,640          |
| Large (12 workloads) | 16 servers| $11,744          | $3,968           | $7,776          |
| Enterprise (24+ wkld)| 32 servers| $23,488          | $7,440           | $16,048         |

At enterprise scale, CPU migration could save **over $192,000 per year**.

---

*For benchmark execution guidance, see the [Benchmark Playbook](02_BENCHMARK_PLAYBOOK.md). For deployment details, see the [Air-Gapped Deployment Guide](04_AIRGAPPED_DEPLOYMENT_GUIDE.md).*
