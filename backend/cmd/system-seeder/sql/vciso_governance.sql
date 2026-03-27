DELETE FROM vciso_approvals WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_control_ownership WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_integrations WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_control_tests WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_obligations WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_playbooks WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_escalation_rules WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_iam_findings WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_awareness_programs WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_budget_items WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_benchmarks WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_maturity_assessments WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_control_dependencies WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_evidence WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_questionnaires WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_vendors WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_policy_exceptions WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_policies WHERE tenant_id = '{{ .MainTenantID }}'::uuid;
DELETE FROM vciso_risks WHERE tenant_id = '{{ .MainTenantID }}'::uuid;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 24
        WHEN 'large' THEN 80
        ELSE 240
    END AS row_count
)
INSERT INTO vciso_risks (
    id, tenant_id, title, description, category, department, inherent_score, residual_score,
    likelihood, impact, status, treatment, owner_id, owner_name, review_date,
    business_services, controls, tags, treatment_plan, acceptance_rationale,
    acceptance_approved_by, acceptance_approved_by_name, acceptance_expiry, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-risk-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s %s risk register item', initcap(replace(meta.category, '_', ' ')), gs),
    format(
        'Seeded vCISO governance risk %s for %s with cross-suite impact on %s.',
        gs,
        meta.department,
        meta.service_pair[1]
    ),
    meta.category,
    meta.department,
    meta.inherent_score,
    meta.residual_score,
    meta.likelihood,
    meta.impact,
    meta.status,
    meta.treatment,
    meta.owner_id,
    meta.owner_name,
    to_char(current_date + make_interval(days => meta.review_offset_days), 'YYYY-MM-DD'),
    meta.service_pair,
    meta.controls,
    meta.tags,
    meta.treatment_plan,
    meta.acceptance_rationale,
    meta.acceptance_approved_by,
    meta.acceptance_approved_by_name,
    meta.acceptance_expiry,
    now() - make_interval(days => (gs % 210)),
    now() - make_interval(hours => (gs % 96))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['cybersecurity','third_party','compliance','identity','resilience','data_protection'])[1 + ((gs - 1) % 6)] AS category,
        (ARRAY['Security Operations','Legal','Finance','Technology','Risk','Data Office'])[1 + ((gs - 1) % 6)] AS department,
        GREATEST(18, LEAST(96, 42 + ((gs * 7) % 51))) AS inherent_score,
        GREATEST(8, LEAST(88, 18 + ((gs * 5) % 39))) AS residual_score,
        (ARRAY['low','medium','high','critical'])[1 + ((gs - 1) % 4)] AS likelihood,
        (ARRAY['medium','high','critical','low'])[1 + ((gs - 1) % 4)] AS impact,
        (ARRAY['open','mitigated','accepted','closed'])[1 + ((gs - 1) % 4)] AS status,
        (ARRAY['mitigate','transfer','accept','avoid'])[1 + ((gs - 1) % 4)] AS treatment,
        (ARRAY[
            '{{ .SecurityManagerUserID }}'::uuid,
            '{{ .LegalManagerUserID }}'::uuid,
            '{{ .ExecutiveUserID }}'::uuid,
            '{{ .DataStewardUserID }}'::uuid,
            '{{ .AuditorUserID }}'::uuid,
            '{{ .MainAdminUserID }}'::uuid
        ])[1 + ((gs - 1) % 6)] AS owner_id,
        (ARRAY['Musa Adebayo','Lara Bamidele','Chika Nwachukwu','Ifeoma Nwosu','Emeka Daniels','Ada Okafor'])[1 + ((gs - 1) % 6)] AS owner_name,
        ((gs % 180) - 45) AS review_offset_days,
        ARRAY[
            (ARRAY['Core Banking','Treasury','Vendor Exchange','Identity Fabric','Board Portal','Data Lake'])[1 + ((gs - 1) % 6)],
            (ARRAY['Email Gateway','Disaster Recovery','Claims Workflow','Analytics Hub','Endpoint Fleet','Regulatory Reporting'])[1 + ((gs - 1) % 6)]
        ] AS service_pair,
        ARRAY[
            (ARRAY['AC-2','AC-6','IR-4','RA-5','CP-9','SI-7'])[1 + ((gs - 1) % 6)],
            (ARRAY['AU-6','SA-9','SC-7','PL-2','PM-1','AT-2'])[1 + ((gs - 1) % 6)]
        ] AS controls,
        ARRAY[
            lower(replace((ARRAY['Ransomware','Third Party','Privacy','Privilege','Resilience','Data Exposure'])[1 + ((gs - 1) % 6)], ' ', '_')),
            lower(replace((ARRAY['board_visible','regulator_attention','q2_focus','quarterly_review'])[1 + ((gs - 1) % 4)], ' ', '_'))
        ] AS tags,
        format(
            'Seeded treatment plan %s: tighten controls for %s and rehearse contingency actions.',
            gs,
            (ARRAY['payment operations','vendor onboarding','privacy governance','privileged access','resilience testing','data retention'])[1 + ((gs - 1) % 6)]
        ) AS treatment_plan,
        CASE WHEN ((gs - 1) % 4) = 2
            THEN format('Temporary business acceptance approved for demo seed item %s pending quarterly review.', gs)
            ELSE NULL
        END AS acceptance_rationale,
        CASE WHEN ((gs - 1) % 4) = 2 THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END AS acceptance_approved_by,
        CASE WHEN ((gs - 1) % 4) = 2 THEN 'Chika Nwachukwu' ELSE NULL END AS acceptance_approved_by_name,
        CASE WHEN ((gs - 1) % 4) = 2
            THEN to_char(current_date + make_interval(days => 120 + (gs % 90)), 'YYYY-MM-DD')
            ELSE NULL
        END AS acceptance_expiry
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 16
        WHEN 'large' THEN 48
        ELSE 120
    END AS row_count
)
INSERT INTO vciso_policies (
    id, tenant_id, title, domain, version, status, content, owner_id, owner_name,
    reviewer_id, reviewer_name, approved_by, approved_by_name, approved_at, review_due,
    last_reviewed_at, tags, exceptions_count, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-policy-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s policy %s', initcap(replace(meta.domain, '_', ' ')), gs),
    meta.domain,
    format('%s.%s', 1 + ((gs - 1) / 24), 1 + ((gs - 1) % 6)),
    meta.status,
    format(
        '# %s policy %s\n\nPurpose: establish consistent governance controls for %s.\n\nScope: seeded tenants, business processes, evidence collection, and executive oversight.\n\nReview cadence: quarterly.\n',
        initcap(replace(meta.domain, '_', ' ')),
        gs,
        meta.domain
    ),
    meta.owner_id,
    meta.owner_name,
    meta.reviewer_id,
    meta.reviewer_name,
    meta.approved_by,
    meta.approved_by_name,
    meta.approved_at,
    to_char(current_date + make_interval(days => 30 + (gs % 240)), 'YYYY-MM-DD'),
    meta.last_reviewed_at,
    meta.tags,
    1 + (gs % 6),
    now() - make_interval(days => 240 - LEAST(gs, 200)),
    now() - make_interval(hours => (gs % 72))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY[
            'information_security','incident_response','access_control','data_protection',
            'vendor_security','business_continuity','cloud_security','acceptable_use'
        ])[1 + ((gs - 1) % 8)] AS domain,
        (ARRAY['published','approved','review','draft','retired'])[1 + ((gs - 1) % 5)] AS status,
        '{{ .MainAdminUserID }}'::uuid AS owner_id,
        'Ada Okafor' AS owner_name,
        '{{ .SecurityManagerUserID }}'::uuid AS reviewer_id,
        'Musa Adebayo' AS reviewer_name,
        CASE WHEN ((gs - 1) % 5) IN (0, 1) THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END AS approved_by,
        CASE WHEN ((gs - 1) % 5) IN (0, 1) THEN 'Chika Nwachukwu' ELSE NULL END AS approved_by_name,
        CASE WHEN ((gs - 1) % 5) IN (0, 1) THEN now() - make_interval(days => (gs % 150)) ELSE NULL END AS approved_at,
        CASE WHEN ((gs - 1) % 5) IN (0, 1) THEN now() - make_interval(days => (gs % 90)) ELSE NULL END AS last_reviewed_at,
        ARRAY[
            lower(replace((ARRAY['core','board','operational','regulator_mapped'])[1 + ((gs - 1) % 4)], ' ', '_')),
            lower(replace((ARRAY['iso27001','nist','soc2','privacy'])[1 + ((gs - 1) % 4)], ' ', '_'))
        ] AS tags
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 8
        WHEN 'large' THEN 24
        ELSE 60
    END AS row_count
),
policy_rows AS (
    SELECT row_number() OVER (ORDER BY created_at, id) AS rn, id, title
    FROM vciso_policies
    WHERE tenant_id = '{{ .MainTenantID }}'::uuid
),
policy_total AS (
    SELECT COUNT(*) AS total FROM policy_rows
)
INSERT INTO vciso_policy_exceptions (
    id, tenant_id, policy_id, policy_title, title, description, justification,
    compensating_controls, status, requested_by, requested_by_name, approved_by,
    approved_by_name, decision_notes, expires_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-policy-exception-' || gs),
    '{{ .MainTenantID }}'::uuid,
    policy_item.id,
    policy_item.title,
    format('Exception request %s for %s', gs, lower(policy_item.title)),
    format('Seeded exception request %s created to demonstrate approval flows.', gs),
    format('Operational dependency requires temporary deviation for seeded item %s.', gs),
    format('Enhanced logging, attestation, and weekly approval review for seeded item %s.', gs),
    meta.status,
    meta.requested_by,
    meta.requested_by_name,
    meta.approved_by,
    meta.approved_by_name,
    meta.decision_notes,
    to_char(current_date + make_interval(days => 45 + (gs % 180)), 'YYYY-MM-DD'),
    now() - make_interval(days => (gs % 120)),
    now() - make_interval(hours => (gs % 36))
FROM cfg,
     policy_total,
     generate_series(1, cfg.row_count) AS gs
JOIN LATERAL (
    SELECT id, title
    FROM policy_rows
    WHERE rn = 1 + ((gs - 1) % policy_total.total)
) AS policy_item ON true
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['approved','pending','rejected'])[1 + ((gs - 1) % 3)] AS status,
        (ARRAY[
            '{{ .SecurityManagerUserID }}'::uuid,
            '{{ .LegalManagerUserID }}'::uuid,
            '{{ .DataStewardUserID }}'::uuid
        ])[1 + ((gs - 1) % 3)] AS requested_by,
        (ARRAY['Musa Adebayo','Lara Bamidele','Ifeoma Nwosu'])[1 + ((gs - 1) % 3)] AS requested_by_name,
        CASE WHEN ((gs - 1) % 3) IN (0, 2) THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END AS approved_by,
        CASE WHEN ((gs - 1) % 3) IN (0, 2) THEN 'Chika Nwachukwu' ELSE NULL END AS approved_by_name,
        CASE ((gs - 1) % 3)
            WHEN 0 THEN 'Approved with compensating controls and quarterly review.'
            WHEN 2 THEN 'Rejected because the residual risk exceeds board tolerance.'
            ELSE NULL
        END AS decision_notes
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 18
        WHEN 'large' THEN 50
        ELSE 140
    END AS row_count
)
INSERT INTO vciso_vendors (
    id, tenant_id, name, category, risk_tier, status, risk_score, last_assessment_date,
    next_review_date, contact_name, contact_email, services_provided, data_shared,
    compliance_frameworks, controls_met, controls_total, open_findings, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-vendor-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s %s', meta.provider_stub, gs),
    meta.category,
    meta.risk_tier,
    meta.status,
    meta.risk_score,
    now() - make_interval(days => 15 + (gs % 120)),
    to_char(current_date + make_interval(days => (gs % 180) - 40), 'YYYY-MM-DD'),
    format('%s contact', meta.provider_stub),
    lower(replace(meta.provider_stub, ' ', '.')) || gs || '@vendor.demo',
    meta.services_provided,
    meta.data_shared,
    meta.frameworks,
    meta.controls_met,
    meta.controls_total,
    meta.open_findings,
    now() - make_interval(days => 365 - LEAST(gs, 300)),
    now() - make_interval(hours => (gs % 48))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['Aegis Cloud','Trust Harbor','Ledger Shield','Pulse IAM','Vertex GRC','Orbit Data'])[1 + ((gs - 1) % 6)] AS provider_stub,
        (ARRAY['cloud_security','ticketing','asset_management','iam','data_protection','siem'])[1 + ((gs - 1) % 6)] AS category,
        (ARRAY['critical','high','medium','low'])[1 + ((gs - 1) % 4)] AS risk_tier,
        (ARRAY['active','under_review','onboarding','inactive','offboarding'])[1 + ((gs - 1) % 5)] AS status,
        18 + ((gs * 9) % 77) AS risk_score,
        ARRAY[
            (ARRAY['Cloud posture management','Threat feed ingest','CMDB sync','Identity governance','DLP telemetry','SIEM analytics'])[1 + ((gs - 1) % 6)],
            (ARRAY['Case enrichment','Privileged workflow','Data retention mapping','Board evidence export','Questionnaire automation','Ticket orchestration'])[1 + ((gs - 1) % 6)]
        ] AS services_provided,
        ARRAY[
            (ARRAY['security_logs','identity_events','pii_metadata','asset_inventory','customer_records','control_attestations'])[1 + ((gs - 1) % 6)],
            (ARRAY['configurations','usage_metrics','endpoint_telemetry','audit_trails','vendor_contacts','legal_documents'])[1 + ((gs - 1) % 6)]
        ] AS data_shared,
        ARRAY[
            (ARRAY['ISO27001','SOC2','GDPR','NIST CSF','PCI DSS','HIPAA'])[1 + ((gs - 1) % 6)],
            (ARRAY['NCA ECC','SAMA','SOC2','GDPR','ISO27001','NIST 800-53'])[1 + ((gs - 1) % 6)]
        ] AS frameworks,
        10 + ((gs * 3) % 26) AS controls_met,
        20 + ((gs * 5) % 20) AS controls_total,
        gs % 9 AS open_findings
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 16
        WHEN 'large' THEN 48
        ELSE 120
    END AS row_count
),
vendor_rows AS (
    SELECT row_number() OVER (ORDER BY created_at, id) AS rn, id, name
    FROM vciso_vendors
    WHERE tenant_id = '{{ .MainTenantID }}'::uuid
),
vendor_total AS (
    SELECT COUNT(*) AS total FROM vendor_rows
)
INSERT INTO vciso_questionnaires (
    id, tenant_id, title, type, status, vendor_id, vendor_name, total_questions,
    answered_questions, due_date, completed_at, score, assigned_to, assigned_to_name,
    created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-questionnaire-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s questionnaire %s', initcap(replace(meta.kind, '_', ' ')), gs),
    meta.kind,
    meta.status,
    CASE WHEN meta.kind = 'internal_audit' THEN NULL ELSE vendor_item.id END,
    CASE WHEN meta.kind = 'internal_audit' THEN NULL ELSE vendor_item.name END,
    meta.total_questions,
    meta.answered_questions,
    to_char(current_date + make_interval(days => (gs % 120) - 20), 'YYYY-MM-DD'),
    meta.completed_at,
    meta.score,
    meta.assigned_to,
    meta.assigned_to_name,
    now() - make_interval(days => (gs % 150)),
    now() - make_interval(hours => (gs % 48))
FROM cfg,
     vendor_total,
     generate_series(1, cfg.row_count) AS gs
JOIN LATERAL (
    SELECT id, name
    FROM vendor_rows
    WHERE rn = 1 + ((gs - 1) % vendor_total.total)
) AS vendor_item ON true
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['vendor_assessment','security_review','internal_audit','compliance_check'])[1 + ((gs - 1) % 4)] AS kind,
        (ARRAY['completed','in_progress','draft','sent'])[1 + ((gs - 1) % 4)] AS status,
        40 + ((gs * 7) % 90) AS total_questions,
        CASE ((gs - 1) % 4)
            WHEN 0 THEN 40 + ((gs * 7) % 90)
            WHEN 1 THEN 18 + ((gs * 5) % 30)
            WHEN 2 THEN 0
            ELSE 12 + ((gs * 3) % 20)
        END AS answered_questions,
        CASE WHEN ((gs - 1) % 4) = 0 THEN now() - make_interval(days => (gs % 45)) ELSE NULL END AS completed_at,
        CASE WHEN ((gs - 1) % 4) = 0 THEN 62 + (gs % 34) ELSE NULL END AS score,
        (ARRAY[
            '{{ .SecurityManagerUserID }}'::uuid,
            '{{ .AuditorUserID }}'::uuid,
            '{{ .LegalManagerUserID }}'::uuid,
            '{{ .DataStewardUserID }}'::uuid
        ])[1 + ((gs - 1) % 4)] AS assigned_to,
        (ARRAY['Musa Adebayo','Emeka Daniels','Lara Bamidele','Ifeoma Nwosu'])[1 + ((gs - 1) % 4)] AS assigned_to_name
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 40
        WHEN 'large' THEN 120
        ELSE 320
    END AS row_count
)
INSERT INTO vciso_evidence (
    id, tenant_id, title, description, type, source, status, frameworks, control_ids,
    file_name, file_size, file_url, collected_at, expires_at, collector_name,
    last_verified_at, verified_by, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-evidence-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s evidence pack %s', initcap(replace(meta.evidence_type, '_', ' ')), gs),
    format('Seeded evidence item %s for %s controls.', gs, meta.frameworks[1]),
    meta.evidence_type,
    meta.source,
    meta.status,
    meta.frameworks,
    meta.control_ids,
    CASE WHEN meta.source = 'manual_attestation' THEN NULL ELSE lower(replace(meta.evidence_type, '_', '-')) || '-' || gs || '.pdf' END,
    CASE WHEN meta.source = 'manual_attestation' THEN NULL ELSE 48000 + ((gs * 137) % 1900000) END,
    CASE WHEN meta.source = 'manual_attestation' THEN NULL ELSE format('https://demo.clario.local/evidence/%s', gs) END,
    now() - make_interval(days => (gs % 240)),
    CASE WHEN meta.status IN ('active','verified') THEN now() + make_interval(days => 60 + (gs % 180)) ELSE now() - make_interval(days => (gs % 30)) END,
    meta.collector_name,
    CASE WHEN meta.status = 'verified' THEN now() - make_interval(days => (gs % 45)) ELSE NULL END,
    CASE WHEN meta.status = 'verified' THEN '{{ .AuditorUserID }}'::uuid ELSE NULL END,
    now() - make_interval(days => (gs % 260)),
    now() - make_interval(hours => (gs % 60))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['report','attestation','screen_capture','configuration','certificate','log_export'])[1 + ((gs - 1) % 6)] AS evidence_type,
        (ARRAY['external_audit','automated_scan','manual_attestation','workflow_export'])[1 + ((gs - 1) % 4)] AS source,
        (ARRAY['verified','active','pending','expired'])[1 + ((gs - 1) % 4)] AS status,
        ARRAY[
            (ARRAY['ISO27001','SOC2','NIST CSF','GDPR','PCI DSS','NCA ECC'])[1 + ((gs - 1) % 6)],
            (ARRAY['SAMA','HIPAA','SOC2','ISO27001','NIST 800-53','GDPR'])[1 + ((gs - 1) % 6)]
        ] AS frameworks,
        ARRAY[
            (ARRAY['AC-2','AC-6','IR-4','RA-5','CP-9','AU-6'])[1 + ((gs - 1) % 6)],
            (ARRAY['SC-7','SA-9','AT-2','PM-1','CM-6','SI-7'])[1 + ((gs - 1) % 6)]
        ] AS control_ids,
        (ARRAY['Ada Okafor','Musa Adebayo','Emeka Daniels','Lara Bamidele'])[1 + ((gs - 1) % 4)] AS collector_name
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 6
        WHEN 'large' THEN 18
        ELSE 42
    END AS row_count
)
INSERT INTO vciso_maturity_assessments (
    id, tenant_id, framework, status, overall_score, overall_level, dimensions,
    assessor_name, assessed_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-maturity-' || gs),
    '{{ .MainTenantID }}'::uuid,
    meta.framework,
    meta.status,
    meta.overall_score,
    meta.overall_level,
    jsonb_build_array(
        jsonb_build_object(
            'name', 'Governance',
            'category', 'governance',
            'current_level', meta.overall_level,
            'target_level', LEAST(meta.overall_level + 1, 5),
            'score', meta.overall_score,
            'findings', jsonb_build_array(format('Seeded governance gap %s', gs)),
            'recommendations', jsonb_build_array('Refresh ownership mapping', 'Tighten quarterly evidence review')
        ),
        jsonb_build_object(
            'name', 'Operations',
            'category', 'operations',
            'current_level', GREATEST(meta.overall_level - 1, 1),
            'target_level', LEAST(meta.overall_level + 1, 5),
            'score', GREATEST(meta.overall_score - 0.2, 1.0),
            'findings', jsonb_build_array('Automation maturity still uneven across suites'),
            'recommendations', jsonb_build_array('Expand testing cadence', 'Operationalize exception handling')
        )
    ),
    meta.assessor_name,
    now() - make_interval(days => (gs % 180)),
    now() - make_interval(days => (gs % 180)),
    now() - make_interval(hours => (gs % 48))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['NIST CSF','ISO 27001','CIS Controls v8','SOC 2','NCA ECC','SAMA'])[1 + ((gs - 1) % 6)] AS framework,
        (ARRAY['completed','in_progress','completed'])[1 + ((gs - 1) % 3)] AS status,
        round((2.1 + ((gs % 24) * 0.11))::numeric, 2) AS overall_score,
        1 + (gs % 5) AS overall_level,
        (ARRAY['Security Assessment Team','External Auditor','Risk Assurance Office'])[1 + ((gs - 1) % 3)] AS assessor_name
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 12
        WHEN 'large' THEN 36
        ELSE 96
    END AS row_count
)
INSERT INTO vciso_benchmarks (
    id, tenant_id, dimension, category, organization_score, industry_average,
    industry_top_quartile, peer_average, gap, framework, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-benchmark-' || gs),
    '{{ .MainTenantID }}'::uuid,
    meta.dimension,
    meta.category,
    meta.organization_score,
    meta.industry_average,
    meta.industry_top_quartile,
    meta.peer_average,
    round((meta.organization_score - meta.industry_average)::numeric, 2),
    meta.framework,
    now() - make_interval(days => (gs % 120)),
    now() - make_interval(hours => (gs % 24))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY[
            'Identity & Access Management','Data Protection','Incident Response','Third-Party Risk',
            'Security Awareness','Cloud Security','Regulatory Readiness','Business Continuity'
        ])[1 + ((gs - 1) % 8)] AS dimension,
        (ARRAY['security','security','operations','governance','people','security','governance','resilience'])[1 + ((gs - 1) % 8)] AS category,
        48 + ((gs * 3) % 38) AS organization_score,
        50 + ((gs * 5) % 28) AS industry_average,
        72 + ((gs * 7) % 18) AS industry_top_quartile,
        52 + ((gs * 4) % 24) AS peer_average,
        (ARRAY['NIST CSF','ISO 27001','SOC 2','NCA ECC'])[1 + ((gs - 1) % 4)] AS framework
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 18
        WHEN 'large' THEN 54
        ELSE 150
    END AS row_count
)
INSERT INTO vciso_budget_items (
    id, tenant_id, title, category, type, amount, currency, status, risk_reduction_estimate,
    priority, justification, linked_risk_ids, linked_recommendation_ids, fiscal_year, quarter,
    owner_name, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-budget-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s investment %s', initcap(replace(meta.category, '_', ' ')), gs),
    meta.category,
    meta.item_type,
    meta.amount,
    'USD',
    meta.status,
    meta.risk_reduction_estimate,
    1 + (gs % 5),
    format('Seeded budget item %s for %s improvements.', gs, meta.category),
    ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-risk-' || (1 + ((gs - 1) % risk_cfg.risk_count)))::text],
    ARRAY[format('REC-%s', 1000 + gs), format('REC-%s', 2000 + gs)],
    format('FY%s', EXTRACT(YEAR FROM CURRENT_DATE)::int),
    (ARRAY['Q1','Q2','Q3','Q4'])[1 + ((gs - 1) % 4)],
    meta.owner_name,
    now() - make_interval(days => (gs % 300)),
    now() - make_interval(hours => (gs % 36))
FROM cfg,
     (SELECT CASE '{{ .Scale.Name }}' WHEN 'small' THEN 24 WHEN 'large' THEN 80 ELSE 240 END AS risk_count) AS risk_cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['identity_security','third_party_risk','resilience','data_protection','cloud_security','security_operations'])[1 + ((gs - 1) % 6)] AS category,
        (ARRAY['capex','opex'])[1 + ((gs - 1) % 2)] AS item_type,
        round((25000 + ((gs * 1275) % 210000))::numeric, 2) AS amount,
        (ARRAY['proposed','approved','spent'])[1 + ((gs - 1) % 3)] AS status,
        round((5 + ((gs * 13) % 41))::numeric, 2) AS risk_reduction_estimate,
        (ARRAY['Ada Okafor','Musa Adebayo','Chika Nwachukwu'])[1 + ((gs - 1) % 3)] AS owner_name
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 12
        WHEN 'large' THEN 32
        ELSE 80
    END AS row_count
)
INSERT INTO vciso_awareness_programs (
    id, tenant_id, name, type, status, total_users, completed_users, passed_users,
    failed_users, completion_rate, pass_rate, start_date, end_date, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-awareness-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s cohort %s', initcap(replace(meta.program_type, '_', ' ')), gs),
    meta.program_type,
    meta.status,
    meta.total_users,
    meta.completed_users,
    meta.passed_users,
    meta.failed_users,
    round((meta.completed_users::numeric * 100 / GREATEST(meta.total_users, 1))::numeric, 2),
    round((meta.passed_users::numeric * 100 / GREATEST(meta.total_users, 1))::numeric, 2),
    to_char(current_date - make_interval(days => 30 + (gs % 120)), 'YYYY-MM-DD'),
    to_char(current_date + make_interval(days => 15 + (gs % 90)), 'YYYY-MM-DD'),
    now() - make_interval(days => (gs % 200)),
    now() - make_interval(hours => (gs % 24))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['training','phishing_simulation','briefing','tabletop'])[1 + ((gs - 1) % 4)] AS program_type,
        (ARRAY['active','completed','planned'])[1 + ((gs - 1) % 3)] AS status,
        25 + ((gs * 17) % 700) AS total_users,
        CASE ((gs - 1) % 3)
            WHEN 0 THEN 12 + ((gs * 11) % 420)
            WHEN 1 THEN 25 + ((gs * 17) % 700)
            ELSE 0
        END AS completed_users,
        CASE ((gs - 1) % 3)
            WHEN 0 THEN 10 + ((gs * 9) % 360)
            WHEN 1 THEN 20 + ((gs * 13) % 610)
            ELSE 0
        END AS passed_users,
        CASE ((gs - 1) % 3)
            WHEN 0 THEN gs % 18
            WHEN 1 THEN gs % 35
            ELSE 0
        END AS failed_users
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 24
        WHEN 'large' THEN 80
        ELSE 220
    END AS row_count
)
INSERT INTO vciso_iam_findings (
    id, tenant_id, type, severity, title, description, affected_users, status,
    remediation, discovered_at, resolved_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-iam-finding-' || gs),
    '{{ .MainTenantID }}'::uuid,
    meta.finding_type,
    meta.severity,
    format('%s finding %s', initcap(replace(meta.finding_type, '_', ' ')), gs),
    format('Seeded IAM finding %s for %s review and remediation drill.', gs, meta.finding_type),
    1 + (gs % 42),
    meta.status,
    meta.remediation,
    now() - make_interval(days => (gs % 180)),
    meta.resolved_at,
    now() - make_interval(days => (gs % 200)),
    now() - make_interval(hours => (gs % 30))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['over_privileged','orphaned_account','stale_access','mfa_gap','shared_credentials','segregation_of_duties'])[1 + ((gs - 1) % 6)] AS finding_type,
        (ARRAY['critical','high','medium','low'])[1 + ((gs - 1) % 4)] AS severity,
        (ARRAY['open','in_progress','remediated','accepted'])[1 + ((gs - 1) % 4)] AS status,
        CASE WHEN ((gs - 1) % 4) = 2
            THEN 'Remediation completed and controls retested.'
            ELSE 'Review access path, rotate credentials, and tighten approval workflow.'
        END AS remediation,
        CASE WHEN ((gs - 1) % 4) = 2 THEN now() - make_interval(days => (gs % 60)) ELSE NULL END AS resolved_at
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 8
        WHEN 'large' THEN 20
        ELSE 48
    END AS row_count
)
INSERT INTO vciso_escalation_rules (
    id, tenant_id, name, description, trigger_type, trigger_condition, escalation_target,
    target_contacts, notification_channels, enabled, last_triggered_at, trigger_count,
    created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-escalation-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s escalation rule %s', initcap(meta.trigger_type), gs),
    format('Seeded escalation rule %s for %s routing.', gs, meta.escalation_target),
    meta.trigger_type,
    meta.trigger_condition,
    meta.escalation_target,
    meta.target_contacts,
    meta.notification_channels,
    (gs % 7) <> 0,
    CASE WHEN (gs % 3) = 0 THEN now() - make_interval(days => (gs % 20)) ELSE NULL END,
    gs % 15,
    now() - make_interval(days => (gs % 160)),
    now() - make_interval(hours => (gs % 24))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['severity','time','count','custom'])[1 + ((gs - 1) % 4)] AS trigger_type,
        (ARRAY[
            'severity >= high',
            'open_for_minutes > 30',
            'trigger_count > 5 in 24h',
            'entity_type = ''vendor'' and risk_tier = ''critical'''
        ])[1 + ((gs - 1) % 4)] AS trigger_condition,
        (ARRAY['management','board','legal','regulator','custom'])[1 + ((gs - 1) % 5)] AS escalation_target,
        ARRAY[
            (ARRAY['soc@apexbank.demo','board@apexbank.demo','legal@apexbank.demo','regulator@apexbank.demo','risk@apexbank.demo'])[1 + ((gs - 1) % 5)],
            'exec-ops@apexbank.demo'
        ] AS target_contacts,
        ARRAY[
            (ARRAY['email','slack','teams','sms'])[1 + ((gs - 1) % 4)],
            (ARRAY['email','ticket','pager'])[1 + ((gs - 1) % 3)]
        ] AS notification_channels
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 10
        WHEN 'large' THEN 24
        ELSE 64
    END AS row_count
)
INSERT INTO vciso_playbooks (
    id, tenant_id, name, scenario, status, last_tested_at, next_test_date, owner_id, owner_name,
    steps_count, dependencies, rto_hours, rpo_hours, last_simulation_result, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-playbook-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s playbook %s', initcap(replace(meta.scenario, '_', ' ')), gs),
    meta.scenario,
    meta.status,
    meta.last_tested_at,
    to_char(current_date + make_interval(days => 20 + (gs % 180)), 'YYYY-MM-DD'),
    meta.owner_id,
    meta.owner_name,
    5 + (gs % 9),
    meta.dependencies,
    round((2 + ((gs % 12) * 0.75))::numeric, 2),
    round((1 + ((gs % 8) * 0.5))::numeric, 2),
    meta.last_simulation_result,
    now() - make_interval(days => (gs % 320)),
    now() - make_interval(hours => (gs % 20))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['ransomware','data_breach','cloud_outage','vendor_failure','privilege_abuse','regulatory_response'])[1 + ((gs - 1) % 6)] AS scenario,
        (ARRAY['draft','approved','tested','retired'])[1 + ((gs - 1) % 4)] AS status,
        CASE WHEN ((gs - 1) % 4) IN (1, 2) THEN now() - make_interval(days => (gs % 90)) ELSE NULL END AS last_tested_at,
        (ARRAY['pass','partial','fail'])[1 + ((gs - 1) % 3)] AS last_simulation_result,
        (ARRAY['{{ .SecurityManagerUserID }}'::uuid,'{{ .MainAdminUserID }}'::uuid,'{{ .ExecutiveUserID }}'::uuid])[1 + ((gs - 1) % 3)] AS owner_id,
        (ARRAY['Musa Adebayo','Ada Okafor','Chika Nwachukwu'])[1 + ((gs - 1) % 3)] AS owner_name,
        ARRAY[
            (ARRAY['identity escalation','legal notification','backup validation','vendor bridge','customer communications','board brief'])[1 + ((gs - 1) % 6)],
            (ARRAY['forensics stream','war room','ticket workflow','executive approval','regulator notice','evidence lock'])[1 + ((gs - 1) % 6)]
        ] AS dependencies
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 16
        WHEN 'large' THEN 48
        ELSE 120
    END AS row_count
)
INSERT INTO vciso_obligations (
    id, tenant_id, name, type, jurisdiction, description, requirements, status,
    mapped_controls, total_requirements, met_requirements, owner_id, owner_name,
    effective_date, review_date, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-obligation-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s obligation %s', initcap(replace(meta.obligation_type, '_', ' ')), gs),
    meta.obligation_type,
    meta.jurisdiction,
    format('Seeded obligation %s mapped to governance controls and evidence.', gs),
    meta.requirements,
    meta.status,
    meta.mapped_controls,
    meta.total_requirements,
    meta.met_requirements,
    meta.owner_id,
    meta.owner_name,
    to_char(current_date - make_interval(days => 120 + (gs % 365)), 'YYYY-MM-DD'),
    to_char(current_date + make_interval(days => 20 + (gs % 240)), 'YYYY-MM-DD'),
    now() - make_interval(days => (gs % 420)),
    now() - make_interval(hours => (gs % 24))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['regulatory','contractual','certification','board_mandate'])[1 + ((gs - 1) % 4)] AS obligation_type,
        (ARRAY['Nigeria','Saudi Arabia','United Kingdom','European Union','United States'])[1 + ((gs - 1) % 5)] AS jurisdiction,
        ARRAY[
            format('Requirement %sA', gs),
            format('Requirement %sB', gs),
            format('Requirement %sC', gs)
        ] AS requirements,
        (ARRAY['active','pending','at_risk'])[1 + ((gs - 1) % 3)] AS status,
        4 + (gs % 12) AS mapped_controls,
        6 + (gs % 14) AS total_requirements,
        2 + (gs % 10) AS met_requirements,
        (ARRAY['{{ .LegalManagerUserID }}'::uuid,'{{ .SecurityManagerUserID }}'::uuid,'{{ .MainAdminUserID }}'::uuid])[1 + ((gs - 1) % 3)] AS owner_id,
        (ARRAY['Lara Bamidele','Musa Adebayo','Ada Okafor'])[1 + ((gs - 1) % 3)] AS owner_name
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 24
        WHEN 'large' THEN 90
        ELSE 240
    END AS row_count
)
INSERT INTO vciso_control_tests (
    id, tenant_id, control_id, control_name, framework, test_type, result, tester_name,
    test_date, next_test_date, findings, evidence_ids, test_name, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-control-test-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('CTRL-%03s', 1 + ((gs - 1) % 180)),
    format('%s control %s', initcap(replace(meta.framework, '_', ' ')), 1 + ((gs - 1) % 180)),
    meta.framework,
    meta.test_type,
    meta.result,
    meta.tester_name,
    to_char(current_date - make_interval(days => (gs % 120)), 'YYYY-MM-DD'),
    to_char(current_date + make_interval(days => 25 + (gs % 160)), 'YYYY-MM-DD'),
    meta.findings,
    ARRAY[
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-evidence-' || (1 + ((gs - 1) % evidence_cfg.evidence_count)))::text,
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-evidence-' || (1 + ((gs + 11) % evidence_cfg.evidence_count)))::text
    ],
    format('%s %s validation', initcap(replace(meta.test_type, '_', ' ')), gs),
    now() - make_interval(days => (gs % 140)),
    now() - make_interval(hours => (gs % 24))
FROM cfg,
     (SELECT CASE '{{ .Scale.Name }}' WHEN 'small' THEN 40 WHEN 'large' THEN 120 ELSE 320 END AS evidence_count) AS evidence_cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['NIST','ISO27001','SOC2','CIS'])[1 + ((gs - 1) % 4)] AS framework,
        (ARRAY['design','operating_effectiveness'])[1 + ((gs - 1) % 2)] AS test_type,
        (ARRAY['effective','partially_effective','ineffective','not_tested'])[1 + ((gs - 1) % 4)] AS result,
        (ARRAY['Emeka Daniels','Musa Adebayo','Ada Okafor'])[1 + ((gs - 1) % 3)] AS tester_name,
        format('Seeded control test %s highlighted %s findings.', gs, (ARRAY['minor','material','tracking','evidence'])[1 + ((gs - 1) % 4)]) AS findings
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 12
        WHEN 'large' THEN 36
        ELSE 90
    END AS row_count
)
INSERT INTO vciso_integrations (
    id, tenant_id, name, type, provider, status, last_sync_at, sync_frequency,
    items_synced, config, health_status, error_message, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-integration-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('%s connector %s', initcap(replace(meta.integration_type, '_', ' ')), gs),
    meta.integration_type,
    meta.provider,
    meta.status,
    CASE WHEN meta.status = 'pending' THEN NULL ELSE now() - make_interval(hours => (gs % 72)) END,
    meta.sync_frequency,
    CASE WHEN meta.status = 'connected' THEN 200 + ((gs * 17) % 8000) ELSE 0 END,
    jsonb_build_object(
        'workspace', lower(replace(meta.provider, ' ', '-')) || '-' || gs,
        'region', (ARRAY['eu-west-1','me-central-1','us-east-1'])[1 + ((gs - 1) % 3)],
        'scope', (ARRAY['prod','shared','governance'])[1 + ((gs - 1) % 3)]
    ),
    meta.health_status,
    CASE WHEN meta.status = 'error' THEN 'Seeded sync failure for recovery workflow demo.' ELSE NULL END,
    now() - make_interval(days => (gs % 180)),
    now() - make_interval(hours => (gs % 12))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['asset_management','ticketing','cloud_security','data_protection','siem','iam'])[1 + ((gs - 1) % 6)] AS integration_type,
        (ARRAY['ServiceNow','Jira Service Management','Microsoft Defender for Cloud','BigID','Microsoft Sentinel','Okta'])[1 + ((gs - 1) % 6)] AS provider,
        (ARRAY['connected','disconnected','error','pending'])[1 + ((gs - 1) % 4)] AS status,
        (ARRAY['every_5m','every_15m','every_hour','every_6h','daily'])[1 + ((gs - 1) % 5)] AS sync_frequency,
        (ARRAY['healthy','unavailable','degraded','degraded'])[1 + ((gs - 1) % 4)] AS health_status
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 24
        WHEN 'large' THEN 80
        ELSE 220
    END AS row_count
)
INSERT INTO vciso_control_ownership (
    id, tenant_id, control_id, control_name, framework, owner_id, owner_name, delegate_id,
    delegate_name, status, last_reviewed_at, next_review_date, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-control-owner-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('CTRL-%03s', 1 + ((gs - 1) % 180)),
    format('Control ownership %s', gs),
    meta.framework,
    meta.owner_id,
    meta.owner_name,
    meta.delegate_id,
    meta.delegate_name,
    meta.status,
    CASE WHEN meta.status <> 'new' THEN now() - make_interval(days => (gs % 90)) ELSE NULL END,
    to_char(current_date + make_interval(days => 25 + (gs % 180)), 'YYYY-MM-DD'),
    now() - make_interval(days => (gs % 200)),
    now() - make_interval(hours => (gs % 18))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['NIST','ISO27001','SOC2','CIS'])[1 + ((gs - 1) % 4)] AS framework,
        (ARRAY[
            '{{ .SecurityManagerUserID }}'::uuid,
            '{{ .DataStewardUserID }}'::uuid,
            '{{ .LegalManagerUserID }}'::uuid,
            '{{ .MainAdminUserID }}'::uuid
        ])[1 + ((gs - 1) % 4)] AS owner_id,
        (ARRAY['Musa Adebayo','Ifeoma Nwosu','Lara Bamidele','Ada Okafor'])[1 + ((gs - 1) % 4)] AS owner_name,
        (ARRAY[
            '{{ .ExecutiveUserID }}'::uuid,
            '{{ .AuditorUserID }}'::uuid,
            '{{ .BoardSecretaryUserID }}'::uuid,
            NULL::uuid
        ])[1 + ((gs - 1) % 4)] AS delegate_id,
        (ARRAY['Chika Nwachukwu','Emeka Daniels','Tade Akinola',NULL])[1 + ((gs - 1) % 4)] AS delegate_name,
        (ARRAY['assigned','delegated','review_due','new'])[1 + ((gs - 1) % 4)] AS status
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 18
        WHEN 'large' THEN 60
        ELSE 180
    END AS row_count
)
INSERT INTO vciso_approvals (
    id, tenant_id, type, title, description, status, requested_by, requested_by_name,
    approver_id, approver_name, priority, decision_notes, decided_at, deadline,
    linked_entity_type, linked_entity_id, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-approval-' || gs),
    '{{ .MainTenantID }}'::uuid,
    meta.approval_type,
    format('%s approval %s', initcap(replace(meta.approval_type, '_', ' ')), gs),
    format('Seeded governance approval request %s linked to %s.', gs, meta.linked_entity_type),
    meta.status,
    meta.requested_by,
    meta.requested_by_name,
    meta.approver_id,
    meta.approver_name,
    meta.priority,
    meta.decision_notes,
    meta.decided_at,
    to_char(current_date + make_interval(days => 7 + (gs % 90)), 'YYYY-MM-DD'),
    meta.linked_entity_type,
    meta.linked_entity_id,
    now() - make_interval(days => (gs % 120)),
    now() - make_interval(hours => (gs % 18))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['risk_acceptance','policy_exception','budget_change','vendor_onboarding','control_override'])[1 + ((gs - 1) % 5)] AS approval_type,
        (ARRAY['pending','approved','rejected','escalated'])[1 + ((gs - 1) % 4)] AS status,
        (ARRAY[
            '{{ .SecurityManagerUserID }}'::uuid,
            '{{ .LegalManagerUserID }}'::uuid,
            '{{ .MainAdminUserID }}'::uuid,
            '{{ .DataStewardUserID }}'::uuid
        ])[1 + ((gs - 1) % 4)] AS requested_by,
        (ARRAY['Musa Adebayo','Lara Bamidele','Ada Okafor','Ifeoma Nwosu'])[1 + ((gs - 1) % 4)] AS requested_by_name,
        (ARRAY[
            '{{ .ExecutiveUserID }}'::uuid,
            '{{ .AuditorUserID }}'::uuid,
            '{{ .BoardSecretaryUserID }}'::uuid,
            '{{ .MainAdminUserID }}'::uuid
        ])[1 + ((gs - 1) % 4)] AS approver_id,
        (ARRAY['Chika Nwachukwu','Emeka Daniels','Tade Akinola','Ada Okafor'])[1 + ((gs - 1) % 4)] AS approver_name,
        (ARRAY['low','medium','high','critical'])[1 + ((gs - 1) % 4)] AS priority,
        CASE ((gs - 1) % 4)
            WHEN 1 THEN 'Approved with evidence follow-up.'
            WHEN 2 THEN 'Rejected pending stronger compensating controls.'
            WHEN 3 THEN 'Escalated to board due to regulatory impact.'
            ELSE NULL
        END AS decision_notes,
        CASE WHEN ((gs - 1) % 4) IN (1, 2, 3) THEN now() - make_interval(days => (gs % 21)) ELSE NULL END AS decided_at,
        (ARRAY['risk','policy_exception','budget_item','vendor','control_test'])[1 + ((gs - 1) % 5)] AS linked_entity_type,
        CASE ((gs - 1) % 5)
            WHEN 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-risk-' || (1 + ((gs - 1) % (CASE '{{ .Scale.Name }}' WHEN 'small' THEN 24 WHEN 'large' THEN 80 ELSE 240 END))))::text
            WHEN 1 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-policy-exception-' || (1 + ((gs - 1) % (CASE '{{ .Scale.Name }}' WHEN 'small' THEN 8 WHEN 'large' THEN 24 ELSE 60 END))))::text
            WHEN 2 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-budget-' || (1 + ((gs - 1) % (CASE '{{ .Scale.Name }}' WHEN 'small' THEN 18 WHEN 'large' THEN 54 ELSE 150 END))))::text
            WHEN 3 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-vendor-' || (1 + ((gs - 1) % (CASE '{{ .Scale.Name }}' WHEN 'small' THEN 18 WHEN 'large' THEN 50 ELSE 140 END))))::text
            ELSE uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-control-test-' || (1 + ((gs - 1) % (CASE '{{ .Scale.Name }}' WHEN 'small' THEN 24 WHEN 'large' THEN 90 ELSE 240 END))))::text
        END AS linked_entity_id
) AS meta;

WITH cfg AS (
    SELECT CASE '{{ .Scale.Name }}'
        WHEN 'small' THEN 18
        WHEN 'large' THEN 54
        ELSE 150
    END AS row_count
)
INSERT INTO vciso_control_dependencies (
    id, tenant_id, control_id, control_name, framework, depends_on, depended_by,
    risk_domains, compliance_domains, failure_impact, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-governance-control-dependency-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('CTRL-%03s', gs),
    format('%s dependency control %s', meta.framework, gs),
    meta.framework,
    ARRAY[
        format('CTRL-%03s', GREATEST(1, gs - 1)),
        format('CTRL-%03s', GREATEST(1, gs - 2))
    ],
    ARRAY[
        format('CTRL-%03s', LEAST(gs + 1, cfg.row_count)),
        format('CTRL-%03s', LEAST(gs + 2, cfg.row_count))
    ],
    meta.risk_domains,
    meta.compliance_domains,
    meta.failure_impact,
    now() - make_interval(days => (gs % 180)),
    now() - make_interval(hours => (gs % 12))
FROM cfg,
     generate_series(1, cfg.row_count) AS gs
CROSS JOIN LATERAL (
    SELECT
        (ARRAY['NIST','ISO27001','SOC2','CIS'])[1 + ((gs - 1) % 4)] AS framework,
        ARRAY[
            (ARRAY['identity','third_party','resilience','privacy','operations'])[1 + ((gs - 1) % 5)],
            (ARRAY['regulatory','vendor','endpoint','cloud','governance'])[1 + ((gs - 1) % 5)]
        ] AS risk_domains,
        ARRAY[
            (ARRAY['access_control','incident_response','data_protection','business_continuity'])[1 + ((gs - 1) % 4)],
            (ARRAY['board_reporting','regulatory_filing','vendor_assurance','evidence_management'])[1 + ((gs - 1) % 4)]
        ] AS compliance_domains,
        (ARRAY['medium','high','critical'])[1 + ((gs - 1) % 3)] AS failure_impact
) AS meta;
