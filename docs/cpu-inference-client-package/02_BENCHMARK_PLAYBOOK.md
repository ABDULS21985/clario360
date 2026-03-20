# Benchmark Playbook — CPU vs GPU Inference Testing

| Attribute    | Value                                           |
|--------------|-------------------------------------------------|
| **For**      | DevOps Engineers, ML Engineers, Platform Admins |
| **Date**     | 2026-03-14                                      |
| **Status**   | Phase 1 Delivered                               |
| **Read time**| 10 minutes                                      |

---

## Purpose

This playbook provides a step-by-step guide for running CPU vs GPU inference benchmarks using Clario360's benchmarking framework. It covers environment setup, suite configuration, benchmark execution, result interpretation, and migration decision-making.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Step 1 — Register Inference Servers](#step-1--register-inference-servers)
3. [Step 2 — Create a Benchmark Suite](#step-2--create-a-benchmark-suite)
4. [Step 3 — Execute Benchmarks](#step-3--execute-benchmarks)
5. [Step 4 — Interpret Results](#step-4--interpret-results)
6. [Step 5 — Compare Runs](#step-5--compare-runs)
7. [Step 6 — Estimate Cost Savings](#step-6--estimate-cost-savings)
8. [Recommended Test Configurations](#recommended-test-configurations)
9. [Troubleshooting](#troubleshooting)

---

## 1. Prerequisites

Before running benchmarks, ensure the following are in place:

| Prerequisite                    | Details                                                     |
|---------------------------------|-------------------------------------------------------------|
| Clario360 platform running      | Backend API and frontend accessible                        |
| At least one inference server   | llama.cpp, vLLM, BitNet, or ONNX server running            |
| Server health verified          | Server responds to its `/health` endpoint                   |
| Admin access                    | User with AI governance permissions                         |
| Model loaded on server          | GGUF or model file loaded and serving completions           |

### Verifying Server Availability

Before registering a server in Clario360, confirm it responds to the OpenAI-compatible API:

```
curl -s http://<server-url>/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-name>","messages":[{"role":"user","content":"Hello"}],"max_tokens":10}'
```

A successful response confirms the server is ready for benchmarking.

---

## Step 1 — Register Inference Servers

Navigate to **Admin → AI Governance → Compute Infrastructure** and register each server you want to benchmark.

### What to Register

For a meaningful CPU vs GPU comparison, register at least two servers:

| Server Example       | Backend Type  | Purpose                |
|----------------------|---------------|------------------------|
| `llamacpp-cpu-dev`   | llama.cpp CPU | CPU inference baseline |
| `vllm-gpu-prod`     | vLLM GPU      | GPU inference baseline |

### Registration Fields

| Field             | Guidance                                                       |
|-------------------|----------------------------------------------------------------|
| **Name**          | Descriptive name (e.g., `llamacpp-cpu-bench-01`)              |
| **Backend Type**  | Select from 8 options; must match the actual server runtime    |
| **Base URL**      | Full URL to the `/v1` endpoint (e.g., `http://10.0.1.50:8080/v1`) |
| **Health Endpoint**| Usually `/health` — used for liveness monitoring              |
| **Model Name**    | The model identifier served by this endpoint                   |
| **Quantization**  | If applicable (e.g., `Q4_0`, `Q5_K_M`, `Q8_0`)              |
| **CPU Cores**     | Number of CPU cores allocated to this server                   |
| **Memory (MB)**   | RAM allocated to this server                                   |
| **Max Concurrent**| Maximum parallel requests the server supports                  |

After registration, the server will appear with **Provisioning** status. Verify connectivity and mark it as **Healthy** when confirmed.

---

## Step 2 — Create a Benchmark Suite

Navigate to **Admin → AI Governance → Benchmarks** and click **New Suite**.

### Suite Configuration Guide

| Parameter          | Recommended Value | Guidance                                                |
|--------------------|-------------------|---------------------------------------------------------|
| **Name**           | Descriptive       | Include model and test type (e.g., `threat-scorer-latency-test`) |
| **Model Slug**     | Match your model  | Must match the model slug in your AI governance registry |
| **Warmup Count**   | 5–10              | Enough to warm JIT, caches, and model loading           |
| **Iteration Count**| 50–200            | More iterations = more statistical confidence           |
| **Concurrency**    | 1–4               | Start at 1, increase to test throughput under load      |
| **Timeout (sec)**  | 30–120            | Higher for CPU/BitNet servers; lower for GPU            |
| **Description**    | Optional          | Note what you're testing and why                        |

### Prompt Dataset

After creating the suite, you can add prompts via the API. A good prompt dataset should:

- **Represent real workloads** — use actual prompts from your production use cases
- **Vary in length** — include short (1-2 sentences) and long (paragraph) inputs
- **Cover different tasks** — threat analysis, classification, summarisation, etc.
- **Include 10–50 prompts** — enough diversity to avoid overfitting to one pattern

---

## Step 3 — Execute Benchmarks

From the Suites tab, find your suite and click **Run**. Select the target inference server and confirm.

### Execution Sequence

1. **Run created** — status: `pending`
2. **Warmup phase** — N warmup iterations executed and discarded
3. **Measured phase** — configured iterations run with concurrency control
4. **Aggregation** — latency percentiles, throughput, and quality metrics computed
5. **Results stored** — status: `completed` (or `failed` with error message)

### Running Against Multiple Servers

To compare CPU vs GPU, execute the **same suite** against **each registered server**:

1. Run suite against `llamacpp-cpu-dev` → produces Run A
2. Run suite against `vllm-gpu-prod` → produces Run B
3. Optionally run against `bitnet-cpu-dev` → produces Run C

This gives you directly comparable results since the prompts, iteration count, and configuration are identical.

---

## Step 4 — Interpret Results

Each completed run provides metrics across five dimensions:

### Latency Metrics

| Metric        | What It Tells You                                            |
|---------------|--------------------------------------------------------------|
| **p50**       | Median latency — "typical" user experience                   |
| **p95**       | 95th percentile — worst case for most users                  |
| **p99**       | 99th percentile — tail latency for SLA planning              |
| **Average**   | Mean latency — useful for cost calculations                  |
| **Min / Max** | Range — indicates consistency                                |

**Rule of thumb**: Focus on **p95** for production decisions. If CPU p95 is within 3x of GPU p95, CPU is generally viable for non-real-time workloads.

### Throughput Metrics

| Metric              | What It Tells You                                     |
|---------------------|-------------------------------------------------------|
| **Tokens/second**   | How fast the model generates output                   |
| **Requests/second** | How many concurrent users can be served               |
| **Failed requests** | Reliability indicator — should be 0 or near-0         |
| **Retried requests**| Transient failure recovery — indicates server pressure |

### Streaming Metrics (when enabled)

| Metric         | What It Tells You                                        |
|----------------|----------------------------------------------------------|
| **TTFT p50**   | Time to first token — perceived responsiveness           |
| **TTFT p95**   | Worst-case first-token latency                           |
| **TTFT avg**   | Average first-token latency                              |

**TTFT is critical for interactive workloads** — users perceive responsiveness based on when the first token appears, not when the full response completes.

### Quality Metrics

| Metric                   | What It Tells You                               |
|--------------------------|-------------------------------------------------|
| **Semantic similarity**  | How close CPU responses are to GPU responses    |
| **BLEU / ROUGE-L**      | N-gram overlap with reference outputs           |
| **Factual accuracy**     | Correctness of factual claims                   |
| **Perplexity**           | Model confidence — lower is better              |

### Resource Utilisation

| Metric             | What It Tells You                                    |
|--------------------|------------------------------------------------------|
| **Peak CPU %**     | Maximum CPU usage during benchmark                   |
| **Peak Memory MB** | Maximum RAM consumption                              |
| **Avg CPU / Mem**  | Sustained resource usage for capacity planning       |

---

## Step 5 — Compare Runs

Navigate to the suite detail page, select two or more completed runs using the checkboxes, and click **Compare Selected**.

### Comparison Output

| Metric                | Interpretation                                          |
|-----------------------|---------------------------------------------------------|
| **Cost delta**        | Monthly cost difference (negative = CPU saves money)   |
| **Latency delta %**   | p95 latency difference as percentage                   |
| **Quality delta %**   | Semantic similarity difference as percentage           |
| **Recommendation**    | Automated viability assessment                         |

### Recommendation Guide

| Recommendation     | What It Means                                           | Action                          |
|--------------------|---------------------------------------------------------|---------------------------------|
| **CPU Viable**     | Latency < 3x GPU, quality loss < 10%                  | Safe to migrate batch workloads |
| **GPU Required**   | Latency ≥ 3x GPU                                       | Keep GPU for this workload      |
| **Needs More Data**| Results inconclusive                                    | Run more iterations or prompts  |

---

## Step 6 — Estimate Cost Savings

Use the cost estimation API to calculate projected monthly savings:

**Via API**: `POST /api/ai/compute-costs/estimate`

Provide a CPU run ID and GPU run ID. The system returns:
- Monthly cost for each backend
- Monthly savings amount and percentage
- Latency trade-off percentage
- Throughput comparison

### Decision Matrix

| Latency Delta | Quality Delta | Cost Savings | Decision                                   |
|---------------|---------------|--------------|---------------------------------------------|
| < 2x slower   | < 5% loss     | > 50%        | **Migrate to CPU** — clear win              |
| 2–3x slower   | < 10% loss    | > 70%        | **Migrate batch workloads** to CPU          |
| 3–5x slower   | < 10% loss    | > 80%        | **Migrate async workloads** only            |
| > 5x slower   | Any           | Any          | **Keep on GPU** — latency unacceptable      |
| Any           | > 15% loss    | Any          | **Keep on GPU** — quality degradation too high |

---

## Recommended Test Configurations

### Quick Validation (5 minutes)

| Parameter      | Value  |
|----------------|--------|
| Warmup         | 3      |
| Iterations     | 20     |
| Concurrency    | 1      |
| Prompts        | 5      |

Good for: initial smoke test, verifying server connectivity.

### Standard Benchmark (15–30 minutes)

| Parameter      | Value  |
|----------------|--------|
| Warmup         | 5      |
| Iterations     | 100    |
| Concurrency    | 1      |
| Prompts        | 20     |

Good for: baseline latency measurement, quality assessment.

### Production Simulation (1–2 hours)

| Parameter      | Value  |
|----------------|--------|
| Warmup         | 10     |
| Iterations     | 500    |
| Concurrency    | 4      |
| Prompts        | 50     |

Good for: throughput testing, resource utilisation analysis, final migration decision.

### Stress Test (2–4 hours)

| Parameter      | Value  |
|----------------|--------|
| Warmup         | 10     |
| Iterations     | 1000   |
| Concurrency    | 8      |
| Prompts        | 50     |

Good for: finding breaking points, maximum throughput, stability testing.

---

## Troubleshooting

| Issue                          | Likely Cause                           | Resolution                                    |
|--------------------------------|----------------------------------------|-----------------------------------------------|
| Run fails immediately          | Server unreachable or model not loaded | Check server health endpoint; verify base URL |
| All requests timeout           | Timeout too low for CPU server         | Increase timeout to 120s for CPU backends     |
| High failure rate (> 10%)      | Server overloaded or under-resourced   | Reduce concurrency; add CPU/memory resources  |
| Quality metrics empty          | No reference outputs for comparison    | Add `expected_reference` to prompt dataset    |
| Inconsistent latency           | Too few iterations                     | Increase iteration count to 200+              |
| TTFT metrics empty             | Stream not enabled on suite or server  | Enable streaming on suite and ensure server is stream-capable |
| Server shows "provisioning"    | Not manually confirmed after register  | Use "Mark Healthy" button after verifying connectivity |

---

*For API details, see the [Technical Architecture](../architecture/CPU_INFERENCE_ARCHITECTURE.md). For cost analysis, see the [ROI & TCO Analysis](03_ROI_TCO_ANALYSIS.md).*
