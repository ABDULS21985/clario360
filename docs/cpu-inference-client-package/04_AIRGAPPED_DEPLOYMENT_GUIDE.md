# Air-Gapped Deployment Guide — CPU Inference Without External Dependencies

| Attribute    | Value                                                  |
|--------------|--------------------------------------------------------|
| **For**      | Security Architects, DevOps, Compliance Officers       |
| **Date**     | 2026-03-14                                             |
| **Status**   | Phase 1 Delivered                                      |
| **Read time**| 8 minutes                                              |

---

## Purpose

This guide describes how to deploy Clario360's AI inference capabilities in environments with no internet connectivity, no cloud API access, and no GPU hardware. It is designed for regulated, classified, and high-security environments where data sovereignty and network isolation are mandatory.

---

## Table of Contents

1. [Air-Gapped Architecture](#1-air-gapped-architecture)
2. [Component Inventory](#2-component-inventory)
3. [Model Provisioning](#3-model-provisioning)
4. [Deployment Topology](#4-deployment-topology)
5. [Network Configuration](#5-network-configuration)
6. [Operational Procedures](#6-operational-procedures)
7. [Compliance Mapping](#7-compliance-mapping)
8. [Limitations & Trade-offs](#8-limitations--trade-offs)

---

## 1. Air-Gapped Architecture

### Design Principles

| Principle                | Implementation                                                |
|--------------------------|---------------------------------------------------------------|
| **Zero external calls**  | No internet egress; all inference runs on local CPU servers   |
| **Self-contained models**| GGUF model files pre-loaded from internal registry            |
| **No GPU dependency**    | llama.cpp and BitNet run on commodity CPU hardware            |
| **No cloud provider**    | No AWS, Azure, or GCP API calls for any AI functionality      |
| **Data stays local**     | All prompts, responses, and metrics remain within the network |

### Architecture Comparison

```
  Standard Deployment                    Air-Gapped Deployment
  ────────────────────                   ──────────────────────

  ┌──────────┐                           ┌──────────┐
  │ Clario360│                           │ Clario360│
  │ Platform │                           │ Platform │
  └────┬─────┘                           └────┬─────┘
       │                                      │
  ┌────▼─────┐  ┌──────────┐            ┌────▼─────────────┐
  │ API GW   │──│ Internet │            │ Internal Network │
  └────┬─────┘  └────┬─────┘            │ (isolated)       │
       │              │                  └────┬─────────────┘
  ┌────▼────┐   ┌─────▼─────┐           ┌────▼──────────┐
  │ Cloud   │   │ OpenAI    │           │ llama.cpp     │
  │ GPU     │   │ Anthropic │           │ CPU Server    │
  │ (vLLM)  │   │ Azure     │           │ (local)       │
  └─────────┘   └───────────┘           │               │
                                        │ Pre-loaded    │
  External deps: 3+                     │ GGUF model    │
  GPU required: Yes                     └───────────────┘
  Internet: Required
                                        External deps: 0
                                        GPU required: No
                                        Internet: Not needed
```

---

## 2. Component Inventory

### Required Components (Air-Gapped)

| Component               | Purpose                            | Network Access    |
|-------------------------|------------------------------------|-------------------|
| Clario360 Backend       | API server, AI governance          | Internal only     |
| Clario360 Frontend      | Admin dashboard                    | Internal only     |
| PostgreSQL              | Data storage with RLS              | Internal only     |
| Redis                   | Caching and session storage        | Internal only     |
| llama.cpp Server        | CPU inference endpoint             | Internal only     |
| GGUF Model File(s)      | Pre-loaded AI models               | None (file-based) |

### Components NOT Required (Air-Gapped)

| Component               | Why Not Needed                                        |
|--------------------------|-------------------------------------------------------|
| GPU hardware             | CPU inference via llama.cpp / BitNet                  |
| CUDA / ROCm drivers      | No GPU = no GPU drivers                               |
| Cloud API keys           | No external LLM providers                             |
| Internet connectivity    | All components run locally                            |
| vLLM                     | GPU-optimised framework; not needed for CPU           |
| Model download URLs      | Models pre-loaded during provisioning                 |

---

## 3. Model Provisioning

### Model Transfer Process

Since the deployment environment has no internet, model files must be transferred via secure media:

```
┌───────────────┐     ┌──────────────┐     ┌───────────────────┐
│  Model Source  │     │   Secure     │     │  Air-Gapped       │
│  (Internet-   │────▶│   Transfer   │────▶│  Environment      │
│   connected)  │     │   Media      │     │                   │
│               │     │              │     │  /models/          │
│  Download     │     │  Encrypted   │     │    model.gguf     │
│  GGUF from    │     │  USB/disk    │     │                   │
│  HuggingFace  │     │  with chain  │     │  Verified via     │
│  or internal  │     │  of custody  │     │  SHA-256 hash     │
│  registry     │     │              │     │                   │
└───────────────┘     └──────────────┘     └───────────────────┘
```

### Recommended Models for Air-Gapped Deployment

| Model                              | Format | Size    | Backend    | Use Case                    |
|------------------------------------|--------|---------|------------|------------------------------|
| Llama 3.1 8B Instruct (Q4_0)      | GGUF   | ~4.7 GB | llama.cpp  | General security analysis    |
| Llama 3.1 8B Instruct (Q5_K_M)    | GGUF   | ~5.7 GB | llama.cpp  | Higher quality, slightly larger |
| Mistral 7B Instruct (Q4_0)        | GGUF   | ~4.1 GB | llama.cpp  | Efficient general purpose    |
| BitNet b1.58 3B                    | GGUF   | ~0.6 GB | BitNet/cpp | Minimal footprint, max efficiency |

### Model Integrity Verification

After transfer, verify model integrity:

1. **Pre-transfer**: Record SHA-256 hash of model file on source system
2. **Post-transfer**: Compute SHA-256 hash on target system
3. **Compare**: Hashes must match exactly
4. **Document**: Record hash, transfer date, and chain of custody in audit log

---

## 4. Deployment Topology

### Minimum Viable Air-Gapped Deployment

```
┌──────────────────────────────────────────────────────┐
│                  Air-Gapped Network                    │
│                                                        │
│  ┌─────────────────────────────────────┐              │
│  │           Server 1                   │              │
│  │           (8 CPU cores, 32 GB RAM)   │              │
│  │                                      │              │
│  │  ┌────────────┐  ┌───────────────┐  │              │
│  │  │ Clario360  │  │  PostgreSQL   │  │              │
│  │  │ Backend    │  │  + Redis      │  │              │
│  │  │ + Frontend │  │               │  │              │
│  │  └──────┬─────┘  └───────────────┘  │              │
│  │         │                            │              │
│  │  ┌──────▼─────────────────────────┐  │              │
│  │  │  llama.cpp Server              │  │              │
│  │  │  Port 8081                     │  │              │
│  │  │  Model: llama-3.1-8b-Q4_0     │  │              │
│  │  │  Threads: 4  |  Context: 4096 │  │              │
│  │  └────────────────────────────────┘  │              │
│  └─────────────────────────────────────┘              │
│                                                        │
│  Total servers: 1                                      │
│  Total GPUs: 0                                         │
│  Internet connections: 0                               │
│                                                        │
└──────────────────────────────────────────────────────┘
```

### Production Air-Gapped Deployment

```
┌──────────────────────────────────────────────────────────────┐
│                     Air-Gapped Network                        │
│                                                                │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │
│  │ App Server 1   │  │ App Server 2   │  │ DB Server      │  │
│  │ (HA pair)      │  │ (HA pair)      │  │                │  │
│  │                │  │                │  │ PostgreSQL     │  │
│  │ Clario360      │  │ Clario360      │  │ (primary)      │  │
│  │ Backend        │  │ Backend        │  │                │  │
│  │ + Frontend     │  │ + Frontend     │  │ Redis          │  │
│  └───────┬────────┘  └───────┬────────┘  └────────────────┘  │
│          │                   │                                │
│          └────────┬──────────┘                                │
│                   │                                           │
│  ┌────────────────▼───────────────────────────────────────┐  │
│  │              Inference Tier                             │  │
│  │                                                         │  │
│  │  ┌─────────────────┐  ┌─────────────────┐             │  │
│  │  │ Inference Srv 1 │  │ Inference Srv 2 │             │  │
│  │  │ (16 CPU, 64 GB) │  │ (8 CPU, 32 GB)  │             │  │
│  │  │                 │  │                 │             │  │
│  │  │ llama.cpp       │  │ llama.cpp       │             │  │
│  │  │ Llama 3.1 8B    │  │ BitNet 3B       │             │  │
│  │  │ Q5_K_M          │  │ 1-bit           │             │  │
│  │  │ High quality     │  │ Low latency     │             │  │
│  │  └─────────────────┘  └─────────────────┘             │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  Total servers: 5                                              │
│  Total GPUs: 0                                                 │
│  Internet connections: 0                                       │
│                                                                │
└──────────────────────────────────────────────────────────────┘
```

### Hardware Requirements

| Role              | Minimum Spec                         | Recommended Spec                   |
|-------------------|--------------------------------------|------------------------------------|
| Application       | 4 CPU, 16 GB RAM, 100 GB disk       | 8 CPU, 32 GB RAM, 200 GB SSD      |
| Database          | 4 CPU, 16 GB RAM, 200 GB SSD        | 8 CPU, 32 GB RAM, 500 GB SSD      |
| Inference (7–8B)  | 8 CPU, 16 GB RAM, 20 GB disk        | 16 CPU, 64 GB RAM, 50 GB SSD      |
| Inference (3B)    | 4 CPU, 8 GB RAM, 10 GB disk         | 8 CPU, 16 GB RAM, 20 GB SSD       |

---

## 5. Network Configuration

### Required Internal Network Paths

| Source            | Destination         | Port  | Protocol | Purpose                    |
|-------------------|---------------------|-------|----------|----------------------------|
| Frontend          | Backend             | 8080  | HTTPS    | API requests               |
| Backend           | PostgreSQL          | 5432  | TCP      | Database queries           |
| Backend           | Redis               | 6379  | TCP      | Cache and sessions         |
| Backend           | Inference Server(s) | 8080+ | HTTP(S)  | LLM inference requests     |
| Admin workstations| Frontend            | 443   | HTTPS    | Dashboard access           |

### Blocked Network Paths

| Direction  | Destination           | Reason                                       |
|------------|-----------------------|----------------------------------------------|
| Outbound   | Internet              | Air-gapped — no external access              |
| Outbound   | api.openai.com        | No cloud LLM dependencies                   |
| Outbound   | api.anthropic.com     | No cloud LLM dependencies                   |
| Inbound    | External networks     | No inbound connections from untrusted zones  |

---

## 6. Operational Procedures

### Model Updates

Since the environment is air-gapped, model updates follow a controlled process:

1. **Evaluate** — test new model version in a connected environment first
2. **Benchmark** — run full benchmark suite to verify performance meets requirements
3. **Approve** — security review and change advisory board approval
4. **Transfer** — copy GGUF file to secure transfer media with chain of custody
5. **Deploy** — load model file into inference server, verify SHA-256 hash
6. **Validate** — run smoke test benchmark to confirm functionality
7. **Cutover** — update Clario360 server registry to point to new model

### Health Monitoring

| Check                    | Method                        | Frequency | Alert Threshold         |
|--------------------------|-------------------------------|-----------|-------------------------|
| Inference server health  | GET /health                   | 15 sec    | 3 consecutive failures  |
| Inference latency        | Prometheus metric             | Continuous| p95 > 5 seconds         |
| Disk space (models)      | System monitoring             | 5 min     | < 20% free              |
| CPU utilisation          | System monitoring             | 1 min     | Sustained > 90%         |
| Memory utilisation       | System monitoring             | 1 min     | Sustained > 85%         |

### Backup & Recovery

| Component       | Backup Strategy                           | RTO        | RPO        |
|-----------------|-------------------------------------------|------------|------------|
| PostgreSQL      | Daily full + WAL archiving                | 1 hour     | 5 minutes  |
| Model files     | Read-only on shared storage               | 15 minutes | N/A (static)|
| Configuration   | Version-controlled Helm values            | 30 minutes | Latest commit|
| Benchmark data  | Included in PostgreSQL backup             | 1 hour     | 5 minutes  |

---

## 7. Compliance Mapping

### How Air-Gapped CPU Inference Supports Compliance

| Framework       | Requirement                              | How Clario360 Meets It                          |
|-----------------|------------------------------------------|-------------------------------------------------|
| **FedRAMP**     | Data must remain in authorised boundary  | All processing local; no external API calls     |
| **NIST 800-53** | SC-7: Boundary protection                | No internet egress; internal-only communication |
| **NIST 800-53** | SC-28: Protection of information at rest | Models encrypted at rest on local storage       |
| **SOC 2**       | CC6.1: Logical access controls           | RLS tenant isolation; JWT authentication        |
| **SOC 2**       | CC6.6: External system boundaries        | No external system dependencies                 |
| **ITAR**        | No foreign access to controlled data     | Fully on-premises; no cloud provider access     |
| **ISO 27001**   | A.13.1: Network security management      | Air-gapped network; no external communication   |
| **HIPAA**       | 164.312(e): Transmission security        | No data transmitted outside the network boundary|
| **GDPR**        | Art. 44: Transfer to third countries     | No data leaves the deployment boundary          |

### Data Residency Guarantee

In an air-gapped deployment:
- Prompts **never leave** the local network
- Model responses **never leave** the local network
- Benchmark results are stored in **local PostgreSQL only**
- No telemetry, analytics, or usage data is transmitted externally
- Model files are static artifacts with **no phone-home capability**

---

## 8. Limitations & Trade-offs

### Performance Considerations

| Aspect              | Air-Gapped Impact                                     | Mitigation                                  |
|---------------------|-------------------------------------------------------|---------------------------------------------|
| Inference speed     | CPU is 2–5x slower than GPU                           | Acceptable for batch/async workloads        |
| Model size          | Limited by available RAM                              | Use quantised models (Q4_0 = ~5 GB for 8B) |
| Context window      | Limited by CPU memory bandwidth                       | 4096 tokens standard; 8192 with sufficient RAM |
| Concurrent users    | Lower throughput than GPU                             | Scale horizontally with multiple CPU servers|
| Model variety       | Only GGUF-format models supported                     | Most popular models available in GGUF       |

### Operational Considerations

| Aspect              | Air-Gapped Impact                                     | Mitigation                                  |
|---------------------|-------------------------------------------------------|---------------------------------------------|
| Model updates       | Manual transfer process required                      | Quarterly update cycle with change control  |
| No automatic patches| Security patches require manual intervention          | Maintain a patch pipeline with secure media |
| Limited model choice| Cannot dynamically download new models                | Pre-qualify a model catalog for the environment|
| No cloud fallback   | If CPU server fails, no external backup               | Deploy redundant inference servers          |

---

*For benchmark guidance in air-gapped environments, see the [Benchmark Playbook](02_BENCHMARK_PLAYBOOK.md). For security details, see the [Security & Compliance Addendum](07_SECURITY_COMPLIANCE_ADDENDUM.md).*
