# API Integration Guide — Compute & Benchmark APIs

| Attribute    | Value                                                  |
|--------------|--------------------------------------------------------|
| **For**      | Developer Partners, DevOps, CI/CD Engineers            |
| **Date**     | 2026-03-14                                             |
| **Status**   | Phase 1 Delivered                                      |
| **Read time**| 10 minutes                                             |

---

## Purpose

This guide describes how to integrate Clario360's compute infrastructure and benchmarking APIs into automation workflows, CI/CD pipelines, and external systems. It covers authentication, endpoint reference, common workflows, and error handling.

---

## Table of Contents

1. [Authentication](#1-authentication)
2. [Base URL & Headers](#2-base-url--headers)
3. [API Endpoint Reference](#3-api-endpoint-reference)
4. [Common Integration Workflows](#4-common-integration-workflows)
5. [Pagination](#5-pagination)
6. [Error Handling](#6-error-handling)
7. [Webhook & Event Integration](#7-webhook--event-integration)
8. [CI/CD Pipeline Integration](#8-cicd-pipeline-integration)

---

## 1. Authentication

All API requests require a valid JWT bearer token obtained via the Clario360 authentication flow.

### Obtaining a Token

| Step | Action                                                       |
|------|--------------------------------------------------------------|
| 1    | POST to `/api/v1/auth/login` with email and password        |
| 2    | Receive access token (short-lived) and refresh token        |
| 3    | Include access token in all subsequent requests              |
| 4    | When token expires, use refresh token to obtain a new one    |

### Token Usage

All API calls must include the `Authorization` header:

```
Authorization: Bearer <access_token>
```

The token carries the tenant context — the API automatically scopes all data operations to the authenticated tenant.

---

## 2. Base URL & Headers

| Parameter         | Value                                         |
|-------------------|-----------------------------------------------|
| **Base URL**      | `https://<clario360-host>/api/ai`             |
| **Content-Type**  | `application/json`                             |
| **Authorization** | `Bearer <jwt_token>`                          |
| **Response format**| JSON envelope: `{ "data": ..., "meta": ... }` |

---

## 3. API Endpoint Reference

### Inference Servers

| Operation           | Method   | Path                                  | Purpose                                |
|---------------------|----------|---------------------------------------|----------------------------------------|
| List servers        | `GET`    | `/inference-servers`                  | Get all registered servers (paginated) |
| Create server       | `POST`   | `/inference-servers`                  | Register a new inference endpoint      |
| Get server          | `GET`    | `/inference-servers/{serverId}`       | Get server details by ID               |
| Update server       | `PUT`    | `/inference-servers/{serverId}`       | Update server configuration            |
| Update status       | `PUT`    | `/inference-servers/{serverId}/status`| Change server status                   |
| Delete server       | `DELETE` | `/inference-servers/{serverId}`       | Decommission (soft delete)             |

#### Create Server — Request Body

| Field              | Type     | Required | Description                                   |
|--------------------|----------|----------|-----------------------------------------------|
| `name`             | string   | Yes      | Unique name within tenant                     |
| `backend_type`     | string   | Yes      | One of: `inline_go`, `vllm_gpu`, `vllm_cpu`, `llamacpp_cpu`, `llamacpp_gpu`, `bitnet_cpu`, `onnx_cpu`, `onnx_gpu` |
| `base_url`         | string   | Yes      | URL to `/v1` endpoint                         |
| `health_endpoint`  | string   | No       | Health check path (default: `/health`)        |
| `model_name`       | string   | No       | Model identifier served by this endpoint      |
| `quantization`     | string   | No       | Quantization format (e.g., `Q4_0`)           |
| `cpu_cores`        | integer  | No       | CPU cores allocated                           |
| `memory_mb`        | integer  | No       | Memory in MB allocated                        |
| `gpu_type`         | string   | No       | GPU model (e.g., `A100`)                     |
| `gpu_count`        | integer  | No       | Number of GPUs (default: 0)                   |
| `max_concurrent`   | integer  | No       | Max parallel requests (default: 1)            |
| `stream_capable`   | boolean  | No       | Whether server supports SSE streaming         |
| `metadata`         | object   | No       | Arbitrary metadata JSON                       |

#### Update Server Status — Request Body

| Field    | Type   | Required | Values                                                       |
|----------|--------|----------|--------------------------------------------------------------|
| `status` | string | Yes      | `provisioning`, `healthy`, `degraded`, `offline`, `decommissioned` |

### Benchmark Suites

| Operation           | Method   | Path                                        | Purpose                         |
|---------------------|----------|---------------------------------------------|---------------------------------|
| List suites         | `GET`    | `/benchmarks/suites`                        | Get all suites (paginated)      |
| Create suite        | `POST`   | `/benchmarks/suites`                        | Create a new benchmark suite    |
| Get suite           | `GET`    | `/benchmarks/suites/{suiteId}`              | Get suite details by ID         |
| Update suite        | `PUT`    | `/benchmarks/suites/{suiteId}`              | Update suite configuration      |
| Delete suite        | `DELETE` | `/benchmarks/suites/{suiteId}`              | Delete suite (if no active runs)|
| Run benchmark       | `POST`   | `/benchmarks/suites/{suiteId}/run`          | Execute benchmark on a server   |

#### Create Suite — Request Body

| Field              | Type          | Required | Description                            |
|--------------------|---------------|----------|----------------------------------------|
| `name`             | string        | Yes      | Suite name                             |
| `description`      | string        | No       | Description of what this suite tests   |
| `model_slug`       | string        | Yes      | Target model to benchmark              |
| `prompt_dataset`   | array         | No       | Array of `{system_prompt, user_message}` objects |
| `warmup_count`     | integer       | No       | Warmup iterations (default: 5)         |
| `iteration_count`  | integer       | No       | Measured iterations (default: 100)     |
| `concurrency`      | integer       | No       | Parallel requests (default: 1)         |
| `timeout_seconds`  | integer       | No       | Per-request timeout (default: 60)      |
| `stream_enabled`   | boolean       | No       | Enable SSE streaming (default: false)  |
| `max_retries`      | integer       | No       | Retry count for failures (default: 3)  |

#### Run Benchmark — Request Body

| Field       | Type   | Required | Description                           |
|-------------|--------|----------|---------------------------------------|
| `server_id` | string | Yes      | UUID of the target inference server   |

### Benchmark Runs

| Operation           | Method   | Path                                  | Purpose                              |
|---------------------|----------|---------------------------------------|--------------------------------------|
| List runs           | `GET`    | `/benchmarks/runs`                    | Get all runs (paginated, filterable) |
| Get run             | `GET`    | `/benchmarks/runs/{runId}`            | Get full run results                 |
| Compare runs        | `POST`   | `/benchmarks/runs/compare`            | Compare 2–10 runs side-by-side       |

#### List Runs — Query Parameters

| Parameter   | Type   | Description                           |
|-------------|--------|---------------------------------------|
| `suite_id`  | string | Filter runs by suite ID               |
| `page`      | integer| Page number (default: 1)              |
| `per_page`  | integer| Results per page (default: 20)        |

#### Compare Runs — Request Body

| Field     | Type     | Required | Description                                |
|-----------|----------|----------|--------------------------------------------|
| `run_ids` | string[] | Yes      | Array of 2–10 run UUIDs to compare         |

### Cost Models

| Operation           | Method   | Path                          | Purpose                              |
|---------------------|----------|-------------------------------|--------------------------------------|
| List cost models    | `GET`    | `/compute-costs`              | Get all cost models                  |
| Create cost model   | `POST`   | `/compute-costs`              | Define pricing for a backend config  |
| Estimate savings    | `POST`   | `/compute-costs/estimate`     | Compare CPU vs GPU monthly costs     |

#### Create Cost Model — Request Body

| Field                | Type    | Required | Description                                 |
|----------------------|---------|----------|---------------------------------------------|
| `name`               | string  | Yes      | Name for this cost configuration            |
| `backend_type`       | string  | Yes      | Backend type this pricing applies to        |
| `instance_type`      | string  | Yes      | Cloud instance or hardware description      |
| `hourly_cost_usd`    | number  | Yes      | Hourly cost in USD                          |
| `cpu_cores`          | integer | No       | CPU cores for this configuration            |
| `memory_gb`          | integer | No       | Memory in GB                                |
| `gpu_type`           | string  | No       | GPU model name                              |
| `gpu_count`          | integer | No       | Number of GPUs (default: 0)                 |
| `max_tokens_per_second`| number| No       | Maximum throughput capacity                 |
| `notes`              | string  | No       | Additional notes                            |

#### Estimate Savings — Request Body

| Field        | Type   | Required | Description                         |
|--------------|--------|----------|-------------------------------------|
| `cpu_run_id` | string | Yes      | UUID of a completed CPU benchmark run |
| `gpu_run_id` | string | Yes      | UUID of a completed GPU benchmark run |

---

## 4. Common Integration Workflows

### Workflow 1: Automated Benchmark After Model Deployment

Use this workflow to automatically benchmark a model after deploying it to a new inference server.

```
1. Register server    POST /inference-servers
       ↓                    → returns server_id
2. Create suite       POST /benchmarks/suites
       ↓                    → returns suite_id
3. Execute benchmark  POST /benchmarks/suites/{suiteId}/run
       ↓                    body: { server_id: <server_id> }
4. Poll for completion GET /benchmarks/runs/{runId}
       ↓                    → wait until status = "completed"
5. Check results      Read p95_latency_ms, tokens_per_second
       ↓
6. Decision           If p95 < threshold → mark server healthy
                      If p95 > threshold → mark server degraded
```

### Workflow 2: Nightly CPU vs GPU Regression Test

```
1. List servers       GET /inference-servers?status=healthy
       ↓
2. For each server:
   a. Run benchmark   POST /benchmarks/suites/{suiteId}/run
   b. Wait for result GET /benchmarks/runs/{runId}
       ↓
3. Compare runs       POST /benchmarks/runs/compare
       ↓                    body: { run_ids: [cpu_run, gpu_run] }
4. Evaluate           Check recommendation field
       ↓
5. Alert if needed    If recommendation changed from previous run
```

### Workflow 3: Cost Monitoring Pipeline

```
1. Run benchmarks     (from Workflow 2)
       ↓
2. Estimate savings   POST /compute-costs/estimate
       ↓                    body: { cpu_run_id, gpu_run_id }
3. Record metrics     Store monthly_savings in monitoring system
       ↓
4. Dashboard          Display cost trends over time
```

---

## 5. Pagination

All list endpoints use consistent pagination:

### Request Parameters

| Parameter   | Default | Description                    |
|-------------|---------|--------------------------------|
| `page`      | 1       | Page number (1-indexed)        |
| `per_page`  | 20      | Items per page (max: 100)      |

### Response Envelope

```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 47,
    "total_pages": 3
  }
}
```

To iterate through all pages, increment `page` until `page > total_pages`.

---

## 6. Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "benchmark suite model_slug is required",
    "details": null
  }
}
```

### Common Error Codes

| HTTP Status | Error Code          | Meaning                                       |
|-------------|---------------------|-----------------------------------------------|
| 400         | `INVALID_INPUT`     | Request body validation failed                |
| 400         | `INVALID_ID`        | UUID parameter is malformed                   |
| 401         | `UNAUTHORIZED`      | Missing or invalid JWT token                  |
| 403         | `FORBIDDEN`         | Insufficient permissions                      |
| 404         | `NOT_FOUND`         | Resource doesn't exist or belongs to another tenant |
| 409         | `CONFLICT`          | Duplicate name or conflicting state            |
| 422         | `UNPROCESSABLE`     | Semantic validation failed (e.g., deleting suite with active runs) |
| 500         | `INTERNAL_ERROR`    | Server-side error                             |

### Retry Guidance

| Status Code | Retry? | Guidance                                        |
|-------------|--------|-------------------------------------------------|
| 400         | No     | Fix the request body                            |
| 401         | Once   | Refresh the access token and retry              |
| 404         | No     | Resource doesn't exist                          |
| 429         | Yes    | Back off and retry after delay                  |
| 500         | Yes    | Retry with exponential backoff (max 3 attempts) |
| 502/503     | Yes    | Service temporarily unavailable; retry          |

---

## 7. Webhook & Event Integration

### Prometheus Metrics (Pull-Based)

Clario360 exposes Prometheus metrics that external monitoring systems can scrape:

| Metric                           | Type      | Use Case                              |
|----------------------------------|-----------|---------------------------------------|
| `ai_benchmark_runs_total`        | Counter   | Track benchmark execution volume      |
| `ai_benchmark_latency_seconds`   | Histogram | Alert on latency regressions          |
| `ai_inference_server_health`     | Gauge     | Monitor server availability           |
| `ai_compute_cost_per_token`      | Gauge     | Track cost trends                     |

### Integration with External Systems

| System          | Integration Method                                       |
|-----------------|----------------------------------------------------------|
| Grafana         | Prometheus data source → custom dashboard               |
| PagerDuty       | Prometheus alertmanager → PagerDuty webhook             |
| Slack           | Prometheus alertmanager → Slack webhook                 |
| Jira            | API polling → create tickets for failed benchmarks      |
| CI/CD (Jenkins) | API calls in pipeline stages (see below)                |
| CI/CD (GitLab)  | API calls in `.gitlab-ci.yml` stages                    |

---

## 8. CI/CD Pipeline Integration

### Use Case: Benchmark Gate in Deployment Pipeline

Add a benchmark validation step to your deployment pipeline that prevents deploying model updates if CPU inference quality drops below acceptable thresholds.

### Pipeline Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Build   │────▶│  Deploy  │────▶│Benchmark │────▶│  Gate    │
│  Model   │     │  to Test │     │  Suite   │     │ Decision │
│          │     │  Server  │     │  Run     │     │          │
└──────────┘     └──────────┘     └──────────┘     └────┬─────┘
                                                        │
                                                   ┌────▼────┐
                                              Pass │         │ Fail
                                              ┌────▼──┐ ┌────▼──┐
                                              │Deploy │ │Reject │
                                              │to Prod│ │+ Alert│
                                              └───────┘ └───────┘
```

### Gate Criteria

| Metric              | Pass Threshold            | Fail Action                    |
|---------------------|---------------------------|--------------------------------|
| p95 latency         | < 3× baseline GPU         | Block deployment               |
| Failed requests     | < 5% of total             | Block deployment               |
| Tokens per second   | > 50% of GPU throughput   | Warning (non-blocking)         |
| Quality delta       | < 10% degradation         | Block deployment               |

### Example: Polling for Benchmark Completion

After triggering a benchmark run, poll for completion:

```
1. POST /benchmarks/suites/{id}/run → get run_id
2. Loop:
   a. GET /benchmarks/runs/{run_id}
   b. If status = "completed" → check metrics
   c. If status = "failed" → fail pipeline
   d. If status = "running" → wait 10 seconds, retry
   e. Timeout after 30 minutes → fail pipeline
3. Evaluate: compare metrics against gate thresholds
4. Decision: pass or fail the pipeline stage
```

---

*For endpoint details with full request/response schemas, see the [Technical Architecture](../architecture/CPU_INFERENCE_ARCHITECTURE.md). For benchmark configuration guidance, see the [Benchmark Playbook](02_BENCHMARK_PLAYBOOK.md).*
