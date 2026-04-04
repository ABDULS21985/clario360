-- Compute backend infrastructure for CPU/GPU inference benchmarking.
-- Phase 1: inference server registry, benchmark suites/runs, cost models.
-- Includes streaming SSE support, retry tracking, and TTFT metrics.

-- Extend artifact_type to support new model formats.
ALTER TABLE ai_model_versions DROP CONSTRAINT IF EXISTS ai_model_versions_artifact_type_check;
ALTER TABLE ai_model_versions
    ADD CONSTRAINT ai_model_versions_artifact_type_check
    CHECK (artifact_type IN (
        'go_function', 'rule_set', 'statistical_config', 'template_config',
        'serialized_model', 'gguf_model', 'bitnet_model', 'onnx_model'
    ));

-- ═══════════════════════════════════════════════════════════════════════
-- Inference server registry
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ai_inference_servers (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    backend_type    TEXT        NOT NULL
                    CHECK (backend_type IN (
                        'inline_go', 'vllm_gpu', 'vllm_cpu',
                        'llamacpp_cpu', 'llamacpp_gpu',
                        'bitnet_cpu', 'onnx_cpu', 'onnx_gpu'
                    )),
    base_url        TEXT        NOT NULL,
    health_endpoint TEXT        NOT NULL DEFAULT '/health',
    model_name      TEXT,
    api_key         TEXT,
    quantization    TEXT,
    status          TEXT        NOT NULL DEFAULT 'provisioning'
                    CHECK (status IN ('provisioning', 'healthy', 'degraded', 'offline', 'decommissioned')),
    cpu_cores       INT,
    memory_mb       INT,
    gpu_type        TEXT,
    gpu_count       INT         NOT NULL DEFAULT 0,
    max_concurrent  INT         NOT NULL DEFAULT 1,
    stream_capable  BOOLEAN     NOT NULL DEFAULT FALSE,
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_inference_servers_tenant_name
    ON ai_inference_servers (tenant_id, name)
    WHERE status <> 'decommissioned';

CREATE INDEX idx_inference_servers_tenant_status
    ON ai_inference_servers (tenant_id, status);

-- Trigger for updated_at.
CREATE TRIGGER trg_ai_inference_servers_updated_at
    BEFORE UPDATE ON ai_inference_servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- RLS for inference servers.
ALTER TABLE ai_inference_servers ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_inference_servers_tenant_isolation ON ai_inference_servers
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- Benchmark suite definition
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ai_benchmark_suites (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    model_slug      TEXT        NOT NULL,
    prompt_dataset  JSONB       NOT NULL DEFAULT '[]'::jsonb,
    dataset_size    INT         NOT NULL DEFAULT 0,
    warmup_count    INT         NOT NULL DEFAULT 5,
    iteration_count INT         NOT NULL DEFAULT 100,
    concurrency     INT         NOT NULL DEFAULT 1,
    timeout_seconds INT         NOT NULL DEFAULT 60,
    stream_enabled  BOOLEAN     NOT NULL DEFAULT FALSE,
    max_retries     INT         NOT NULL DEFAULT 3,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ
);

CREATE INDEX idx_benchmark_suites_tenant
    ON ai_benchmark_suites (tenant_id, created_at DESC);

ALTER TABLE ai_benchmark_suites ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_benchmark_suites_tenant_isolation ON ai_benchmark_suites
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- Individual benchmark run
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ai_benchmark_runs (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    suite_id                  UUID        NOT NULL REFERENCES ai_benchmark_suites(id) ON DELETE CASCADE,
    server_id                 UUID        NOT NULL REFERENCES ai_inference_servers(id),
    backend_type              TEXT        NOT NULL,
    model_name                TEXT        NOT NULL,
    quantization              TEXT,
    status                    TEXT        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    stream_used               BOOLEAN     NOT NULL DEFAULT FALSE,

    -- Latency metrics (milliseconds).
    p50_latency_ms            DOUBLE PRECISION,
    p95_latency_ms            DOUBLE PRECISION,
    p99_latency_ms            DOUBLE PRECISION,
    avg_latency_ms            DOUBLE PRECISION,
    min_latency_ms            DOUBLE PRECISION,
    max_latency_ms            DOUBLE PRECISION,

    -- Throughput.
    tokens_per_second         DOUBLE PRECISION,
    requests_per_second       DOUBLE PRECISION,
    total_tokens              BIGINT,
    total_requests            INT,
    failed_requests           INT         DEFAULT 0,
    retried_requests          INT         DEFAULT 0,

    -- Time-to-first-token (populated only when stream_used = true).
    p50_ttft_ms               DOUBLE PRECISION,
    p95_ttft_ms               DOUBLE PRECISION,
    avg_ttft_ms               DOUBLE PRECISION,

    -- Quality metrics.
    avg_perplexity            DOUBLE PRECISION,
    bleu_score                DOUBLE PRECISION,
    rouge_l_score             DOUBLE PRECISION,
    semantic_similarity       DOUBLE PRECISION,
    factual_accuracy          DOUBLE PRECISION,

    -- Resource utilization.
    peak_cpu_percent          DOUBLE PRECISION,
    peak_memory_mb            INT,
    avg_cpu_percent           DOUBLE PRECISION,
    avg_memory_mb             INT,

    -- Cost.
    estimated_hourly_cost_usd DOUBLE PRECISION,
    cost_per_1k_tokens_usd   DOUBLE PRECISION,

    -- Timing.
    started_at                TIMESTAMPTZ,
    completed_at              TIMESTAMPTZ,
    duration_seconds          INT,
    error_message             TEXT,
    raw_results               JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_by                UUID        NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_benchmark_runs_suite
    ON ai_benchmark_runs (suite_id, created_at DESC);

CREATE INDEX idx_benchmark_runs_server
    ON ai_benchmark_runs (server_id, created_at DESC);

CREATE INDEX idx_benchmark_runs_tenant
    ON ai_benchmark_runs (tenant_id, status, created_at DESC);

ALTER TABLE ai_benchmark_runs ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_benchmark_runs_tenant_isolation ON ai_benchmark_runs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- ═══════════════════════════════════════════════════════════════════════
-- Cost model definitions for CPU vs GPU comparison
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ai_compute_cost_models (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                  TEXT        NOT NULL,
    backend_type          TEXT        NOT NULL,
    instance_type         TEXT        NOT NULL,
    hourly_cost_usd       DOUBLE PRECISION NOT NULL,
    cpu_cores             INT,
    memory_gb             INT,
    gpu_type              TEXT,
    gpu_count             INT         DEFAULT 0,
    max_tokens_per_second DOUBLE PRECISION,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_compute_cost_models_tenant
    ON ai_compute_cost_models (tenant_id, backend_type);

ALTER TABLE ai_compute_cost_models ENABLE ROW LEVEL SECURITY;
CREATE POLICY ai_compute_cost_models_tenant_isolation ON ai_compute_cost_models
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);