CREATE TABLE IF NOT EXISTS risk_score_history (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID         NOT NULL,
    overall_score        DECIMAL(5,2) NOT NULL CHECK (overall_score BETWEEN 0.00 AND 100.00),
    grade                CHAR(1)      NOT NULL CHECK (grade IN ('A', 'B', 'C', 'D', 'F')),
    vulnerability_score  DECIMAL(5,2) NOT NULL,
    threat_score         DECIMAL(5,2) NOT NULL,
    config_score         DECIMAL(5,2) NOT NULL,
    surface_score        DECIMAL(5,2) NOT NULL,
    compliance_score     DECIMAL(5,2) NOT NULL,
    total_assets         INT          NOT NULL,
    total_open_vulns     INT          NOT NULL,
    total_open_alerts    INT          NOT NULL,
    total_active_threats INT          NOT NULL,
    components           JSONB        NOT NULL,
    top_contributors     JSONB        NOT NULL DEFAULT '[]',
    recommendations      JSONB        NOT NULL DEFAULT '[]',
    snapshot_type        TEXT         NOT NULL DEFAULT 'daily'
                                     CHECK (snapshot_type IN ('daily', 'on_demand', 'event_triggered')),
    trigger_event        TEXT,
    calculated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_risk_history_daily_unique
    ON risk_score_history (tenant_id, snapshot_type, (calculated_at::date))
    WHERE snapshot_type = 'daily';

CREATE INDEX IF NOT EXISTS idx_risk_history_tenant
    ON risk_score_history (tenant_id, calculated_at DESC);

CREATE INDEX IF NOT EXISTS idx_risk_history_trend
    ON risk_score_history (tenant_id, snapshot_type, calculated_at DESC);

CREATE INDEX IF NOT EXISTS idx_alerts_tenant_created_hour
    ON alerts (tenant_id, date_trunc('hour', created_at))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_response_time
    ON alerts (tenant_id, severity, created_at)
    WHERE acknowledged_at IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_resolve_time
    ON alerts (tenant_id, severity, created_at)
    WHERE resolved_at IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_analyst_workload
    ON alerts (tenant_id, assigned_to, status)
    WHERE assigned_to IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_mitre_technique
    ON alerts (tenant_id, mitre_technique_id)
    WHERE mitre_technique_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_vuln_aging
    ON vulnerabilities (tenant_id, discovered_at, severity)
    WHERE status IN ('open', 'in_progress') AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_alerts_asset_count
    ON alerts (tenant_id, asset_id)
    WHERE asset_id IS NOT NULL
      AND status IN ('new', 'acknowledged', 'investigating')
      AND deleted_at IS NULL;
