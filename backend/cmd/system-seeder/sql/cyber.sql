INSERT INTO scan_history (
    id, tenant_id, scan_type, config, status, assets_discovered, assets_new, assets_updated,
    error_count, errors, started_at, completed_at, duration_ms, created_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-scan-history-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'network'
        WHEN 1 THEN 'cloud'
        WHEN 2 THEN 'agent'
        ELSE 'import'
    END,
    jsonb_build_object('profile', format('seeded-scan-%s', gs), 'seeded', true),
    CASE WHEN gs % 11 = 0 THEN 'failed' WHEN gs % 5 = 0 THEN 'running' ELSE 'completed' END,
    60 + (gs * 8),
    12 + (gs % 30),
    24 + (gs % 45),
    CASE WHEN gs % 11 = 0 THEN 2 ELSE 0 END,
    CASE WHEN gs % 11 = 0 THEN jsonb_build_array('Seeded connector timeout', 'Seeded API throttling') ELSE '[]'::jsonb END,
    now() - make_interval(days => gs),
    CASE WHEN gs % 5 = 0 THEN NULL ELSE now() - make_interval(days => gs) + interval '18 minutes' END,
    CASE WHEN gs % 5 = 0 THEN NULL ELSE 1080000 END,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => gs)
FROM generate_series(1, {{ .Scale.CTEMAssessmentCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    status = EXCLUDED.status,
    assets_discovered = EXCLUDED.assets_discovered,
    assets_new = EXCLUDED.assets_new,
    assets_updated = EXCLUDED.assets_updated,
    error_count = EXCLUDED.error_count,
    errors = EXCLUDED.errors,
    completed_at = EXCLUDED.completed_at,
    duration_ms = EXCLUDED.duration_ms;

INSERT INTO cve_database (
    cve_id, description, severity, cvss_v3_score, cvss_v3_vector, cpe_matches,
    affected_products, published_at, modified_at, "references", created_at
)
SELECT
    format('CVE-2026-%06s', gs),
    format('Seeded CVE %s for vulnerability enrichment and prioritization demos.', gs),
    CASE
        WHEN gs % 17 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    CASE
        WHEN gs % 17 = 0 THEN 9.8
        WHEN gs % 7 = 0 THEN 8.2
        WHEN gs % 3 = 0 THEN 6.4
        ELSE 3.9
    END,
    'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H',
    ARRAY[format('cpe:2.3:a:seeded:product:%s:*:*:*:*:*:*:*:*', gs)],
    jsonb_build_array(jsonb_build_object('product', format('seeded-product-%s', gs), 'version', '2026.1')),
    now() - make_interval(days => 180 - (gs % 120)),
    now() - make_interval(days => 30 - (gs % 20)),
    jsonb_build_array(format('https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2026-%06s', gs)),
    now() - make_interval(days => 180 - (gs % 120))
FROM generate_series(1, {{ .Scale.VulnerabilityCount }}) gs
ON CONFLICT (cve_id) DO UPDATE SET
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    cvss_v3_score = EXCLUDED.cvss_v3_score,
    modified_at = EXCLUDED.modified_at,
    "references" = EXCLUDED."references";

INSERT INTO threat_feed_configs (
    id, tenant_id, name, type, url, auth_type, auth_config, sync_interval, default_severity,
    default_confidence, default_tags, indicator_types, enabled, status, last_sync_at,
    last_sync_status, last_error, created_by, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'threat-feed-config-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE gs
        WHEN 1 THEN 'Seeded TAXII Feed'
        WHEN 2 THEN 'Seeded STIX Feed'
        WHEN 3 THEN 'Seeded MISP Feed'
        WHEN 4 THEN 'Seeded Vendor IOC Feed'
        WHEN 5 THEN 'Seeded CSV IOC Feed'
        ELSE 'Seeded Manual IOC Feed'
    END,
    CASE gs
        WHEN 1 THEN 'taxii'
        WHEN 2 THEN 'stix'
        WHEN 3 THEN 'misp'
        WHEN 4 THEN 'manual'
        WHEN 5 THEN 'csv_url'
        ELSE 'manual'
    END,
    format('https://feeds.seeded.local/%s', gs),
    CASE WHEN gs IN (1, 3, 4) THEN 'api_key' ELSE 'none' END,
    CASE WHEN gs IN (1, 3, 4) THEN jsonb_build_object('api_key', format('seeded-key-%s', gs)) ELSE '{}'::jsonb END,
    CASE
        WHEN gs % 3 = 0 THEN 'hourly'
        WHEN gs % 2 = 0 THEN 'daily'
        ELSE 'every_6h'
    END,
    CASE WHEN gs = 4 THEN 'high' WHEN gs = 1 THEN 'critical' ELSE 'medium' END,
    CASE WHEN gs = 1 THEN 0.94 WHEN gs = 4 THEN 0.88 ELSE 0.80 END,
    ARRAY['seeded', 'threat-feed'],
    ARRAY['ip', 'domain', 'url', 'file_hash_sha256'],
    gs <> 6,
    CASE WHEN gs = 6 THEN 'paused' WHEN gs = 5 THEN 'error' ELSE 'active' END,
    now() - make_interval(hours => gs * 3),
    CASE WHEN gs = 5 THEN 'failed' ELSE 'success' END,
    CASE WHEN gs = 5 THEN 'Seeded feed parsing error.' ELSE NULL END,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => 20 - gs),
    now() - make_interval(hours => gs)
FROM generate_series(1, 6) gs
ON CONFLICT (tenant_id, name) DO UPDATE SET
    type = EXCLUDED.type,
    url = EXCLUDED.url,
    auth_type = EXCLUDED.auth_type,
    auth_config = EXCLUDED.auth_config,
    sync_interval = EXCLUDED.sync_interval,
    default_severity = EXCLUDED.default_severity,
    default_confidence = EXCLUDED.default_confidence,
    default_tags = EXCLUDED.default_tags,
    indicator_types = EXCLUDED.indicator_types,
    enabled = EXCLUDED.enabled,
    status = EXCLUDED.status,
    last_sync_at = EXCLUDED.last_sync_at,
    last_sync_status = EXCLUDED.last_sync_status,
    last_error = EXCLUDED.last_error,
    updated_at = EXCLUDED.updated_at;

INSERT INTO threat_feed_sync_history (
    id, tenant_id, feed_id, status, indicators_parsed, indicators_imported, indicators_skipped,
    indicators_failed, duration_ms, error_message, metadata, started_at, completed_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'threat-feed-sync-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'threat-feed-config-' || (((gs - 1) % 6) + 1)),
    CASE WHEN gs % 9 = 0 THEN 'failed' WHEN gs % 5 = 0 THEN 'partial' ELSE 'success' END,
    180 + (gs * 9),
    150 + (gs * 8),
    12 + (gs % 25),
    CASE WHEN gs % 9 = 0 THEN 4 ELSE 0 END,
    60000 + (gs * 1200),
    CASE WHEN gs % 9 = 0 THEN 'Seeded upstream timeout.' ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 14 - (gs % 10), hours => gs),
    now() - make_interval(days => 14 - (gs % 10), hours => gs) + interval '4 minutes'
FROM generate_series(1, 30) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO assets (
    id, tenant_id, name, type, ip_address, hostname, mac_address, os, os_version, owner, department,
    criticality, status, discovered_at, last_seen_at, metadata, tags, created_at, updated_at,
    created_by, updated_by, discovery_source, location, last_scan_id
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Asset %s', lpad(gs::text, 5, '0')),
    CASE (gs - 1) % 8
        WHEN 0 THEN 'server'
        WHEN 1 THEN 'endpoint'
        WHEN 2 THEN 'network_device'
        WHEN 3 THEN 'cloud_resource'
        WHEN 4 THEN 'iot_device'
        WHEN 5 THEN 'application'
        WHEN 6 THEN 'database'
        ELSE 'container'
    END::asset_type,
    format('10.230.%s.%s', ((gs - 1) / 250) % 250, ((gs - 1) % 250) + 1)::inet,
    format('seeded-asset-%s.demo.local', lpad(gs::text, 5, '0')),
    format(
        '02:%s:%s:%s:%s:%s',
        lpad(to_hex((gs / 65536) % 256), 2, '0'),
        lpad(to_hex((gs / 4096) % 256), 2, '0'),
        lpad(to_hex((gs / 256) % 256), 2, '0'),
        lpad(to_hex((gs / 16) % 256), 2, '0'),
        lpad(to_hex(gs % 256), 2, '0')
    ),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'Ubuntu'
        WHEN 1 THEN 'Windows'
        WHEN 2 THEN 'RHEL'
        WHEN 3 THEN 'Debian'
        ELSE 'Alpine'
    END,
    CASE WHEN gs % 5 = 0 THEN '22.04' WHEN gs % 2 = 0 THEN '11' ELSE '2025.3' END,
    CASE ((gs - 1) % 5) + 1
        WHEN 1 THEN 'Security Operations'
        WHEN 2 THEN 'Data Platform'
        WHEN 3 THEN 'Infrastructure'
        WHEN 4 THEN 'Product Engineering'
        ELSE 'Executive Office'
    END,
    CASE ((gs - 1) % 5) + 1
        WHEN 1 THEN 'Security'
        WHEN 2 THEN 'Data'
        WHEN 3 THEN 'Infrastructure'
        WHEN 4 THEN 'Engineering'
        ELSE 'Executive'
    END,
    CASE
        WHEN gs % 13 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END::asset_criticality,
    CASE
        WHEN gs % 29 = 0 THEN 'inactive'
        WHEN gs % 41 = 0 THEN 'unknown'
        ELSE 'active'
    END::asset_status,
    now() - make_interval(days => 120 - (gs % 90)),
    now() - make_interval(hours => gs % 96),
    jsonb_build_object(
        'environment', CASE WHEN gs % 5 = 0 THEN 'prod' WHEN gs % 2 = 0 THEN 'staging' ELSE 'dev' END,
        'business_service', format('seeded-service-%s', ((gs - 1) % 32) + 1),
        'seed_key', '{{ .SeedKey }}'
    ),
    ARRAY[
        'seeded',
        CASE WHEN gs % 5 = 0 THEN 'internet-facing' ELSE 'internal' END,
        CASE WHEN gs % 3 = 0 THEN 'crown-jewel' ELSE 'standard' END
    ],
    now() - make_interval(days => 120 - (gs % 90)),
    now() - make_interval(hours => gs % 96),
    '{{ .SecurityManagerUserID }}'::uuid,
    '{{ .SecurityManagerUserID }}'::uuid,
    CASE (gs - 1) % 5
        WHEN 0 THEN 'network_scan'
        WHEN 1 THEN 'cloud_scan'
        WHEN 2 THEN 'agent'
        WHEN 3 THEN 'import'
        ELSE 'manual'
    END,
    CASE ((gs - 1) % 6) + 1
        WHEN 1 THEN 'Lagos DC'
        WHEN 2 THEN 'US East'
        WHEN 3 THEN 'US West'
        WHEN 4 THEN 'Frankfurt'
        WHEN 5 THEN 'Azure Nigeria'
        ELSE 'Remote Edge'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-scan-history-' || (((gs - 1) % {{ .Scale.CTEMAssessmentCount }}) + 1))
FROM generate_series(1, {{ .Scale.AssetCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    ip_address = EXCLUDED.ip_address,
    hostname = EXCLUDED.hostname,
    mac_address = EXCLUDED.mac_address,
    os = EXCLUDED.os,
    os_version = EXCLUDED.os_version,
    owner = EXCLUDED.owner,
    department = EXCLUDED.department,
    criticality = EXCLUDED.criticality,
    status = EXCLUDED.status,
    discovered_at = EXCLUDED.discovered_at,
    last_seen_at = EXCLUDED.last_seen_at,
    metadata = EXCLUDED.metadata,
    tags = EXCLUDED.tags,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    discovery_source = EXCLUDED.discovery_source,
    location = EXCLUDED.location,
    last_scan_id = EXCLUDED.last_scan_id,
    deleted_at = NULL;

INSERT INTO asset_relationships (
    id, tenant_id, source_asset_id, target_asset_id, relationship_type, metadata, created_at, created_by
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'asset-relationship-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || ((gs % {{ .Scale.AssetCount }}) + 1)),
    CASE (gs - 1) % 7
        WHEN 0 THEN 'connects_to'
        WHEN 1 THEN 'depends_on'
        WHEN 2 THEN 'managed_by'
        WHEN 3 THEN 'runs_on'
        WHEN 4 THEN 'backs_up'
        WHEN 5 THEN 'load_balances'
        ELSE 'hosts'
    END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 30 - (gs % 20)),
    '{{ .SecurityManagerUserID }}'::uuid
FROM generate_series(1, LEAST({{ .Scale.AssetCount }} - 1, 1800)) gs
ON CONFLICT (id) DO UPDATE SET
    relationship_type = EXCLUDED.relationship_type,
    metadata = EXCLUDED.metadata;

INSERT INTO vulnerabilities (
    id, tenant_id, asset_id, cve_id, title, description, severity, cvss_score, cvss_vector, status,
    discovered_at, resolved_at, due_date, assigned_to, metadata, created_at, updated_at, created_by,
    updated_by, source, remediation, proof
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-vulnerability-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
    format('CVE-2026-%06s', gs),
    format('Seeded Vulnerability %s', lpad(gs::text, 5, '0')),
    'Seeded vulnerability for exposure management, remediation, and CTEM prioritization demos.',
    CASE
        WHEN gs % 17 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END::severity_level,
    CASE
        WHEN gs % 17 = 0 THEN 9.8
        WHEN gs % 7 = 0 THEN 8.4
        WHEN gs % 3 = 0 THEN 6.7
        ELSE 4.3
    END,
    'CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H',
    CASE
        WHEN gs % 19 = 0 THEN 'resolved'
        WHEN gs % 13 = 0 THEN 'in_progress'
        WHEN gs % 11 = 0 THEN 'accepted'
        ELSE 'open'
    END::vulnerability_status,
    now() - make_interval(days => 60 - (gs % 45)),
    CASE WHEN gs % 19 = 0 THEN now() - make_interval(days => 2) ELSE NULL END,
    current_date + ((gs % 30) + 5),
    CASE (gs - 1) % 4
        WHEN 0 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .DataStewardUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    jsonb_build_object('source', 'seeded', 'asset_index', ((gs - 1) % {{ .Scale.AssetCount }}) + 1),
    now() - make_interval(days => 60 - (gs % 45)),
    now() - make_interval(hours => gs % 120),
    '{{ .SecurityManagerUserID }}'::uuid,
    '{{ .SecurityManagerUserID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'scan_tool'
        WHEN 1 THEN 'cve_enrichment'
        WHEN 2 THEN 'manual'
        ELSE 'penetration_test'
    END,
    'Apply vendor patch and validate service restart.',
    format('Seeded PoC reference %s', gs)
FROM generate_series(1, {{ .Scale.VulnerabilityCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    asset_id = EXCLUDED.asset_id,
    cve_id = EXCLUDED.cve_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    cvss_score = EXCLUDED.cvss_score,
    status = EXCLUDED.status,
    discovered_at = EXCLUDED.discovered_at,
    resolved_at = EXCLUDED.resolved_at,
    due_date = EXCLUDED.due_date,
    assigned_to = EXCLUDED.assigned_to,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    source = EXCLUDED.source,
    remediation = EXCLUDED.remediation,
    proof = EXCLUDED.proof,
    deleted_at = NULL;

INSERT INTO detection_rules (
    id, tenant_id, name, description, rule_type, rule_content, severity, mitre_technique_ids, enabled,
    last_triggered_at, trigger_count, created_by, created_at, updated_at, updated_by, mitre_tactic_ids,
    base_confidence, false_positive_count, true_positive_count, tags, is_template, template_id
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'detection-rule-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Detection Rule %s', lpad(gs::text, 3, '0')),
    'Seeded detection rule for SIEM, UEBA, and threat correlation demonstrations.',
    CASE (gs - 1) % 4
        WHEN 0 THEN 'sigma'
        WHEN 1 THEN 'threshold'
        WHEN 2 THEN 'correlation'
        ELSE 'anomaly'
    END,
    jsonb_build_object(
        'logic', format('seeded condition %s', gs),
        'window_minutes', 5 + (gs % 20),
        'threshold', 10 + (gs % 25)
    ),
    CASE
        WHEN gs % 11 = 0 THEN 'critical'
        WHEN gs % 5 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    ARRAY[format('T1%03s', 20 + (gs % 80))],
    gs % 17 <> 0,
    now() - make_interval(hours => gs % 240),
    40 + (gs * 3),
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => 45 - (gs % 30)),
    now() - make_interval(hours => gs % 96),
    '{{ .SecurityManagerUserID }}'::uuid,
    ARRAY[CASE WHEN gs % 2 = 0 THEN 'TA0001' ELSE 'TA0008' END],
    CASE WHEN gs % 7 = 0 THEN 0.92 WHEN gs % 3 = 0 THEN 0.84 ELSE 0.73 END,
    CASE WHEN gs % 13 = 0 THEN 3 ELSE 0 END,
    20 + (gs % 40),
    ARRAY['seeded', CASE WHEN gs % 2 = 0 THEN 'siem' ELSE 'behavioral' END],
    false,
    NULL
FROM generate_series(1, {{ .Scale.DetectionRuleCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    rule_type = EXCLUDED.rule_type,
    rule_content = EXCLUDED.rule_content,
    severity = EXCLUDED.severity,
    mitre_technique_ids = EXCLUDED.mitre_technique_ids,
    enabled = EXCLUDED.enabled,
    last_triggered_at = EXCLUDED.last_triggered_at,
    trigger_count = EXCLUDED.trigger_count,
    mitre_tactic_ids = EXCLUDED.mitre_tactic_ids,
    base_confidence = EXCLUDED.base_confidence,
    false_positive_count = EXCLUDED.false_positive_count,
    true_positive_count = EXCLUDED.true_positive_count,
    tags = EXCLUDED.tags,
    updated_at = EXCLUDED.updated_at,
    deleted_at = NULL;

INSERT INTO threats (
    id, tenant_id, type, name, description, severity, confidence_score, source, indicators,
    mitre_technique_id, mitre_tactic, status, detected_at, resolved_at, metadata, created_at,
    updated_at, created_by, updated_by, threat_actor, campaign, mitre_tactic_ids,
    mitre_technique_ids, affected_asset_count, alert_count, first_seen_at, last_seen_at,
    contained_at, tags
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-threat-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 10
        WHEN 0 THEN 'malware'
        WHEN 1 THEN 'phishing'
        WHEN 2 THEN 'apt'
        WHEN 3 THEN 'ransomware'
        WHEN 4 THEN 'ddos'
        WHEN 5 THEN 'insider_threat'
        WHEN 6 THEN 'supply_chain'
        WHEN 7 THEN 'zero_day'
        WHEN 8 THEN 'brute_force'
        ELSE 'other'
    END,
    format('Seeded Threat %s', lpad(gs::text, 3, '0')),
    'Seeded threat campaign for hunting, correlation, and executive briefing scenarios.',
    CASE
        WHEN gs % 13 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    round((0.62 + ((gs % 30)::numeric / 100)), 4),
    CASE WHEN gs % 4 = 0 THEN 'vendor' WHEN gs % 3 = 0 THEN 'osint' ELSE 'internal' END,
    jsonb_build_array(
        jsonb_build_object('type', 'ip', 'value', format('185.10.%s.%s', gs % 200, (gs * 3) % 250 + 1)),
        jsonb_build_object('type', 'domain', 'value', format('seeded-threat-%s.local', gs))
    ),
    format('T1%03s', 20 + (gs % 80)),
    CASE WHEN gs % 2 = 0 THEN 'Credential Access' ELSE 'Initial Access' END,
    CASE
        WHEN gs % 19 = 0 THEN 'closed'
        WHEN gs % 11 = 0 THEN 'eradicated'
        WHEN gs % 5 = 0 THEN 'contained'
        ELSE 'active'
    END,
    now() - make_interval(days => 20 - (gs % 14)),
    CASE WHEN gs % 19 = 0 THEN now() - make_interval(days => 1) ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 20 - (gs % 14)),
    now() - make_interval(hours => gs % 96),
    '{{ .SecurityManagerUserID }}'::uuid,
    '{{ .SecurityManagerUserID }}'::uuid,
    format('Seeded Actor %s', ((gs - 1) % 12) + 1),
    format('Campaign-%s', ((gs - 1) % 18) + 1),
    ARRAY[CASE WHEN gs % 2 = 0 THEN 'TA0006' ELSE 'TA0001' END],
    ARRAY[format('T1%03s', 20 + (gs % 80))],
    2 + (gs % 8),
    3 + (gs % 12),
    now() - make_interval(days => 24 - (gs % 18)),
    now() - make_interval(hours => gs % 48),
    CASE WHEN gs % 5 = 0 THEN now() - interval '2 hours' ELSE NULL END,
    ARRAY['seeded', CASE WHEN gs % 3 = 0 THEN 'campaign' ELSE 'intel' END]
FROM generate_series(1, {{ .Scale.ThreatCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    type = EXCLUDED.type,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    confidence_score = EXCLUDED.confidence_score,
    source = EXCLUDED.source,
    indicators = EXCLUDED.indicators,
    status = EXCLUDED.status,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at,
    threat_actor = EXCLUDED.threat_actor,
    campaign = EXCLUDED.campaign,
    mitre_tactic_ids = EXCLUDED.mitre_tactic_ids,
    mitre_technique_ids = EXCLUDED.mitre_technique_ids,
    affected_asset_count = EXCLUDED.affected_asset_count,
    alert_count = EXCLUDED.alert_count,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    contained_at = EXCLUDED.contained_at,
    tags = EXCLUDED.tags,
    deleted_at = NULL;

INSERT INTO threat_indicators (
    id, tenant_id, threat_id, type, value, confidence, source, first_seen_at, last_seen_at,
    created_at, description, severity, active, expires_at, tags, metadata, created_by, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'threat-indicator-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-threat-' || (((gs - 1) % {{ .Scale.ThreatCount }}) + 1)),
    CASE (gs - 1) % 11
        WHEN 0 THEN 'ip'
        WHEN 1 THEN 'domain'
        WHEN 2 THEN 'url'
        WHEN 3 THEN 'email'
        WHEN 4 THEN 'file_hash_md5'
        WHEN 5 THEN 'file_hash_sha1'
        WHEN 6 THEN 'file_hash_sha256'
        WHEN 7 THEN 'certificate'
        WHEN 8 THEN 'registry_key'
        WHEN 9 THEN 'user_agent'
        ELSE 'cidr'
    END,
    CASE (gs - 1) % 11
        WHEN 0 THEN format('172.31.%s.%s', gs % 250, (gs * 7) % 250 + 1)
        WHEN 1 THEN format('indicator-%s.seeded.local', gs)
        WHEN 2 THEN format('https://indicator-%s.seeded.local/login', gs)
        WHEN 3 THEN format('indicator-%s@seeded.local', gs)
        WHEN 4 THEN md5('md5-seeded-' || gs)
        WHEN 5 THEN md5('sha1-seeded-a-' || gs) || substr(md5('sha1-seeded-b-' || gs), 1, 8)
        WHEN 6 THEN md5('sha256-seeded-a-' || gs) || md5('sha256-seeded-b-' || gs)
        WHEN 7 THEN format('seeded-cert-%s', gs)
        WHEN 8 THEN format('HKLM\\\\Software\\\\Seeded\\\\IOC\\\\%s', gs)
        WHEN 9 THEN format('SeededAgent/%s.%s.%s', (gs % 10) + 1, (gs % 5) + 1, gs)
        ELSE format('10.%s.%s.0/24', ((gs - 1) / 250) % 200, ((gs - 1) % 250))
    END,
    CASE WHEN gs % 7 = 0 THEN 0.95 WHEN gs % 3 = 0 THEN 0.87 ELSE 0.78 END,
    CASE (gs - 1) % 5
        WHEN 0 THEN 'stix_feed'
        WHEN 1 THEN 'osint'
        WHEN 2 THEN 'vendor'
        WHEN 3 THEN 'internal'
        ELSE 'manual'
    END,
    now() - make_interval(days => 35 - (gs % 20)),
    now() - make_interval(hours => gs % 72),
    now() - make_interval(days => 35 - (gs % 20)),
    'Seeded IOC for threat feed, enrichment, and matching demos.',
    CASE WHEN gs % 13 = 0 THEN 'critical' WHEN gs % 5 = 0 THEN 'high' ELSE 'medium' END,
    gs % 17 <> 0,
    CASE WHEN gs % 17 = 0 THEN now() + interval '30 days' ELSE NULL END,
    ARRAY['seeded', 'ioc'],
    jsonb_build_object('feed', (((gs - 1) % 6) + 1)),
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.ThreatIndicatorCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    threat_id = EXCLUDED.threat_id,
    type = EXCLUDED.type,
    value = EXCLUDED.value,
    confidence = EXCLUDED.confidence,
    source = EXCLUDED.source,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    active = EXCLUDED.active,
    expires_at = EXCLUDED.expires_at,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at;

INSERT INTO alerts (
    id, tenant_id, rule_id, title, description, severity, status, confidence_score, explanation,
    contributing_factors, asset_ids, assigned_to, acknowledged_at, resolved_at, resolution_notes,
    created_at, updated_at, created_by, updated_by, source, asset_id, assigned_at, escalated_to,
    escalated_at, mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
    event_count, first_event_at, last_event_at, false_positive_reason, tags, metadata
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'detection-rule-' || (((gs - 1) % {{ .Scale.DetectionRuleCount }}) + 1)),
    format('Seeded Alert %s', lpad(gs::text, 5, '0')),
    'Seeded security alert for triage, investigation, and response workflow demos.',
    CASE
        WHEN gs % 17 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    CASE
        WHEN gs % 23 = 0 THEN 'false_positive'
        WHEN gs % 19 = 0 THEN 'resolved'
        WHEN gs % 11 = 0 THEN 'escalated'
        WHEN gs % 7 = 0 THEN 'investigating'
        WHEN gs % 5 = 0 THEN 'acknowledged'
        ELSE 'new'
    END,
    CASE WHEN gs % 11 = 0 THEN 0.94 WHEN gs % 3 = 0 THEN 0.83 ELSE 0.68 END,
    jsonb_build_object('summary', 'Seeded anomaly confidence', 'top_driver', 'burst_login'),
    jsonb_build_array(
        jsonb_build_object('factor', 'burst_login', 'score', 0.74),
        jsonb_build_object('factor', 'geo_anomaly', 'score', 0.59)
    ),
    CASE
        WHEN gs % 7 = 0 THEN ARRAY[
            uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
            uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs) % {{ .Scale.AssetCount }}) + 1))
        ]
        ELSE ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1))]
    END,
    CASE
        WHEN gs % 19 = 0 THEN '{{ .AuditorUserID }}'::uuid
        WHEN gs % 2 = 0 THEN '{{ .SecurityManagerUserID }}'::uuid
        ELSE '{{ .MainAdminUserID }}'::uuid
    END,
    CASE WHEN gs % 5 = 0 THEN now() - make_interval(hours => gs % 48) ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN now() - make_interval(hours => 1) ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN 'Seeded containment and verification completed.' ELSE NULL END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 5),
    '{{ .SecurityManagerUserID }}'::uuid,
    '{{ .SecurityManagerUserID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'siem'
        WHEN 1 THEN 'edr'
        WHEN 2 THEN 'ueba'
        ELSE 'xdr'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
    CASE WHEN gs % 5 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 15) ELSE NULL END,
    CASE WHEN gs % 11 = 0 THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 11 = 0 THEN date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 35) ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN 'TA0006' ELSE 'TA0001' END,
    CASE WHEN gs % 2 = 0 THEN 'Credential Access' ELSE 'Initial Access' END,
    format('T1%03s', 20 + (gs % 80)),
    CASE WHEN gs % 2 = 0 THEN 'Valid Accounts' ELSE 'Phishing' END,
    5 + (gs % 40),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 3),
    CASE WHEN gs % 23 = 0 THEN 'Seeded expected admin maintenance window.' ELSE NULL END,
    ARRAY['seeded', CASE WHEN gs % 3 = 0 THEN 'investigate' ELSE 'monitor' END],
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs)
FROM generate_series(1, {{ .Scale.AlertCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    rule_id = EXCLUDED.rule_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    status = EXCLUDED.status,
    confidence_score = EXCLUDED.confidence_score,
    explanation = EXCLUDED.explanation,
    contributing_factors = EXCLUDED.contributing_factors,
    asset_ids = EXCLUDED.asset_ids,
    assigned_to = EXCLUDED.assigned_to,
    acknowledged_at = EXCLUDED.acknowledged_at,
    resolved_at = EXCLUDED.resolved_at,
    resolution_notes = EXCLUDED.resolution_notes,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    source = EXCLUDED.source,
    asset_id = EXCLUDED.asset_id,
    assigned_at = EXCLUDED.assigned_at,
    escalated_to = EXCLUDED.escalated_to,
    escalated_at = EXCLUDED.escalated_at,
    mitre_tactic_id = EXCLUDED.mitre_tactic_id,
    mitre_tactic_name = EXCLUDED.mitre_tactic_name,
    mitre_technique_id = EXCLUDED.mitre_technique_id,
    mitre_technique_name = EXCLUDED.mitre_technique_name,
    event_count = EXCLUDED.event_count,
    first_event_at = EXCLUDED.first_event_at,
    last_event_at = EXCLUDED.last_event_at,
    false_positive_reason = EXCLUDED.false_positive_reason,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    deleted_at = NULL;

INSERT INTO alert_comments (
    id, tenant_id, alert_id, user_id, user_name, user_email, content, is_system, metadata, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'alert-comment-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || (((gs - 1) % {{ .Scale.AlertCount }}) + 1)),
    CASE WHEN gs % 4 = 0 THEN '{{ .MainAdminUserID }}'::uuid ELSE '{{ .SecurityManagerUserID }}'::uuid END,
    CASE WHEN gs % 4 = 0 THEN 'Demo Admin' ELSE 'Security Manager' END,
    CASE WHEN gs % 4 = 0 THEN 'admin@clario.dev' ELSE 'security.manager@clario.dev' END,
    format('Seeded analyst note %s for investigation timeline demonstrations.', gs),
    false,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(hours => gs % 240),
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, GREATEST(40, {{ .Scale.AlertCount }} / 2)) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO alert_timeline (
    id, tenant_id, alert_id, action, actor_id, actor_name, old_value, new_value, description, metadata, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'alert-timeline-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || (((gs - 1) % {{ .Scale.AlertCount }}) + 1)),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'status_changed'
        WHEN 1 THEN 'assigned'
        WHEN 2 THEN 'evidence_added'
        ELSE 'escalated'
    END,
    '{{ .SecurityManagerUserID }}'::uuid,
    'Security Manager',
    CASE WHEN gs % 4 = 0 THEN 'new' ELSE NULL END,
    CASE WHEN gs % 4 = 0 THEN 'investigating' ELSE NULL END,
    format('Seeded alert timeline event %s', gs),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, GREATEST(80, {{ .Scale.AlertCount }})) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO security_events (
    id, tenant_id, timestamp, source, type, severity, source_ip, dest_ip, dest_port, protocol,
    username, process, parent_process, command_line, file_path, file_hash, asset_id, raw_event,
    matched_rules, processed_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'security-event-' || gs),
    '{{ .MainTenantID }}'::uuid,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => (gs % 60)),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'edr'
        WHEN 1 THEN 'firewall'
        WHEN 2 THEN 'iam'
        WHEN 3 THEN 'proxy'
        ELSE 'cloudtrail'
    END,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'process_creation'
        WHEN 1 THEN 'network_connection'
        WHEN 2 THEN 'login'
        WHEN 3 THEN 'dns_query'
        WHEN 4 THEN 'file_write'
        ELSE 'privilege_escalation'
    END,
    CASE
        WHEN gs % 29 = 0 THEN 'critical'
        WHEN gs % 11 = 0 THEN 'high'
        WHEN gs % 5 = 0 THEN 'medium'
        WHEN gs % 3 = 0 THEN 'low'
        ELSE 'info'
    END,
    format('192.168.%s.%s', gs % 200, (gs * 11) % 250 + 1)::inet,
    format('10.42.%s.%s', (gs / 3) % 200, (gs * 5) % 250 + 1)::inet,
    80 + (gs % 400),
    CASE WHEN gs % 3 = 0 THEN 'tcp' WHEN gs % 5 = 0 THEN 'udp' ELSE 'https' END,
    format('seeded-user-%s', ((gs - 1) % 80) + 1),
    CASE WHEN gs % 4 = 0 THEN 'powershell.exe' WHEN gs % 3 = 0 THEN 'cmd.exe' ELSE 'python' END,
    CASE WHEN gs % 4 = 0 THEN 'explorer.exe' WHEN gs % 3 = 0 THEN 'services.exe' ELSE 'systemd' END,
    format('seeded command line %s', gs),
    format('/opt/seeded/bin/file_%s.bin', gs),
    md5('security-event-file-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs, 'event_index', gs),
    ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'detection-rule-' || (((gs - 1) % {{ .Scale.DetectionRuleCount }}) + 1))],
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => ((gs + 5) % 60))
FROM generate_series(1, {{ .Scale.SecurityEventCount }}) gs
ON CONFLICT (id, timestamp) DO NOTHING;

INSERT INTO asset_activity (
    id, tenant_id, asset_id, action, actor_id, actor_name, description, old_value, new_value, metadata, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'asset-activity-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'status_changed'
        WHEN 1 THEN 'owner_changed'
        WHEN 2 THEN 'reclassified'
        WHEN 3 THEN 'tag_added'
        ELSE 'scan_refreshed'
    END,
    '{{ .SecurityManagerUserID }}'::uuid,
    'Security Manager',
    format('Seeded asset activity entry %s', gs),
    CASE WHEN gs % 5 = 0 THEN 'inactive' ELSE NULL END,
    CASE WHEN gs % 5 = 0 THEN 'active' ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(hours => gs % 336)
FROM generate_series(1, {{ .Scale.AssetCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO ctem_assessments (
    id, tenant_id, name, scope, status, started_at, completed_at, findings_count, critical_count,
    high_count, medium_count, low_count, report, created_by, created_at, updated_at, updated_by,
    description, resolved_asset_ids, resolved_asset_count, phases, current_phase, exposure_score,
    score_breakdown, findings_summary, duration_ms, error_message, error_phase, scheduled,
    schedule_cron, parent_assessment_id, tags
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded CTEM Assessment %s', lpad(gs::text, 3, '0')),
    jsonb_build_object('scope_type', 'tenant', 'asset_groups', jsonb_build_array('production', 'internet-facing')),
    CASE
        WHEN gs % 17 = 0 THEN 'failed'
        WHEN gs % 7 = 0 THEN 'validating'
        WHEN gs % 5 = 0 THEN 'discovery'
        ELSE 'completed'
    END,
    now() - make_interval(days => gs),
    CASE WHEN gs % 17 = 0 THEN NULL ELSE now() - make_interval(days => gs) + interval '32 minutes' END,
    GREATEST(1, {{ .Scale.CTEMFindingCount }} / GREATEST({{ .Scale.CTEMAssessmentCount }}, 1)),
    CASE WHEN gs % 4 = 0 THEN 3 ELSE 1 END,
    6 + (gs % 8),
    12 + (gs % 14),
    8 + (gs % 10),
    jsonb_build_object('summary', 'Seeded CTEM assessment report'),
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => gs),
    now() - make_interval(hours => gs % 96),
    '{{ .SecurityManagerUserID }}'::uuid,
    'Seeded continuous exposure assessment for board and analyst reporting.',
    ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1))],
    1,
    jsonb_build_object(
        'discovery', jsonb_build_object('status', 'completed'),
        'prioritizing', jsonb_build_object('status', 'completed'),
        'mobilizing', jsonb_build_object('status', CASE WHEN gs % 17 = 0 THEN 'failed' ELSE 'completed' END)
    ),
    CASE WHEN gs % 17 = 0 THEN 'mobilizing' WHEN gs % 7 = 0 THEN 'validating' ELSE 'completed' END,
    CASE WHEN gs % 17 = 0 THEN 54.0 ELSE 68.0 + (gs % 20) END,
    jsonb_build_object('attack_surface', 24 + (gs % 10), 'vulnerability', 30 + (gs % 15), 'configuration', 18 + (gs % 8)),
    jsonb_build_object('critical', 1 + (gs % 3), 'high', 6 + (gs % 6)),
    CASE WHEN gs % 17 = 0 THEN NULL ELSE 1920000 END,
    CASE WHEN gs % 17 = 0 THEN 'Seeded validation failure.' ELSE NULL END,
    CASE WHEN gs % 17 = 0 THEN 'validating' ELSE NULL END,
    gs % 6 = 0,
    CASE WHEN gs % 6 = 0 THEN '0 4 * * 1' ELSE NULL END,
    CASE WHEN gs > 1 AND gs % 6 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || (gs - 1)) ELSE NULL END,
    ARRAY['seeded', 'ctem']
FROM generate_series(1, {{ .Scale.CTEMAssessmentCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    scope = EXCLUDED.scope,
    status = EXCLUDED.status,
    started_at = EXCLUDED.started_at,
    completed_at = EXCLUDED.completed_at,
    findings_count = EXCLUDED.findings_count,
    critical_count = EXCLUDED.critical_count,
    high_count = EXCLUDED.high_count,
    medium_count = EXCLUDED.medium_count,
    low_count = EXCLUDED.low_count,
    report = EXCLUDED.report,
    updated_at = EXCLUDED.updated_at,
    description = EXCLUDED.description,
    resolved_asset_ids = EXCLUDED.resolved_asset_ids,
    resolved_asset_count = EXCLUDED.resolved_asset_count,
    phases = EXCLUDED.phases,
    current_phase = EXCLUDED.current_phase,
    exposure_score = EXCLUDED.exposure_score,
    score_breakdown = EXCLUDED.score_breakdown,
    findings_summary = EXCLUDED.findings_summary,
    duration_ms = EXCLUDED.duration_ms,
    error_message = EXCLUDED.error_message,
    error_phase = EXCLUDED.error_phase,
    scheduled = EXCLUDED.scheduled,
    schedule_cron = EXCLUDED.schedule_cron,
    parent_assessment_id = EXCLUDED.parent_assessment_id,
    tags = EXCLUDED.tags,
    deleted_at = NULL;

INSERT INTO ctem_findings (
    id, tenant_id, assessment_id, type, category, severity, title, description, evidence,
    affected_asset_ids, affected_asset_count, primary_asset_id, vulnerability_ids, cve_ids,
    business_impact_score, business_impact_factors, exploitability_score, exploitability_factors,
    priority_score, priority_group, priority_rank, validation_status, compensating_controls,
    validation_notes, validated_at, remediation_type, remediation_description, remediation_effort,
    remediation_group_id, estimated_days, status, status_changed_by, status_changed_at, status_notes,
    attack_path, attack_path_length, metadata, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-finding-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || (((gs - 1) % {{ .Scale.CTEMAssessmentCount }}) + 1)),
    CASE (gs - 1) % 8
        WHEN 0 THEN 'vulnerability'
        WHEN 1 THEN 'misconfiguration'
        WHEN 2 THEN 'attack_path'
        WHEN 3 THEN 'exposure'
        WHEN 4 THEN 'weak_credential'
        WHEN 5 THEN 'missing_patch'
        WHEN 6 THEN 'expired_certificate'
        ELSE 'insecure_protocol'
    END,
    CASE WHEN gs % 5 = 0 THEN 'configuration' WHEN gs % 7 = 0 THEN 'architectural' ELSE 'technical' END,
    CASE
        WHEN gs % 19 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    format('Seeded CTEM Finding %s', lpad(gs::text, 5, '0')),
    'Seeded CTEM finding for prioritization, remediation grouping, and reporting demos.',
    jsonb_build_object('evidence_type', 'scan', 'seeded', true),
    CASE
        WHEN gs % 9 = 0 THEN ARRAY[
            uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
            uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs) % {{ .Scale.AssetCount }}) + 1))
        ]
        ELSE ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1))]
    END,
    CASE WHEN gs % 9 = 0 THEN 2 ELSE 1 END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1)),
    ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-vulnerability-' || (((gs - 1) % {{ .Scale.VulnerabilityCount }}) + 1))],
    ARRAY[format('CVE-2026-%06s', (((gs - 1) % {{ .Scale.VulnerabilityCount }}) + 1))],
    CASE WHEN gs % 17 = 0 THEN 96.0 WHEN gs % 7 = 0 THEN 83.0 ELSE 58.0 + (gs % 20) END,
    jsonb_build_array(jsonb_build_object('factor', 'business_process_impact'), jsonb_build_object('factor', 'internet_exposure')),
    CASE WHEN gs % 17 = 0 THEN 91.0 WHEN gs % 7 = 0 THEN 78.0 ELSE 52.0 + (gs % 18) END,
    jsonb_build_array(jsonb_build_object('factor', 'public_exploit'), jsonb_build_object('factor', 'reachable_asset')),
    CASE WHEN gs % 17 = 0 THEN 95.0 WHEN gs % 7 = 0 THEN 82.0 ELSE 60.0 + (gs % 20) END,
    CASE WHEN gs % 17 = 0 THEN 1 WHEN gs % 7 = 0 THEN 2 WHEN gs % 3 = 0 THEN 3 ELSE 4 END,
    gs,
    CASE WHEN gs % 11 = 0 THEN 'validated' WHEN gs % 13 = 0 THEN 'compensated' ELSE 'pending' END,
    CASE WHEN gs % 13 = 0 THEN ARRAY['WAF policy', 'MFA enforcement'] ELSE ARRAY[]::text[] END,
    CASE WHEN gs % 11 = 0 THEN 'Seeded validation completed.' ELSE NULL END,
    CASE WHEN gs % 11 = 0 THEN now() - interval '2 days' ELSE NULL END,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'patch'
        WHEN 1 THEN 'configuration'
        WHEN 2 THEN 'architecture'
        WHEN 3 THEN 'upgrade'
        WHEN 4 THEN 'decommission'
        ELSE 'accept_risk'
    END,
    'Seeded remediation recommendation and effort estimate.',
    CASE WHEN gs % 5 = 0 THEN 'high' WHEN gs % 2 = 0 THEN 'medium' ELSE 'low' END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-remediation-group-' || ((((gs - 1) % GREATEST({{ .Scale.CTEMAssessmentCount }} * 3, 1))) + 1)),
    3 + (gs % 12),
    CASE
        WHEN gs % 23 = 0 THEN 'accepted_risk'
        WHEN gs % 19 = 0 THEN 'remediated'
        WHEN gs % 7 = 0 THEN 'in_remediation'
        ELSE 'open'
    END,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(hours => gs % 96),
    CASE WHEN gs % 23 = 0 THEN 'Seeded risk acceptance.' WHEN gs % 19 = 0 THEN 'Seeded fix validated.' ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN jsonb_build_array(jsonb_build_object('hop', 1), jsonb_build_object('hop', 2)) ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN 2 ELSE NULL END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 20 - (gs % 14)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.CTEMFindingCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    assessment_id = EXCLUDED.assessment_id,
    type = EXCLUDED.type,
    category = EXCLUDED.category,
    severity = EXCLUDED.severity,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    evidence = EXCLUDED.evidence,
    affected_asset_ids = EXCLUDED.affected_asset_ids,
    affected_asset_count = EXCLUDED.affected_asset_count,
    primary_asset_id = EXCLUDED.primary_asset_id,
    vulnerability_ids = EXCLUDED.vulnerability_ids,
    cve_ids = EXCLUDED.cve_ids,
    business_impact_score = EXCLUDED.business_impact_score,
    business_impact_factors = EXCLUDED.business_impact_factors,
    exploitability_score = EXCLUDED.exploitability_score,
    exploitability_factors = EXCLUDED.exploitability_factors,
    priority_score = EXCLUDED.priority_score,
    priority_group = EXCLUDED.priority_group,
    priority_rank = EXCLUDED.priority_rank,
    validation_status = EXCLUDED.validation_status,
    compensating_controls = EXCLUDED.compensating_controls,
    validation_notes = EXCLUDED.validation_notes,
    validated_at = EXCLUDED.validated_at,
    remediation_type = EXCLUDED.remediation_type,
    remediation_description = EXCLUDED.remediation_description,
    remediation_effort = EXCLUDED.remediation_effort,
    remediation_group_id = EXCLUDED.remediation_group_id,
    estimated_days = EXCLUDED.estimated_days,
    status = EXCLUDED.status,
    status_changed_by = EXCLUDED.status_changed_by,
    status_changed_at = EXCLUDED.status_changed_at,
    status_notes = EXCLUDED.status_notes,
    attack_path = EXCLUDED.attack_path,
    attack_path_length = EXCLUDED.attack_path_length,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at;

INSERT INTO ctem_remediation_groups (
    id, tenant_id, assessment_id, title, description, type, finding_count, affected_asset_count,
    cve_ids, max_priority_score, priority_group, effort, estimated_days, score_reduction, status,
    workflow_instance_id, target_date, started_at, completed_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-remediation-group-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || (((gs - 1) % {{ .Scale.CTEMAssessmentCount }}) + 1)),
    format('Seeded CTEM Remediation Group %s', lpad(gs::text, 3, '0')),
    'Seeded grouped remediation plan for related CTEM findings.',
    CASE (gs - 1) % 6
        WHEN 0 THEN 'patch'
        WHEN 1 THEN 'configuration'
        WHEN 2 THEN 'architecture'
        WHEN 3 THEN 'upgrade'
        WHEN 4 THEN 'decommission'
        ELSE 'accept_risk'
    END,
    3 + (gs % 12),
    2 + (gs % 6),
    ARRAY[format('CVE-2026-%06s', ((gs - 1) % {{ .Scale.VulnerabilityCount }}) + 1)],
    CASE WHEN gs % 11 = 0 THEN 94.0 WHEN gs % 3 = 0 THEN 78.0 ELSE 62.0 + (gs % 20) END,
    CASE WHEN gs % 11 = 0 THEN 1 WHEN gs % 4 = 0 THEN 2 ELSE 3 END,
    CASE WHEN gs % 5 = 0 THEN 'high' WHEN gs % 2 = 0 THEN 'medium' ELSE 'low' END,
    4 + (gs % 18),
    12.0 + (gs % 25),
    CASE WHEN gs % 13 = 0 THEN 'completed' WHEN gs % 5 = 0 THEN 'in_progress' ELSE 'planned' END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || (((gs - 1) % {{ .Scale.WorkflowInstanceCount }}) + 1)),
    current_date + ((gs % 30) + 10),
    CASE WHEN gs % 5 = 0 THEN now() - interval '3 days' ELSE NULL END,
    CASE WHEN gs % 13 = 0 THEN now() - interval '1 day' ELSE NULL END,
    now() - make_interval(days => 21 - (gs % 14)),
    now() - make_interval(hours => gs % 96)
FROM generate_series(1, GREATEST({{ .Scale.CTEMAssessmentCount }} * 3, 12)) gs
ON CONFLICT (id) DO UPDATE SET
    assessment_id = EXCLUDED.assessment_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    finding_count = EXCLUDED.finding_count,
    affected_asset_count = EXCLUDED.affected_asset_count,
    cve_ids = EXCLUDED.cve_ids,
    max_priority_score = EXCLUDED.max_priority_score,
    priority_group = EXCLUDED.priority_group,
    effort = EXCLUDED.effort,
    estimated_days = EXCLUDED.estimated_days,
    score_reduction = EXCLUDED.score_reduction,
    status = EXCLUDED.status,
    workflow_instance_id = EXCLUDED.workflow_instance_id,
    target_date = EXCLUDED.target_date,
    started_at = EXCLUDED.started_at,
    completed_at = EXCLUDED.completed_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO exposure_score_snapshots (
    id, tenant_id, score, breakdown, asset_count, vuln_count, finding_count, assessment_id,
    snapshot_type, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'exposure-score-snapshot-' || gs),
    '{{ .MainTenantID }}'::uuid,
    58.0 + (gs % 32),
    jsonb_build_object('attack_surface', 20 + (gs % 10), 'vulnerabilities', 24 + (gs % 12), 'misconfigurations', 15 + (gs % 8)),
    {{ .Scale.AssetCount }},
    {{ .Scale.VulnerabilityCount }},
    {{ .Scale.CTEMFindingCount }},
    CASE WHEN gs <= {{ .Scale.CTEMAssessmentCount }} THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || gs) ELSE NULL END,
    CASE WHEN gs % 5 = 0 THEN 'manual' WHEN gs % 2 = 0 THEN 'daily' ELSE 'assessment' END,
    now() - make_interval(days => gs)
FROM generate_series(1, GREATEST({{ .Scale.CTEMAssessmentCount }}, 12)) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO risk_score_history (
    id, tenant_id, overall_score, grade, vulnerability_score, threat_score, config_score,
    surface_score, compliance_score, total_assets, total_open_vulns, total_open_alerts,
    total_active_threats, components, top_contributors, recommendations, snapshot_type,
    trigger_event, calculated_on, calculated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'risk-score-history-' || gs),
    '{{ .MainTenantID }}'::uuid,
    56.0 + (gs % 28),
    CASE
        WHEN 56.0 + (gs % 28) >= 85 THEN 'A'
        WHEN 56.0 + (gs % 28) >= 75 THEN 'B'
        WHEN 56.0 + (gs % 28) >= 65 THEN 'C'
        WHEN 56.0 + (gs % 28) >= 55 THEN 'D'
        ELSE 'F'
    END,
    50.0 + (gs % 30),
    44.0 + (gs % 35),
    48.0 + (gs % 25),
    42.0 + (gs % 22),
    60.0 + (gs % 18),
    {{ .Scale.AssetCount }},
    {{ .Scale.VulnerabilityCount }} - (gs % 50),
    {{ .Scale.AlertCount }} - (gs % 30),
    {{ .Scale.ThreatCount }} - (gs % 10),
    jsonb_build_object('seeded', true, 'trend_index', gs),
    jsonb_build_array(jsonb_build_object('type', 'vulnerabilities', 'value', 18 + (gs % 10))),
    jsonb_build_array('Increase patch cadence', 'Reduce privileged access', 'Harden internet-facing assets'),
    'event_triggered',
    CASE WHEN gs % 3 = 0 THEN 'alert_burst' ELSE 'daily_rollup' END,
    current_date - gs,
    now() - make_interval(days => gs)
FROM generate_series(1, 90) gs
ON CONFLICT (id) DO UPDATE SET
    overall_score = EXCLUDED.overall_score,
    grade = EXCLUDED.grade,
    vulnerability_score = EXCLUDED.vulnerability_score,
    threat_score = EXCLUDED.threat_score,
    config_score = EXCLUDED.config_score,
    surface_score = EXCLUDED.surface_score,
    compliance_score = EXCLUDED.compliance_score,
    total_assets = EXCLUDED.total_assets,
    total_open_vulns = EXCLUDED.total_open_vulns,
    total_open_alerts = EXCLUDED.total_open_alerts,
    total_active_threats = EXCLUDED.total_active_threats,
    components = EXCLUDED.components,
    top_contributors = EXCLUDED.top_contributors,
    recommendations = EXCLUDED.recommendations,
    trigger_event = EXCLUDED.trigger_event,
    calculated_on = EXCLUDED.calculated_on,
    calculated_at = EXCLUDED.calculated_at;

INSERT INTO remediation_actions (
    id, tenant_id, alert_id, vulnerability_id, type, status, execution_mode, dry_run_result,
    execution_result, rollback_data, approved_by, executed_by, executed_at, completed_at,
    created_at, updated_at, created_by, updated_by, assessment_id, ctem_finding_id,
    remediation_group_id, severity, title, description, plan, affected_asset_ids,
    affected_asset_count, submitted_by, submitted_at, approved_at, rejected_by, rejected_at,
    rejection_reason, approval_notes, requires_approval_from, dry_run_at, dry_run_duration_ms,
    pre_execution_state, execution_started_at, execution_completed_at, execution_duration_ms,
    verification_result, verified_by, verified_at, rollback_result, rollback_reason,
    rollback_approved_by, rolled_back_at, rollback_deadline, workflow_instance_id, tags,
    metadata, created_by_name
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'remediation-action-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE WHEN gs % 2 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || (((gs - 1) % {{ .Scale.AlertCount }}) + 1)) ELSE NULL END,
    CASE WHEN gs % 3 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-vulnerability-' || (((gs - 1) % {{ .Scale.VulnerabilityCount }}) + 1)) ELSE NULL END,
    CASE (gs - 1) % 8
        WHEN 0 THEN 'patch'
        WHEN 1 THEN 'config_change'
        WHEN 2 THEN 'block_ip'
        WHEN 3 THEN 'isolate_asset'
        WHEN 4 THEN 'firewall_rule'
        WHEN 5 THEN 'access_revoke'
        WHEN 6 THEN 'certificate_renew'
        ELSE 'custom'
    END,
    CASE
        WHEN gs % 29 = 0 THEN 'rolled_back'
        WHEN gs % 19 = 0 THEN 'verified'
        WHEN gs % 13 = 0 THEN 'executed'
        WHEN gs % 11 = 0 THEN 'executing'
        WHEN gs % 7 = 0 THEN 'approved'
        ELSE 'pending_approval'
    END,
    CASE WHEN gs % 5 = 0 THEN 'automated' WHEN gs % 2 = 0 THEN 'semi_automated' ELSE 'manual' END,
    CASE WHEN gs % 7 = 0 THEN jsonb_build_object('safe', true, 'duration_ms', 90000) ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN jsonb_build_object('success', true, 'changed', true) ELSE NULL END,
    CASE WHEN gs % 29 = 0 THEN jsonb_build_object('rollback_token', format('rb-%s', gs)) ELSE NULL END,
    CASE WHEN gs % 7 = 0 THEN '{{ .MainAdminUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN '{{ .SecurityManagerUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN now() - make_interval(hours => gs % 48) ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN now() - make_interval(hours => 1) ELSE NULL END,
    now() - make_interval(days => 18 - (gs % 14)),
    now() - make_interval(hours => gs % 72),
    '{{ .SecurityManagerUserID }}'::uuid,
    '{{ .SecurityManagerUserID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-assessment-' || (((gs - 1) % {{ .Scale.CTEMAssessmentCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-finding-' || (((gs - 1) % {{ .Scale.CTEMFindingCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ctem-remediation-group-' || ((((gs - 1) % GREATEST({{ .Scale.CTEMAssessmentCount }} * 3, 1))) + 1)),
    CASE WHEN gs % 17 = 0 THEN 'critical' WHEN gs % 7 = 0 THEN 'high' WHEN gs % 3 = 0 THEN 'medium' ELSE 'low' END,
    format('Seeded Remediation Action %s', lpad(gs::text, 4, '0')),
    'Seeded remediation action for approval, dry-run, execution, and rollback demonstrations.',
    jsonb_build_object(
        'steps', jsonb_build_array(
            jsonb_build_object('number', 1, 'action', 'validate_scope'),
            jsonb_build_object('number', 2, 'action', 'apply_change'),
            jsonb_build_object('number', 3, 'action', 'verify_outcome')
        ),
        'reversible', true,
        'risk_level', 'medium'
    ),
    ARRAY[uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || (((gs - 1) % {{ .Scale.AssetCount }}) + 1))],
    1,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => 18 - (gs % 14)),
    CASE WHEN gs % 7 = 0 THEN now() - make_interval(days => 2) ELSE NULL END,
    CASE WHEN gs % 31 = 0 THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 31 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 31 = 0 THEN 'Seeded business hold.' ELSE NULL END,
    CASE WHEN gs % 7 = 0 THEN 'Seeded approval granted.' ELSE NULL END,
    CASE WHEN gs % 11 = 0 THEN 'ciso' WHEN gs % 3 = 0 THEN 'tenant_admin' ELSE 'security_manager' END,
    CASE WHEN gs % 7 = 0 THEN now() - make_interval(days => 2) ELSE NULL END,
    CASE WHEN gs % 7 = 0 THEN 90000 ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN jsonb_build_object('snapshot', 'pre-change') ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN now() - make_interval(hours => gs % 48, mins => 15) ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN now() - make_interval(hours => gs % 48) ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN 180000 ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN jsonb_build_object('verified', true) ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN '{{ .AuditorUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN now() - interval '30 minutes' ELSE NULL END,
    CASE WHEN gs % 29 = 0 THEN jsonb_build_object('success', true) ELSE NULL END,
    CASE WHEN gs % 29 = 0 THEN 'Seeded application side effect.' ELSE NULL END,
    CASE WHEN gs % 29 = 0 THEN '{{ .ExecutiveUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 29 = 0 THEN now() - interval '20 minutes' ELSE NULL END,
    CASE WHEN gs % 13 = 0 OR gs % 19 = 0 THEN now() + interval '72 hours' ELSE NULL END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'workflow-instance-' || (((gs - 1) % {{ .Scale.WorkflowInstanceCount }}) + 1)),
    ARRAY['seeded', 'remediation'],
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    'Security Manager'
FROM generate_series(1, {{ .Scale.RemediationActionCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    alert_id = EXCLUDED.alert_id,
    vulnerability_id = EXCLUDED.vulnerability_id,
    type = EXCLUDED.type,
    status = EXCLUDED.status,
    execution_mode = EXCLUDED.execution_mode,
    dry_run_result = EXCLUDED.dry_run_result,
    execution_result = EXCLUDED.execution_result,
    rollback_data = EXCLUDED.rollback_data,
    approved_by = EXCLUDED.approved_by,
    executed_by = EXCLUDED.executed_by,
    executed_at = EXCLUDED.executed_at,
    completed_at = EXCLUDED.completed_at,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    assessment_id = EXCLUDED.assessment_id,
    ctem_finding_id = EXCLUDED.ctem_finding_id,
    remediation_group_id = EXCLUDED.remediation_group_id,
    severity = EXCLUDED.severity,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    plan = EXCLUDED.plan,
    affected_asset_ids = EXCLUDED.affected_asset_ids,
    affected_asset_count = EXCLUDED.affected_asset_count,
    submitted_by = EXCLUDED.submitted_by,
    submitted_at = EXCLUDED.submitted_at,
    approved_at = EXCLUDED.approved_at,
    rejected_by = EXCLUDED.rejected_by,
    rejected_at = EXCLUDED.rejected_at,
    rejection_reason = EXCLUDED.rejection_reason,
    approval_notes = EXCLUDED.approval_notes,
    requires_approval_from = EXCLUDED.requires_approval_from,
    dry_run_at = EXCLUDED.dry_run_at,
    dry_run_duration_ms = EXCLUDED.dry_run_duration_ms,
    pre_execution_state = EXCLUDED.pre_execution_state,
    execution_started_at = EXCLUDED.execution_started_at,
    execution_completed_at = EXCLUDED.execution_completed_at,
    execution_duration_ms = EXCLUDED.execution_duration_ms,
    verification_result = EXCLUDED.verification_result,
    verified_by = EXCLUDED.verified_by,
    verified_at = EXCLUDED.verified_at,
    rollback_result = EXCLUDED.rollback_result,
    rollback_reason = EXCLUDED.rollback_reason,
    rollback_approved_by = EXCLUDED.rollback_approved_by,
    rolled_back_at = EXCLUDED.rolled_back_at,
    rollback_deadline = EXCLUDED.rollback_deadline,
    workflow_instance_id = EXCLUDED.workflow_instance_id,
    tags = EXCLUDED.tags,
    metadata = EXCLUDED.metadata,
    created_by_name = EXCLUDED.created_by_name,
    deleted_at = NULL;

INSERT INTO remediation_audit_trail (
    id, tenant_id, remediation_id, action, actor_id, actor_name, old_status, new_status, step_number,
    step_action, step_result, details, error_message, duration_ms, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'remediation-audit-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'remediation-action-' || (((gs - 1) % {{ .Scale.RemediationActionCount }}) + 1)),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'submitted'
        WHEN 1 THEN 'approved'
        WHEN 2 THEN 'executed'
        ELSE 'verified'
    END,
    '{{ .SecurityManagerUserID }}'::uuid,
    'Security Manager',
    CASE WHEN gs % 4 = 0 THEN 'pending_approval' ELSE NULL END,
    CASE WHEN gs % 4 = 0 THEN 'approved' WHEN gs % 4 = 2 THEN 'executed' ELSE NULL END,
    ((gs - 1) % 3) + 1,
    CASE ((gs - 1) % 3) + 1
        WHEN 1 THEN 'validate_scope'
        WHEN 2 THEN 'apply_change'
        ELSE 'verify_outcome'
    END,
    CASE WHEN gs % 17 = 0 THEN 'warning' ELSE 'success' END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    CASE WHEN gs % 17 = 0 THEN 'Seeded warning condition.' ELSE NULL END,
    45000 + (gs % 18000),
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, {{ .Scale.RemediationActionCount }} * 2) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO dspm_scans (
    id, tenant_id, status, assets_scanned, pii_assets_found, high_risk_found, findings_count,
    started_at, completed_at, duration_ms, created_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-scan-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE WHEN gs % 9 = 0 THEN 'failed' WHEN gs % 4 = 0 THEN 'running' ELSE 'completed' END,
    {{ .Scale.DSPMAssetCount }},
    {{ .Scale.DSPMAssetCount }} / 3,
    {{ .Scale.DSPMAssetCount }} / 5,
    20 + (gs * 6),
    now() - make_interval(days => gs),
    CASE WHEN gs % 4 = 0 THEN NULL ELSE now() - make_interval(days => gs) + interval '21 minutes' END,
    CASE WHEN gs % 4 = 0 THEN NULL ELSE 1260000 END,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => gs)
FROM generate_series(1, 12) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO dspm_data_assets (
    id, tenant_id, name, type, location, classification, sensitivity_score, owner, data_types,
    risk_score, last_scanned_at, metadata, created_at, updated_at, created_by, updated_by, asset_id,
    scan_id, data_classification, contains_pii, pii_types, pii_column_count, estimated_record_count,
    encrypted_at_rest, encrypted_in_transit, access_control_type, network_exposure, backup_configured,
    audit_logging, last_access_review, risk_factors, posture_score, posture_findings, consumer_count,
    producer_count, database_type, schema_info
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || gs),
    '{{ .MainTenantID }}'::uuid,
    format('Seeded Data Asset %s', lpad(gs::text, 4, '0')),
    CASE (gs - 1) % 5
        WHEN 0 THEN 'postgresql'
        WHEN 1 THEN 's3_bucket'
        WHEN 2 THEN 'data_warehouse'
        WHEN 3 THEN 'lakehouse'
        ELSE 'api_store'
    END,
    format('seeded://data-asset/%s', gs),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END::data_classification,
    CASE WHEN gs % 11 = 0 THEN 94.0 WHEN gs % 5 = 0 THEN 82.0 ELSE 46.0 + (gs % 32) END,
    '{{ .DataStewardUserID }}'::uuid,
    CASE WHEN gs % 3 = 0 THEN ARRAY['pii', 'financial'] ELSE ARRAY['operational'] END,
    CASE WHEN gs % 11 = 0 THEN 92.0 WHEN gs % 5 = 0 THEN 78.0 ELSE 38.0 + (gs % 40) END,
    now() - make_interval(hours => gs % 120),
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    now() - make_interval(days => 25 - (gs % 20)),
    now() - make_interval(hours => gs % 72),
    '{{ .DataStewardUserID }}'::uuid,
    '{{ .DataStewardUserID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-asset-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-scan-' || (((gs - 1) % 12) + 1)),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email', 'phone_number'] ELSE ARRAY[]::text[] END,
    CASE WHEN gs % 3 = 0 THEN 6 + (gs % 5) ELSE 0 END,
    500000 + (gs * 1200),
    gs % 4 <> 0,
    true,
    CASE WHEN gs % 5 = 0 THEN 'abac' WHEN gs % 2 = 0 THEN 'rbac' ELSE 'basic' END,
    CASE WHEN gs % 5 = 0 THEN 'internet_facing' WHEN gs % 2 = 0 THEN 'vpn_accessible' ELSE 'internal_only' END,
    gs % 7 <> 0,
    true,
    now() - make_interval(days => 14 - (gs % 10)),
    jsonb_build_array(jsonb_build_object('factor', 'sensitive_data'), jsonb_build_object('factor', 'broad_access')),
    CASE WHEN gs % 11 = 0 THEN 61.0 WHEN gs % 5 = 0 THEN 74.0 ELSE 84.0 + (gs % 12) END,
    jsonb_build_array(jsonb_build_object('finding', 'encryption_gap'), jsonb_build_object('finding', 'stale_access')),
    3 + (gs % 9),
    1 + (gs % 5),
    CASE WHEN gs % 2 = 0 THEN 'postgresql' ELSE 'object_storage' END,
    jsonb_build_object('schemas', jsonb_build_array('public', 'analytics'))
FROM generate_series(1, {{ .Scale.DSPMAssetCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    location = EXCLUDED.location,
    classification = EXCLUDED.classification,
    sensitivity_score = EXCLUDED.sensitivity_score,
    owner = EXCLUDED.owner,
    data_types = EXCLUDED.data_types,
    risk_score = EXCLUDED.risk_score,
    last_scanned_at = EXCLUDED.last_scanned_at,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at,
    updated_by = EXCLUDED.updated_by,
    asset_id = EXCLUDED.asset_id,
    scan_id = EXCLUDED.scan_id,
    data_classification = EXCLUDED.data_classification,
    contains_pii = EXCLUDED.contains_pii,
    pii_types = EXCLUDED.pii_types,
    pii_column_count = EXCLUDED.pii_column_count,
    estimated_record_count = EXCLUDED.estimated_record_count,
    encrypted_at_rest = EXCLUDED.encrypted_at_rest,
    encrypted_in_transit = EXCLUDED.encrypted_in_transit,
    access_control_type = EXCLUDED.access_control_type,
    network_exposure = EXCLUDED.network_exposure,
    backup_configured = EXCLUDED.backup_configured,
    audit_logging = EXCLUDED.audit_logging,
    last_access_review = EXCLUDED.last_access_review,
    risk_factors = EXCLUDED.risk_factors,
    posture_score = EXCLUDED.posture_score,
    posture_findings = EXCLUDED.posture_findings,
    consumer_count = EXCLUDED.consumer_count,
    producer_count = EXCLUDED.producer_count,
    database_type = EXCLUDED.database_type,
    schema_info = EXCLUDED.schema_info;

INSERT INTO dspm_access_mappings (
    id, tenant_id, identity_type, identity_id, identity_name, identity_source, data_asset_id,
    data_asset_name, data_classification, permission_type, permission_source, permission_path,
    is_wildcard, last_used_at, usage_count_30d, usage_count_90d, is_stale, sensitivity_weight,
    access_risk_score, status, expires_at, discovered_at, last_verified_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-access-mapping-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'role'
        WHEN 3 THEN 'group'
        WHEN 4 THEN 'api_key'
        ELSE 'application'
    END,
    format('seeded-identity-%s', ((gs - 1) % {{ .Scale.DSPMIdentityProfileCount }}) + 1),
    format('Seeded Identity %s', ((gs - 1) % {{ .Scale.DSPMIdentityProfileCount }}) + 1),
    CASE WHEN gs % 2 = 0 THEN 'iam' ELSE 'database' END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    format('Seeded Data Asset %s', lpad((((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)::text, 4, '0')),
    CASE (((gs - 1) % {{ .Scale.DSPMAssetCount }}) % 4)
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    CASE (gs - 1) % 8
        WHEN 0 THEN 'read'
        WHEN 1 THEN 'write'
        WHEN 2 THEN 'admin'
        WHEN 3 THEN 'delete'
        WHEN 4 THEN 'create'
        WHEN 5 THEN 'alter'
        WHEN 6 THEN 'execute'
        ELSE 'full_control'
    END,
    CASE WHEN gs % 3 = 0 THEN 'role_binding' WHEN gs % 2 = 0 THEN 'direct_grant' ELSE 'inherited_group' END,
    ARRAY['db', 'schema', format('table_%s', gs % 50)],
    gs % 17 = 0,
    now() - make_interval(days => gs % 45),
    4 + (gs % 30),
    8 + (gs % 60),
    gs % 19 = 0,
    CASE WHEN gs % 11 = 0 THEN 2.5 ELSE 1.0 + ((gs % 4)::float / 10.0) END,
    CASE WHEN gs % 17 = 0 THEN 92.0 WHEN gs % 5 = 0 THEN 74.0 ELSE 28.0 + (gs % 50) END,
    CASE WHEN gs % 19 = 0 THEN 'pending_review' WHEN gs % 23 = 0 THEN 'expired' ELSE 'active' END,
    CASE WHEN gs % 23 = 0 THEN now() + interval '10 days' ELSE NULL END,
    now() - make_interval(days => 30 - (gs % 20)),
    now() - make_interval(days => gs % 14),
    now() - make_interval(days => 30 - (gs % 20)),
    now() - make_interval(hours => gs % 96)
FROM generate_series(1, {{ .Scale.DSPMAccessMappingCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    identity_type = EXCLUDED.identity_type,
    identity_id = EXCLUDED.identity_id,
    identity_name = EXCLUDED.identity_name,
    identity_source = EXCLUDED.identity_source,
    data_asset_id = EXCLUDED.data_asset_id,
    data_asset_name = EXCLUDED.data_asset_name,
    data_classification = EXCLUDED.data_classification,
    permission_type = EXCLUDED.permission_type,
    permission_source = EXCLUDED.permission_source,
    permission_path = EXCLUDED.permission_path,
    is_wildcard = EXCLUDED.is_wildcard,
    last_used_at = EXCLUDED.last_used_at,
    usage_count_30d = EXCLUDED.usage_count_30d,
    usage_count_90d = EXCLUDED.usage_count_90d,
    is_stale = EXCLUDED.is_stale,
    sensitivity_weight = EXCLUDED.sensitivity_weight,
    access_risk_score = EXCLUDED.access_risk_score,
    status = EXCLUDED.status,
    expires_at = EXCLUDED.expires_at,
    last_verified_at = EXCLUDED.last_verified_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_identity_profiles (
    id, tenant_id, identity_type, identity_id, identity_name, identity_email, identity_source,
    total_assets_accessible, sensitive_assets_count, permission_count, overprivileged_count,
    stale_permission_count, blast_radius_score, blast_radius_level, access_risk_score,
    access_risk_level, risk_factors, last_activity_at, avg_daily_access_count,
    access_pattern_summary, recommendations, status, last_review_at, next_review_due,
    created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-identity-profile-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'role'
        WHEN 3 THEN 'group'
        WHEN 4 THEN 'api_key'
        ELSE 'application'
    END,
    format('seeded-identity-%s', gs),
    format('Seeded Identity %s', gs),
    format('identity-%s@seeded.local', gs),
    CASE WHEN gs % 2 = 0 THEN 'iam' ELSE 'database' END,
    4 + (gs % 18),
    1 + (gs % 8),
    8 + (gs % 24),
    CASE WHEN gs % 9 = 0 THEN 3 ELSE 0 END,
    CASE WHEN gs % 7 = 0 THEN 2 ELSE 0 END,
    CASE WHEN gs % 11 = 0 THEN 94.0 WHEN gs % 5 = 0 THEN 72.0 ELSE 24.0 + (gs % 42) END,
    CASE
        WHEN gs % 11 = 0 THEN 'critical'
        WHEN gs % 5 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    CASE WHEN gs % 11 = 0 THEN 92.0 WHEN gs % 5 = 0 THEN 76.0 ELSE 30.0 + (gs % 44) END,
    CASE
        WHEN gs % 11 = 0 THEN 'critical'
        WHEN gs % 5 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    jsonb_build_array(jsonb_build_object('factor', 'sensitive_asset_access'), jsonb_build_object('factor', 'stale_permission')),
    now() - make_interval(hours => gs % 96),
    2.5 + ((gs % 15)::float / 2.0),
    jsonb_build_object('peak_hours', jsonb_build_array(9, 10, 11), 'weekend_access', gs % 6 = 0),
    jsonb_build_array('Review broad permissions', 'Rotate stale credentials'),
    CASE WHEN gs % 13 = 0 THEN 'under_review' WHEN gs % 17 = 0 THEN 'remediated' ELSE 'active' END,
    now() - make_interval(days => gs % 30),
    now() + make_interval(days => 30 + (gs % 30)),
    now() - make_interval(days => 40 - (gs % 20)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.DSPMIdentityProfileCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    identity_type = EXCLUDED.identity_type,
    identity_id = EXCLUDED.identity_id,
    identity_name = EXCLUDED.identity_name,
    identity_email = EXCLUDED.identity_email,
    identity_source = EXCLUDED.identity_source,
    total_assets_accessible = EXCLUDED.total_assets_accessible,
    sensitive_assets_count = EXCLUDED.sensitive_assets_count,
    permission_count = EXCLUDED.permission_count,
    overprivileged_count = EXCLUDED.overprivileged_count,
    stale_permission_count = EXCLUDED.stale_permission_count,
    blast_radius_score = EXCLUDED.blast_radius_score,
    blast_radius_level = EXCLUDED.blast_radius_level,
    access_risk_score = EXCLUDED.access_risk_score,
    access_risk_level = EXCLUDED.access_risk_level,
    risk_factors = EXCLUDED.risk_factors,
    last_activity_at = EXCLUDED.last_activity_at,
    avg_daily_access_count = EXCLUDED.avg_daily_access_count,
    access_pattern_summary = EXCLUDED.access_pattern_summary,
    recommendations = EXCLUDED.recommendations,
    status = EXCLUDED.status,
    last_review_at = EXCLUDED.last_review_at,
    next_review_due = EXCLUDED.next_review_due,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_access_policies (
    id, tenant_id, name, description, policy_type, rule_config, enforcement, severity,
    enabled, created_by, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-access-policy-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE gs
        WHEN 1 THEN 'Idle Access Review'
        WHEN 2 THEN 'Restricted Data Access'
        WHEN 3 THEN 'Segregation of Duties'
        WHEN 4 THEN 'Time Bound Access'
        WHEN 5 THEN 'Blast Radius Limit'
        ELSE 'Periodic Review'
    END,
    'Seeded DSPM access governance policy.',
    CASE gs
        WHEN 1 THEN 'max_idle_days'
        WHEN 2 THEN 'classification_restrict'
        WHEN 3 THEN 'separation_of_duties'
        WHEN 4 THEN 'time_bound_access'
        WHEN 5 THEN 'blast_radius_limit'
        ELSE 'periodic_review'
    END,
    jsonb_build_object('threshold', 30 + (gs * 5), 'seeded', true),
    CASE WHEN gs % 3 = 0 THEN 'auto_remediate' WHEN gs % 2 = 0 THEN 'block' ELSE 'alert' END,
    CASE WHEN gs % 5 = 0 THEN 'critical' WHEN gs % 2 = 0 THEN 'high' ELSE 'medium' END,
    true,
    '{{ .SecurityManagerUserID }}'::uuid,
    now() - make_interval(days => 10 - gs),
    now() - make_interval(hours => gs)
FROM generate_series(1, 6) gs
ON CONFLICT (tenant_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    policy_type = EXCLUDED.policy_type,
    rule_config = EXCLUDED.rule_config,
    enforcement = EXCLUDED.enforcement,
    severity = EXCLUDED.severity,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_access_audit (
    id, tenant_id, identity_type, identity_id, data_asset_id, action, source_ip, query_hash,
    rows_affected, duration_ms, success, access_mapping_id, table_name, database_name,
    event_timestamp, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-access-audit-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE ((gs - 1) % 6)
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'role'
        WHEN 3 THEN 'group'
        WHEN 4 THEN 'api_key'
        ELSE 'application'
    END,
    format('seeded-identity-%s', ((gs - 1) % {{ .Scale.DSPMIdentityProfileCount }}) + 1),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    CASE (gs - 1) % 6
        WHEN 0 THEN 'select'
        WHEN 1 THEN 'insert'
        WHEN 2 THEN 'update'
        WHEN 3 THEN 'export'
        WHEN 4 THEN 'download'
        ELSE 'api_call'
    END,
    format('10.60.%s.%s', gs % 200, (gs * 7) % 250 + 1),
    md5('dspm-audit-' || gs),
    50 + (gs % 5000),
    20 + (gs % 900),
    gs % 29 <> 0,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-access-mapping-' || (((gs - 1) % {{ .Scale.DSPMAccessMappingCount }}) + 1)),
    format('table_%s', gs % 120),
    CASE WHEN gs % 2 = 0 THEN 'seeded_wh' ELSE 'seeded_app' END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => (gs % 60)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => ((gs + 8) % 60))
FROM generate_series(1, {{ .Scale.DSPMAccessAuditCount }}) gs
ON CONFLICT (id, created_at) DO NOTHING;

INSERT INTO dspm_remediations (
    id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name, identity_id, playbook_id,
    title, description, severity, steps, current_step, total_steps, assigned_to, assigned_team,
    sla_due_at, sla_breached, risk_score_before, risk_score_after, risk_reduction, pre_action_state,
    rollback_available, rolled_back, status, cyber_alert_id, created_by, created_at, updated_at,
    completed_at, compliance_tags
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-remediation-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 11
        WHEN 0 THEN 'posture_gap'
        WHEN 1 THEN 'overprivileged_access'
        WHEN 2 THEN 'stale_access'
        WHEN 3 THEN 'classification_drift'
        WHEN 4 THEN 'shadow_copy'
        WHEN 5 THEN 'policy_violation'
        WHEN 6 THEN 'encryption_missing'
        WHEN 7 THEN 'exposure_risk'
        WHEN 8 THEN 'pii_unprotected'
        WHEN 9 THEN 'retention_expired'
        ELSE 'blast_radius_excessive'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-finding-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    format('Seeded Data Asset %s', lpad((((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)::text, 4, '0')),
    format('seeded-identity-%s', ((gs - 1) % {{ .Scale.DSPMIdentityProfileCount }}) + 1),
    format('playbook-%s', ((gs - 1) % 12) + 1),
    format('Seeded DSPM Remediation %s', lpad(gs::text, 4, '0')),
    'Seeded DSPM remediation for posture, access, and compliance operations.',
    CASE WHEN gs % 17 = 0 THEN 'critical' WHEN gs % 5 = 0 THEN 'high' WHEN gs % 2 = 0 THEN 'medium' ELSE 'low' END,
    jsonb_build_array(
        jsonb_build_object('step', 1, 'action', 'review'),
        jsonb_build_object('step', 2, 'action', 'apply'),
        jsonb_build_object('step', 3, 'action', 'verify')
    ),
    CASE WHEN gs % 13 = 0 THEN 3 WHEN gs % 5 = 0 THEN 2 ELSE 1 END,
    3,
    CASE WHEN gs % 2 = 0 THEN '{{ .DataStewardUserID }}'::uuid ELSE '{{ .SecurityManagerUserID }}'::uuid END,
    CASE WHEN gs % 2 = 0 THEN 'Data Governance' ELSE 'Security Operations' END,
    now() + make_interval(days => 7 + (gs % 21)),
    gs % 23 = 0,
    74.0 + (gs % 20),
    CASE WHEN gs % 13 = 0 THEN 24.0 + (gs % 15) ELSE 42.0 + (gs % 18) END,
    18.0 + (gs % 28),
    jsonb_build_object('policy_state', 'before'),
    true,
    gs % 29 = 0,
    CASE
        WHEN gs % 29 = 0 THEN 'rolled_back'
        WHEN gs % 19 = 0 THEN 'completed'
        WHEN gs % 13 = 0 THEN 'in_progress'
        WHEN gs % 11 = 0 THEN 'awaiting_approval'
        ELSE 'open'
    END,
    CASE WHEN gs % 3 = 0 THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || (((gs - 1) % {{ .Scale.AlertCount }}) + 1)) ELSE NULL END,
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 14 - (gs % 10)),
    now() - make_interval(hours => gs % 72),
    CASE WHEN gs % 19 = 0 THEN now() - interval '1 hour' ELSE NULL END,
    jsonb_build_array('gdpr', 'soc2')
FROM generate_series(1, {{ .Scale.DSPMRemediationCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    finding_type = EXCLUDED.finding_type,
    finding_id = EXCLUDED.finding_id,
    data_asset_id = EXCLUDED.data_asset_id,
    data_asset_name = EXCLUDED.data_asset_name,
    identity_id = EXCLUDED.identity_id,
    playbook_id = EXCLUDED.playbook_id,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    steps = EXCLUDED.steps,
    current_step = EXCLUDED.current_step,
    total_steps = EXCLUDED.total_steps,
    assigned_to = EXCLUDED.assigned_to,
    assigned_team = EXCLUDED.assigned_team,
    sla_due_at = EXCLUDED.sla_due_at,
    sla_breached = EXCLUDED.sla_breached,
    risk_score_before = EXCLUDED.risk_score_before,
    risk_score_after = EXCLUDED.risk_score_after,
    risk_reduction = EXCLUDED.risk_reduction,
    pre_action_state = EXCLUDED.pre_action_state,
    rollback_available = EXCLUDED.rollback_available,
    rolled_back = EXCLUDED.rolled_back,
    status = EXCLUDED.status,
    cyber_alert_id = EXCLUDED.cyber_alert_id,
    updated_at = EXCLUDED.updated_at,
    completed_at = EXCLUDED.completed_at,
    compliance_tags = EXCLUDED.compliance_tags;

INSERT INTO dspm_remediation_history (
    id, tenant_id, remediation_id, action, actor_id, actor_type, details, entry_hash, prev_hash, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-remediation-history-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-remediation-' || (((gs - 1) % {{ .Scale.DSPMRemediationCount }}) + 1)),
    CASE (gs - 1) % 4
        WHEN 0 THEN 'created'
        WHEN 1 THEN 'assigned'
        WHEN 2 THEN 'executed'
        ELSE 'verified'
    END,
    CASE WHEN gs % 2 = 0 THEN '{{ .DataStewardUserID }}'::uuid ELSE '{{ .SecurityManagerUserID }}'::uuid END,
    CASE WHEN gs % 5 = 0 THEN 'policy_engine' WHEN gs % 2 = 0 THEN 'user' ELSE 'system' END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    md5('dspm-remediation-history-' || gs),
    CASE WHEN gs = 1 THEN NULL ELSE md5('dspm-remediation-history-' || (gs - 1)) END,
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, {{ .Scale.DSPMRemediationCount }} * 2) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO dspm_data_policies (
    id, tenant_id, name, description, category, rule, enforcement, auto_playbook_id, severity,
    scope_classification, scope_asset_types, enabled, last_evaluated_at, violation_count,
    compliance_frameworks, created_by, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-policy-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE gs
        WHEN 1 THEN 'Encryption At Rest Policy'
        WHEN 2 THEN 'Classification Integrity Policy'
        WHEN 3 THEN 'Retention Window Policy'
        WHEN 4 THEN 'External Exposure Policy'
        WHEN 5 THEN 'PII Protection Policy'
        WHEN 6 THEN 'Access Review Policy'
        WHEN 7 THEN 'Backup Coverage Policy'
        ELSE 'Audit Logging Policy'
    END,
    'Seeded DSPM data policy for posture and compliance enforcement.',
    CASE gs
        WHEN 1 THEN 'encryption'
        WHEN 2 THEN 'classification'
        WHEN 3 THEN 'retention'
        WHEN 4 THEN 'exposure'
        WHEN 5 THEN 'pii_protection'
        WHEN 6 THEN 'access_review'
        WHEN 7 THEN 'backup'
        ELSE 'audit_logging'
    END,
    jsonb_build_object('required', true, 'threshold', 30 + gs),
    CASE WHEN gs % 3 = 0 THEN 'auto_remediate' WHEN gs % 2 = 0 THEN 'block' ELSE 'alert' END,
    format('playbook-%s', gs),
    CASE WHEN gs % 5 = 0 THEN 'critical' WHEN gs % 2 = 0 THEN 'high' ELSE 'medium' END,
    ARRAY['internal', 'confidential', 'restricted'],
    ARRAY['postgresql', 's3_bucket', 'lakehouse'],
    true,
    now() - make_interval(days => gs),
    5 + (gs * 2),
    ARRAY['gdpr', 'soc2', 'iso27001'],
    '{{ .DataStewardUserID }}'::uuid,
    now() - make_interval(days => 12 - gs),
    now() - make_interval(hours => gs)
FROM generate_series(1, 8) gs
ON CONFLICT (tenant_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    rule = EXCLUDED.rule,
    enforcement = EXCLUDED.enforcement,
    auto_playbook_id = EXCLUDED.auto_playbook_id,
    severity = EXCLUDED.severity,
    scope_classification = EXCLUDED.scope_classification,
    scope_asset_types = EXCLUDED.scope_asset_types,
    enabled = EXCLUDED.enabled,
    last_evaluated_at = EXCLUDED.last_evaluated_at,
    violation_count = EXCLUDED.violation_count,
    compliance_frameworks = EXCLUDED.compliance_frameworks,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_risk_exceptions (
    id, tenant_id, exception_type, remediation_id, data_asset_id, policy_id, justification,
    business_reason, compensating_controls, risk_score, risk_level, requested_by, approved_by,
    approval_status, approved_at, rejection_reason, expires_at, review_interval_days,
    next_review_at, last_reviewed_at, review_count, status, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-risk-exception-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 5
        WHEN 0 THEN 'posture_finding'
        WHEN 1 THEN 'policy_violation'
        WHEN 2 THEN 'overprivileged_access'
        WHEN 3 THEN 'exposure_risk'
        ELSE 'encryption_gap'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-remediation-' || (((gs - 1) % {{ .Scale.DSPMRemediationCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-policy-' || (((gs - 1) % 8) + 1)),
    'Seeded exception justification for temporary business accommodation.',
    'Temporary migration window',
    'Compensating control: monitoring, logging, and weekly review.',
    62.0 + (gs % 28),
    CASE WHEN gs % 11 = 0 THEN 'critical' WHEN gs % 5 = 0 THEN 'high' WHEN gs % 2 = 0 THEN 'medium' ELSE 'low' END,
    '{{ .ExecutiveUserID }}'::uuid,
    CASE WHEN gs % 4 = 0 THEN '{{ .MainAdminUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 4 = 0 THEN 'approved' WHEN gs % 7 = 0 THEN 'rejected' ELSE 'pending' END,
    CASE WHEN gs % 4 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 7 = 0 THEN 'Seeded exception denied.' ELSE NULL END,
    now() + make_interval(days => 30 + (gs % 60)),
    90,
    now() + make_interval(days => 45 + (gs % 30)),
    CASE WHEN gs % 4 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 4 = 0 THEN 1 ELSE 0 END,
    CASE WHEN gs % 13 = 0 THEN 'expired' WHEN gs % 17 = 0 THEN 'revoked' ELSE 'active' END,
    now() - make_interval(days => 6 + gs),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, GREATEST(12, {{ .Scale.DSPMRemediationCount }} / 8)) gs
ON CONFLICT (id) DO UPDATE SET
    exception_type = EXCLUDED.exception_type,
    remediation_id = EXCLUDED.remediation_id,
    data_asset_id = EXCLUDED.data_asset_id,
    policy_id = EXCLUDED.policy_id,
    justification = EXCLUDED.justification,
    business_reason = EXCLUDED.business_reason,
    compensating_controls = EXCLUDED.compensating_controls,
    risk_score = EXCLUDED.risk_score,
    risk_level = EXCLUDED.risk_level,
    approved_by = EXCLUDED.approved_by,
    approval_status = EXCLUDED.approval_status,
    approved_at = EXCLUDED.approved_at,
    rejection_reason = EXCLUDED.rejection_reason,
    expires_at = EXCLUDED.expires_at,
    next_review_at = EXCLUDED.next_review_at,
    last_reviewed_at = EXCLUDED.last_reviewed_at,
    review_count = EXCLUDED.review_count,
    status = EXCLUDED.status,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_data_lineage (
    id, tenant_id, source_asset_id, source_asset_name, source_table, target_asset_id, target_asset_name,
    target_table, edge_type, transformation, pipeline_id, pipeline_name, source_classification,
    target_classification, classification_changed, pii_types_transferred, confidence, evidence,
    status, last_transfer_at, transfer_count_30d, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-lineage-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || gs),
    format('Seeded Data Asset %s', lpad(gs::text, 4, '0')),
    format('source_table_%s', gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || ((gs % {{ .Scale.DSPMAssetCount }}) + 1)),
    format('Seeded Data Asset %s', lpad((((gs % {{ .Scale.DSPMAssetCount }}) + 1))::text, 4, '0')),
    format('target_table_%s', gs),
    CASE (gs - 1) % 8
        WHEN 0 THEN 'etl_pipeline'
        WHEN 1 THEN 'replication'
        WHEN 2 THEN 'api_transfer'
        WHEN 3 THEN 'manual_copy'
        WHEN 4 THEN 'query_derived'
        WHEN 5 THEN 'stream'
        WHEN 6 THEN 'export'
        ELSE 'inferred'
    END,
    'Seeded transfer and transformation path.',
    format('pipeline-%s', ((gs - 1) % {{ .Scale.PipelineCount }}) + 1),
    format('Seeded Pipeline %s', lpad((((gs - 1) % {{ .Scale.PipelineCount }}) + 1)::text, 2, '0')),
    CASE ((gs - 1) % 4)
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    CASE ((gs) % 4)
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    gs % 7 = 0,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email'] ELSE ARRAY[]::text[] END,
    CASE WHEN gs % 8 = 0 THEN 0.76 ELSE 0.94 END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    CASE WHEN gs % 19 = 0 THEN 'deprecated' WHEN gs % 11 = 0 THEN 'broken' ELSE 'active' END,
    now() - make_interval(hours => gs % 120),
    20 + (gs % 60),
    now() - make_interval(days => 18 - (gs % 12)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.DSPMAssetCount }} - 1) gs
ON CONFLICT (id) DO UPDATE SET
    source_asset_name = EXCLUDED.source_asset_name,
    target_asset_name = EXCLUDED.target_asset_name,
    target_table = EXCLUDED.target_table,
    edge_type = EXCLUDED.edge_type,
    transformation = EXCLUDED.transformation,
    pipeline_id = EXCLUDED.pipeline_id,
    pipeline_name = EXCLUDED.pipeline_name,
    source_classification = EXCLUDED.source_classification,
    target_classification = EXCLUDED.target_classification,
    classification_changed = EXCLUDED.classification_changed,
    pii_types_transferred = EXCLUDED.pii_types_transferred,
    confidence = EXCLUDED.confidence,
    evidence = EXCLUDED.evidence,
    status = EXCLUDED.status,
    last_transfer_at = EXCLUDED.last_transfer_at,
    transfer_count_30d = EXCLUDED.transfer_count_30d,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_ai_data_usage (
    id, tenant_id, data_asset_id, data_asset_name, data_classification, contains_pii, pii_types,
    usage_type, model_id, model_name, model_slug, pipeline_id, pipeline_name, ai_risk_score,
    ai_risk_level, risk_factors, consent_verified, data_minimization, anonymization_level,
    retention_compliant, status, first_detected_at, last_detected_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-ai-data-usage-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    format('Seeded Data Asset %s', lpad((((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)::text, 4, '0')),
    CASE ((((gs - 1) % {{ .Scale.DSPMAssetCount }}) % 4))
        WHEN 0 THEN 'public'
        WHEN 1 THEN 'internal'
        WHEN 2 THEN 'confidential'
        ELSE 'restricted'
    END,
    gs % 3 = 0,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email'] ELSE ARRAY[]::text[] END,
    CASE (gs - 1) % 7
        WHEN 0 THEN 'training_data'
        WHEN 1 THEN 'evaluation_data'
        WHEN 2 THEN 'inference_input'
        WHEN 3 THEN 'rag_knowledge_base'
        WHEN 4 THEN 'prompt_context'
        WHEN 5 THEN 'feature_store'
        ELSE 'embedding_source'
    END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ai-usage-model-' || (((gs - 1) % 10) + 1)),
    format('Seeded AI Model %s', ((gs - 1) % 10) + 1),
    format('seeded-ai-model-%s', ((gs - 1) % 10) + 1),
    format('pipeline-%s', ((gs - 1) % {{ .Scale.PipelineCount }}) + 1),
    format('Seeded Pipeline %s', lpad((((gs - 1) % {{ .Scale.PipelineCount }}) + 1)::text, 2, '0')),
    CASE WHEN gs % 13 = 0 THEN 91.0 WHEN gs % 5 = 0 THEN 73.0 ELSE 34.0 + (gs % 40) END,
    CASE WHEN gs % 13 = 0 THEN 'critical' WHEN gs % 5 = 0 THEN 'high' WHEN gs % 2 = 0 THEN 'medium' ELSE 'low' END,
    jsonb_build_array(jsonb_build_object('factor', 'pii_usage'), jsonb_build_object('factor', 'retention_window')),
    gs % 4 <> 0,
    gs % 3 = 0,
    CASE WHEN gs % 5 = 0 THEN 'pseudonymized' WHEN gs % 3 = 0 THEN 'anonymized' ELSE 'none' END,
    gs % 7 <> 0,
    CASE WHEN gs % 17 = 0 THEN 'under_review' WHEN gs % 23 = 0 THEN 'blocked' ELSE 'active' END,
    now() - make_interval(days => 20 - (gs % 15)),
    now() - make_interval(hours => gs % 120),
    now() - make_interval(days => 20 - (gs % 15)),
    now() - make_interval(hours => gs % 120)
FROM generate_series(1, GREATEST(40, {{ .Scale.DSPMAssetCount }} / 2)) gs
ON CONFLICT (id) DO UPDATE SET
    data_asset_id = EXCLUDED.data_asset_id,
    data_asset_name = EXCLUDED.data_asset_name,
    data_classification = EXCLUDED.data_classification,
    contains_pii = EXCLUDED.contains_pii,
    pii_types = EXCLUDED.pii_types,
    usage_type = EXCLUDED.usage_type,
    model_id = EXCLUDED.model_id,
    model_name = EXCLUDED.model_name,
    model_slug = EXCLUDED.model_slug,
    pipeline_id = EXCLUDED.pipeline_id,
    pipeline_name = EXCLUDED.pipeline_name,
    ai_risk_score = EXCLUDED.ai_risk_score,
    ai_risk_level = EXCLUDED.ai_risk_level,
    risk_factors = EXCLUDED.risk_factors,
    consent_verified = EXCLUDED.consent_verified,
    data_minimization = EXCLUDED.data_minimization,
    anonymization_level = EXCLUDED.anonymization_level,
    retention_compliant = EXCLUDED.retention_compliant,
    status = EXCLUDED.status,
    last_detected_at = EXCLUDED.last_detected_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_classification_history (
    id, tenant_id, data_asset_id, old_classification, new_classification, old_pii_types, new_pii_types,
    change_type, detected_by, confidence, evidence, actor_id, actor_type, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-classification-history-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    CASE WHEN gs % 5 = 0 THEN 'internal' ELSE NULL END,
    CASE
        WHEN gs % 11 = 0 THEN 'restricted'
        WHEN gs % 3 = 0 THEN 'confidential'
        ELSE 'internal'
    END,
    CASE WHEN gs % 5 = 0 THEN ARRAY[]::text[] ELSE ARRAY['email'] END,
    CASE WHEN gs % 3 = 0 THEN ARRAY['email', 'phone_number'] ELSE ARRAY['email'] END,
    CASE
        WHEN gs % 11 = 0 THEN 'escalation'
        WHEN gs % 5 = 0 THEN 'pii_added'
        WHEN gs % 7 = 0 THEN 'reclassification'
        ELSE 'initial'
    END,
    CASE WHEN gs % 2 = 0 THEN 'classifier' ELSE 'manual_review' END,
    CASE WHEN gs % 7 = 0 THEN 0.82 ELSE 0.95 END,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    CASE WHEN gs % 2 = 0 THEN '{{ .DataStewardUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN 'user' ELSE 'system' END,
    now() - make_interval(days => 40 - (gs % 30))
FROM generate_series(1, {{ .Scale.DSPMAssetCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO dspm_compliance_posture (
    id, tenant_id, framework, overall_score, controls_total, controls_compliant, controls_partial,
    controls_non_compliant, controls_not_applicable, control_details, score_7d_ago, score_30d_ago,
    score_90d_ago, trend_direction, estimated_fine_exposure, fine_currency, evaluated_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-compliance-posture-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE gs
        WHEN 1 THEN 'gdpr'
        WHEN 2 THEN 'hipaa'
        WHEN 3 THEN 'soc2'
        WHEN 4 THEN 'pci_dss'
        WHEN 5 THEN 'saudi_pdpl'
        ELSE 'iso27001'
    END,
    62.0 + (gs * 4),
    48,
    28 + (gs * 2),
    8 + gs,
    6 - (gs % 3),
    4,
    jsonb_build_array(jsonb_build_object('control', 'A.8', 'status', 'partial')),
    58.0 + (gs * 4),
    54.0 + (gs * 4),
    50.0 + (gs * 4),
    CASE WHEN gs % 5 = 0 THEN 'stable' ELSE 'improving' END,
    250000.0 + (gs * 100000.0),
    'USD',
    now() - make_interval(days => gs),
    now() - make_interval(days => gs),
    now() - make_interval(hours => gs)
FROM generate_series(1, 6) gs
ON CONFLICT (tenant_id, framework) DO UPDATE SET
    overall_score = EXCLUDED.overall_score,
    controls_total = EXCLUDED.controls_total,
    controls_compliant = EXCLUDED.controls_compliant,
    controls_partial = EXCLUDED.controls_partial,
    controls_non_compliant = EXCLUDED.controls_non_compliant,
    controls_not_applicable = EXCLUDED.controls_not_applicable,
    control_details = EXCLUDED.control_details,
    score_7d_ago = EXCLUDED.score_7d_ago,
    score_30d_ago = EXCLUDED.score_30d_ago,
    score_90d_ago = EXCLUDED.score_90d_ago,
    trend_direction = EXCLUDED.trend_direction,
    estimated_fine_exposure = EXCLUDED.estimated_fine_exposure,
    fine_currency = EXCLUDED.fine_currency,
    evaluated_at = EXCLUDED.evaluated_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO dspm_financial_impact (
    id, tenant_id, data_asset_id, estimated_breach_cost, cost_per_record, record_count, cost_breakdown,
    methodology, methodology_details, applicable_regulations, max_regulatory_fine,
    breach_probability_annual, annual_expected_loss, calculated_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-financial-impact-' || gs),
    '{{ .MainTenantID }}'::uuid,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || gs),
    150000.0 + (gs * 4500.0),
    3.2 + ((gs % 20)::float / 10.0),
    500000 + (gs * 2000),
    jsonb_build_object('notification', 25000, 'forensics', 40000, 'downtime', 85000),
    'ibm_ponemon',
    jsonb_build_object('seeded', true),
    ARRAY['gdpr', 'soc2'],
    500000.0 + (gs * 12000.0),
    0.06 + ((gs % 10)::float / 100.0),
    12000.0 + (gs * 350.0),
    now() - make_interval(days => gs % 20),
    now() - make_interval(days => gs % 20),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.DSPMAssetCount }}) gs
ON CONFLICT (tenant_id, data_asset_id) DO UPDATE SET
    estimated_breach_cost = EXCLUDED.estimated_breach_cost,
    cost_per_record = EXCLUDED.cost_per_record,
    record_count = EXCLUDED.record_count,
    cost_breakdown = EXCLUDED.cost_breakdown,
    methodology = EXCLUDED.methodology,
    methodology_details = EXCLUDED.methodology_details,
    applicable_regulations = EXCLUDED.applicable_regulations,
    max_regulatory_fine = EXCLUDED.max_regulatory_fine,
    breach_probability_annual = EXCLUDED.breach_probability_annual,
    annual_expected_loss = EXCLUDED.annual_expected_loss,
    calculated_at = EXCLUDED.calculated_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO ueba_profiles (
    id, tenant_id, entity_type, entity_id, entity_name, entity_email, baseline, observation_count,
    profile_maturity, first_seen_at, last_seen_at, days_active, risk_score, risk_level, risk_factors,
    risk_last_updated, risk_last_decayed, alert_count_7d, alert_count_30d, last_alert_at, status,
    suppressed_until, suppressed_reason, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ueba-profile-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'application'
        ELSE 'api_key'
    END,
    format('ueba-entity-%s', gs),
    format('Seeded UEBA Entity %s', gs),
    CASE WHEN gs % 2 = 0 THEN format('ueba-%s@seeded.local', gs) ELSE NULL END,
    jsonb_build_object('hourly_pattern', jsonb_build_array(9, 10, 11), 'avg_rows', 320 + (gs % 90)),
    300 + (gs * 14),
    CASE WHEN gs % 13 = 0 THEN 'mature' WHEN gs % 5 = 0 THEN 'baseline' ELSE 'learning' END,
    now() - make_interval(days => 120 - (gs % 90)),
    now() - make_interval(hours => gs % 96),
    40 + (gs % 180),
    CASE WHEN gs % 17 = 0 THEN 94.0 WHEN gs % 7 = 0 THEN 78.0 WHEN gs % 3 = 0 THEN 58.0 ELSE 24.0 + (gs % 26) END,
    CASE
        WHEN gs % 17 = 0 THEN 'critical'
        WHEN gs % 7 = 0 THEN 'high'
        WHEN gs % 3 = 0 THEN 'medium'
        ELSE 'low'
    END,
    jsonb_build_array(jsonb_build_object('factor', 'after_hours_access'), jsonb_build_object('factor', 'unusual_export')),
    now() - make_interval(hours => gs % 48),
    now() - make_interval(days => gs % 14),
    gs % 8,
    gs % 20,
    CASE WHEN gs % 8 > 0 THEN now() - make_interval(hours => gs % 96) ELSE NULL END,
    CASE WHEN gs % 23 = 0 THEN 'suppressed' WHEN gs % 29 = 0 THEN 'whitelisted' ELSE 'active' END,
    CASE WHEN gs % 23 = 0 THEN now() + interval '7 days' ELSE NULL END,
    CASE WHEN gs % 23 = 0 THEN 'Seeded maintenance exception.' ELSE NULL END,
    now() - make_interval(days => 120 - (gs % 90)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.UEBAProfileCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    entity_type = EXCLUDED.entity_type,
    entity_id = EXCLUDED.entity_id,
    entity_name = EXCLUDED.entity_name,
    entity_email = EXCLUDED.entity_email,
    baseline = EXCLUDED.baseline,
    observation_count = EXCLUDED.observation_count,
    profile_maturity = EXCLUDED.profile_maturity,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    days_active = EXCLUDED.days_active,
    risk_score = EXCLUDED.risk_score,
    risk_level = EXCLUDED.risk_level,
    risk_factors = EXCLUDED.risk_factors,
    risk_last_updated = EXCLUDED.risk_last_updated,
    risk_last_decayed = EXCLUDED.risk_last_decayed,
    alert_count_7d = EXCLUDED.alert_count_7d,
    alert_count_30d = EXCLUDED.alert_count_30d,
    last_alert_at = EXCLUDED.last_alert_at,
    status = EXCLUDED.status,
    suppressed_until = EXCLUDED.suppressed_until,
    suppressed_reason = EXCLUDED.suppressed_reason,
    updated_at = EXCLUDED.updated_at;

INSERT INTO ueba_access_events (
    id, tenant_id, entity_type, entity_id, source_type, source_id, action, database_name, schema_name,
    table_name, query_hash, rows_accessed, bytes_accessed, duration_ms, source_ip, user_agent, success,
    error_message, table_sensitivity, contains_pii, anomaly_signals, anomaly_count, event_timestamp, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ueba-access-event-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE ((gs - 1) % 4)
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'application'
        ELSE 'api_key'
    END,
    format('ueba-entity-%s', ((gs - 1) % {{ .Scale.UEBAProfileCount }}) + 1),
    CASE WHEN gs % 2 = 0 THEN 'database' WHEN gs % 3 = 0 THEN 'api' ELSE 'warehouse' END,
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'dspm-data-asset-' || (((gs - 1) % {{ .Scale.DSPMAssetCount }}) + 1)),
    CASE (gs - 1) % 11
        WHEN 0 THEN 'select'
        WHEN 1 THEN 'insert'
        WHEN 2 THEN 'update'
        WHEN 3 THEN 'delete'
        WHEN 4 THEN 'create'
        WHEN 5 THEN 'alter'
        WHEN 6 THEN 'drop'
        WHEN 7 THEN 'login'
        WHEN 8 THEN 'logout'
        WHEN 9 THEN 'export'
        ELSE 'api_call'
    END,
    CASE WHEN gs % 2 = 0 THEN 'seeded_wh' ELSE 'seeded_app' END,
    'public',
    format('table_%s', gs % 150),
    md5('ueba-query-' || gs),
    10 + (gs % 5000),
    1024 + (gs * 128),
    20 + (gs % 900),
    format('10.80.%s.%s', gs % 200, (gs * 9) % 250 + 1),
    format('SeededClient/%s', (gs % 7) + 1),
    gs % 37 <> 0,
    CASE WHEN gs % 37 = 0 THEN 'Seeded access denied.' ELSE NULL END,
    CASE WHEN gs % 9 = 0 THEN 'restricted' WHEN gs % 5 = 0 THEN 'confidential' ELSE 'internal' END,
    gs % 9 = 0,
    CASE
        WHEN gs % 29 = 0 THEN jsonb_build_array('after_hours_access', 'mass_download')
        WHEN gs % 13 = 0 THEN jsonb_build_array('new_geo')
        ELSE '[]'::jsonb
    END,
    CASE WHEN gs % 29 = 0 THEN 2 WHEN gs % 13 = 0 THEN 1 ELSE 0 END,
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => (gs % 60)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320), secs => ((gs + 4) % 60))
FROM generate_series(1, {{ .Scale.UEBAAccessEventCount }}) gs
ON CONFLICT (id, created_at) DO NOTHING;

INSERT INTO ueba_alerts (
    id, tenant_id, cyber_alert_id, entity_type, entity_id, entity_name, alert_type, severity,
    confidence, risk_score_before, risk_score_after, risk_score_delta, title, description,
    triggering_signals, triggering_event_ids, baseline_comparison, correlated_signal_count,
    correlation_window_start, correlation_window_end, mitre_technique_ids, mitre_tactic, status,
    resolved_at, resolved_by, resolution_notes, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ueba-alert-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE WHEN gs <= {{ .Scale.AlertCount }} THEN uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'cyber-alert-' || gs) ELSE NULL END,
    CASE ((gs - 1) % 4)
        WHEN 0 THEN 'user'
        WHEN 1 THEN 'service_account'
        WHEN 2 THEN 'application'
        ELSE 'api_key'
    END,
    format('ueba-entity-%s', ((gs - 1) % {{ .Scale.UEBAProfileCount }}) + 1),
    format('Seeded UEBA Entity %s', ((gs - 1) % {{ .Scale.UEBAProfileCount }}) + 1),
    CASE (gs - 1) % 8
        WHEN 0 THEN 'possible_data_exfiltration'
        WHEN 1 THEN 'possible_credential_compromise'
        WHEN 2 THEN 'possible_insider_threat'
        WHEN 3 THEN 'possible_lateral_movement'
        WHEN 4 THEN 'possible_privilege_abuse'
        WHEN 5 THEN 'unusual_activity'
        WHEN 6 THEN 'data_reconnaissance'
        ELSE 'policy_violation'
    END,
    CASE WHEN gs % 17 = 0 THEN 'critical' WHEN gs % 7 = 0 THEN 'high' WHEN gs % 3 = 0 THEN 'medium' ELSE 'low' END,
    round((0.70 + ((gs % 20)::numeric / 100)), 4),
    22.0 + (gs % 24),
    CASE WHEN gs % 17 = 0 THEN 92.0 WHEN gs % 7 = 0 THEN 78.0 ELSE 46.0 + (gs % 22) END,
    CASE WHEN gs % 17 = 0 THEN 48.0 WHEN gs % 7 = 0 THEN 26.0 ELSE 12.0 + (gs % 14) END,
    format('Seeded UEBA Alert %s', lpad(gs::text, 4, '0')),
    'Seeded UEBA alert for anomalous access and insider-risk investigations.',
    jsonb_build_array(jsonb_build_object('signal', 'after_hours_access'), jsonb_build_object('signal', 'export_spike')),
    ARRAY[
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ueba-access-event-' || (((gs - 1) % {{ .Scale.UEBAAccessEventCount }}) + 1)),
        uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'ueba-access-event-' || (((gs) % {{ .Scale.UEBAAccessEventCount }}) + 1))
    ],
    jsonb_build_object('baseline_rows', 240, 'observed_rows', 4200),
    2 + (gs % 6),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320)),
    date_trunc('month', now()) + make_interval(mins => ((gs - 1) % 40320) + 30),
    ARRAY[format('T1%03s', 20 + (gs % 80))],
    CASE WHEN gs % 2 = 0 THEN 'Credential Access' ELSE 'Collection' END,
    CASE WHEN gs % 19 = 0 THEN 'resolved' WHEN gs % 11 = 0 THEN 'investigating' WHEN gs % 5 = 0 THEN 'acknowledged' ELSE 'new' END,
    CASE WHEN gs % 19 = 0 THEN now() - interval '1 day' ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN '{{ .SecurityManagerUserID }}'::uuid ELSE NULL END,
    CASE WHEN gs % 19 = 0 THEN 'Seeded UEBA containment complete.' ELSE NULL END,
    now() - make_interval(days => 12 - (gs % 8)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.UEBAAlertCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    cyber_alert_id = EXCLUDED.cyber_alert_id,
    entity_type = EXCLUDED.entity_type,
    entity_id = EXCLUDED.entity_id,
    entity_name = EXCLUDED.entity_name,
    alert_type = EXCLUDED.alert_type,
    severity = EXCLUDED.severity,
    confidence = EXCLUDED.confidence,
    risk_score_before = EXCLUDED.risk_score_before,
    risk_score_after = EXCLUDED.risk_score_after,
    risk_score_delta = EXCLUDED.risk_score_delta,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    triggering_signals = EXCLUDED.triggering_signals,
    triggering_event_ids = EXCLUDED.triggering_event_ids,
    baseline_comparison = EXCLUDED.baseline_comparison,
    correlated_signal_count = EXCLUDED.correlated_signal_count,
    correlation_window_start = EXCLUDED.correlation_window_start,
    correlation_window_end = EXCLUDED.correlation_window_end,
    mitre_technique_ids = EXCLUDED.mitre_technique_ids,
    mitre_tactic = EXCLUDED.mitre_tactic,
    status = EXCLUDED.status,
    resolved_at = EXCLUDED.resolved_at,
    resolved_by = EXCLUDED.resolved_by,
    resolution_notes = EXCLUDED.resolution_notes,
    updated_at = EXCLUDED.updated_at;

INSERT INTO vciso_llm_system_prompts (
    id, version, prompt_text, prompt_hash, tool_schemas, description, created_by, active, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-llm-prompt-' || gs),
    CASE gs WHEN 1 THEN 'v1.0' ELSE 'v2.0' END,
    CASE gs
        WHEN 1 THEN 'Seeded vCISO system prompt baseline.'
        ELSE 'Seeded vCISO system prompt with compliance and governance tuning.'
    END,
    md5(CASE gs WHEN 1 THEN 'seeded-vciso-prompt-v1' ELSE 'seeded-vciso-prompt-v2' END),
    jsonb_build_array(jsonb_build_object('name', 'search_alerts'), jsonb_build_object('name', 'summarize_risk')),
    'Seeded system prompt registry row.',
    'system-seeder',
    false,
    now() - make_interval(days => 10 - gs)
FROM generate_series(1, 2) gs
ON CONFLICT (version) DO UPDATE SET
    prompt_text = EXCLUDED.prompt_text,
    prompt_hash = EXCLUDED.prompt_hash,
    tool_schemas = EXCLUDED.tool_schemas,
    description = EXCLUDED.description,
    active = EXCLUDED.active;

INSERT INTO vciso_llm_rate_limits (
    id, tenant_id, max_calls_per_minute, max_calls_per_hour, max_calls_per_day, max_tokens_per_day,
    max_cost_per_day_usd, current_calls_minute, current_calls_hour, current_calls_day,
    current_tokens_day, current_cost_day_usd, minute_reset_at, hour_reset_at, day_reset_at, updated_at
)
VALUES (
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-llm-rate-limit-main'),
    '{{ .MainTenantID }}'::uuid,
    45,
    1200,
    8000,
    2500000,
    250.00,
    0,
    0,
    0,
    0,
    0.00,
    now() + interval '1 minute',
    now() + interval '1 hour',
    now() + interval '1 day',
    now()
)
ON CONFLICT (tenant_id) DO UPDATE SET
    max_calls_per_minute = EXCLUDED.max_calls_per_minute,
    max_calls_per_hour = EXCLUDED.max_calls_per_hour,
    max_calls_per_day = EXCLUDED.max_calls_per_day,
    max_tokens_per_day = EXCLUDED.max_tokens_per_day,
    max_cost_per_day_usd = EXCLUDED.max_cost_per_day_usd,
    minute_reset_at = EXCLUDED.minute_reset_at,
    hour_reset_at = EXCLUDED.hour_reset_at,
    day_reset_at = EXCLUDED.day_reset_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO vciso_conversations (
    id, tenant_id, user_id, title, status, message_count, last_context, last_message_at, created_at, updated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-conversation-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE ((gs - 1) % 7) + 1
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 3 THEN '{{ .DataStewardUserID }}'::uuid
        WHEN 4 THEN '{{ .LegalManagerUserID }}'::uuid
        WHEN 5 THEN '{{ .BoardSecretaryUserID }}'::uuid
        WHEN 6 THEN '{{ .ExecutiveUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    format('Seeded vCISO Conversation %s', lpad(gs::text, 3, '0')),
    CASE WHEN gs % 11 = 0 THEN 'archived' ELSE 'active' END,
    0,
    jsonb_build_object('seed_key', '{{ .SeedKey }}', 'sequence', gs),
    NULL,
    now() - make_interval(days => 10 - (gs % 7)),
    now() - make_interval(hours => gs % 72)
FROM generate_series(1, {{ .Scale.VCISOConversationCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    title = EXCLUDED.title,
    status = EXCLUDED.status,
    last_context = EXCLUDED.last_context,
    updated_at = EXCLUDED.updated_at;

INSERT INTO vciso_messages (
    id, conversation_id, tenant_id, role, content, intent, intent_confidence, match_method,
    matched_pattern, extracted_entities, tool_name, tool_params, tool_result, tool_latency_ms,
    tool_error, response_type, suggested_actions, entity_references, prediction_log_id, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-message-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-conversation-' || (((gs - 1) % {{ .Scale.VCISOConversationCount }}) + 1)),
    '{{ .MainTenantID }}'::uuid,
    CASE WHEN gs % 6 = 0 THEN 'system' WHEN gs % 2 = 0 THEN 'assistant' ELSE 'user' END,
    CASE
        WHEN gs % 6 = 0 THEN 'Seeded system context for vCISO follow-up.'
        WHEN gs % 2 = 0 THEN format('Seeded assistant response %s with risk summary and next steps.', gs)
        ELSE format('Seeded user question %s about risk, compliance, or operations.', gs)
    END,
    CASE
        WHEN gs % 2 = 0 THEN 'risk_summary'
        WHEN gs % 5 = 0 THEN 'compliance_status'
        WHEN gs % 3 = 0 THEN 'alert_triage'
        ELSE 'executive_brief'
    END,
    CASE WHEN gs % 6 = 0 THEN NULL ELSE 0.8200 END,
    CASE WHEN gs % 2 = 0 THEN 'hybrid' ELSE 'intent_classifier' END,
    CASE WHEN gs % 2 = 0 THEN 'seeded_risk_pattern' ELSE 'seeded_question_pattern' END,
    jsonb_build_object('asset_count', {{ .Scale.AssetCount }}, 'alert_count', {{ .Scale.AlertCount }}),
    CASE WHEN gs % 2 = 0 THEN 'summarize_risk' WHEN gs % 5 = 0 THEN 'search_alerts' ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN jsonb_build_object('window', '30d') ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN jsonb_build_object('summary', 'Seeded tool result') ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN 120 + (gs % 300) ELSE NULL END,
    CASE WHEN gs % 37 = 0 THEN 'Seeded tool timeout.' ELSE NULL END,
    CASE WHEN gs % 2 = 0 THEN 'summary' ELSE 'question' END,
    CASE WHEN gs % 2 = 0 THEN jsonb_build_array('Review open remediations', 'Check exposure trend') ELSE '[]'::jsonb END,
    CASE WHEN gs % 3 = 0 THEN jsonb_build_array(jsonb_build_object('type', 'alert', 'id', gs)) ELSE '[]'::jsonb END,
    NULL,
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, {{ .Scale.VCISOMessageCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    conversation_id = EXCLUDED.conversation_id,
    role = EXCLUDED.role,
    content = EXCLUDED.content,
    intent = EXCLUDED.intent,
    intent_confidence = EXCLUDED.intent_confidence,
    match_method = EXCLUDED.match_method,
    matched_pattern = EXCLUDED.matched_pattern,
    extracted_entities = EXCLUDED.extracted_entities,
    tool_name = EXCLUDED.tool_name,
    tool_params = EXCLUDED.tool_params,
    tool_result = EXCLUDED.tool_result,
    tool_latency_ms = EXCLUDED.tool_latency_ms,
    tool_error = EXCLUDED.tool_error,
    response_type = EXCLUDED.response_type,
    suggested_actions = EXCLUDED.suggested_actions,
    entity_references = EXCLUDED.entity_references;

UPDATE vciso_conversations c
SET message_count = msg.message_count,
    last_message_at = msg.last_message_at,
    updated_at = now()
FROM (
    SELECT conversation_id, COUNT(*) AS message_count, MAX(created_at) AS last_message_at
    FROM vciso_messages
    WHERE tenant_id = '{{ .MainTenantID }}'::uuid
    GROUP BY conversation_id
) msg
WHERE c.id = msg.conversation_id
  AND c.tenant_id = '{{ .MainTenantID }}'::uuid;

INSERT INTO vciso_llm_audit_log (
    id, message_id, conversation_id, tenant_id, user_id, provider, model, prompt_tokens,
    completion_tokens, total_tokens, estimated_cost_usd, llm_latency_ms, total_latency_ms,
    system_prompt_hash, system_prompt_version, user_message, context_turns, raw_completion,
    tool_calls_json, tool_call_count, reasoning_trace, grounding_result, pii_detections,
    injection_flags, final_response, prediction_log_id, engine_used, routing_reason, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-llm-audit-' || gs),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-message-' || ((((gs - 1) * 2) % {{ .Scale.VCISOMessageCount }}) + 1)),
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-conversation-' || (((gs - 1) % {{ .Scale.VCISOConversationCount }}) + 1)),
    '{{ .MainTenantID }}'::uuid,
    CASE ((gs - 1) % 7) + 1
        WHEN 1 THEN '{{ .MainAdminUserID }}'::uuid
        WHEN 2 THEN '{{ .SecurityManagerUserID }}'::uuid
        WHEN 3 THEN '{{ .DataStewardUserID }}'::uuid
        WHEN 4 THEN '{{ .LegalManagerUserID }}'::uuid
        WHEN 5 THEN '{{ .BoardSecretaryUserID }}'::uuid
        WHEN 6 THEN '{{ .ExecutiveUserID }}'::uuid
        ELSE '{{ .AuditorUserID }}'::uuid
    END,
    CASE WHEN gs % 5 = 0 THEN 'anthropic' ELSE 'openai' END,
    CASE WHEN gs % 5 = 0 THEN 'claude-seeded' ELSE 'gpt-seeded' END,
    600 + (gs % 1200),
    200 + (gs % 900),
    800 + (gs % 1800),
    round(((800 + (gs % 1800))::numeric / 100000), 6),
    300 + (gs % 1500),
    450 + (gs % 1800),
    md5('seeded-vciso-prompt-v2'),
    'v2.0',
    format('Seeded user message %s', gs),
    2 + (gs % 8),
    format('Seeded raw completion %s', gs),
    CASE WHEN gs % 2 = 0 THEN jsonb_build_array(jsonb_build_object('tool', 'summarize_risk')) ELSE '[]'::jsonb END,
    CASE WHEN gs % 2 = 0 THEN 1 ELSE 0 END,
    jsonb_build_array(jsonb_build_object('step', 'retrieve'), jsonb_build_object('step', 'synthesize')),
    CASE WHEN gs % 37 = 0 THEN 'blocked' WHEN gs % 11 = 0 THEN 'corrected' ELSE 'passed' END,
    CASE WHEN gs % 13 = 0 THEN 1 ELSE 0 END,
    CASE WHEN gs % 29 = 0 THEN 1 ELSE 0 END,
    format('Seeded final response %s', gs),
    NULL,
    CASE WHEN gs % 31 = 0 THEN 'fallback' WHEN gs % 17 = 0 THEN 'rule_based' ELSE 'llm' END,
    CASE WHEN gs % 31 = 0 THEN 'provider_quota' WHEN gs % 17 = 0 THEN 'template_route' ELSE 'default_route' END,
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, {{ .Scale.VCISOLLMAuditCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    message_id = EXCLUDED.message_id,
    conversation_id = EXCLUDED.conversation_id,
    user_id = EXCLUDED.user_id,
    provider = EXCLUDED.provider,
    model = EXCLUDED.model,
    prompt_tokens = EXCLUDED.prompt_tokens,
    completion_tokens = EXCLUDED.completion_tokens,
    total_tokens = EXCLUDED.total_tokens,
    estimated_cost_usd = EXCLUDED.estimated_cost_usd,
    llm_latency_ms = EXCLUDED.llm_latency_ms,
    total_latency_ms = EXCLUDED.total_latency_ms,
    system_prompt_hash = EXCLUDED.system_prompt_hash,
    system_prompt_version = EXCLUDED.system_prompt_version,
    user_message = EXCLUDED.user_message,
    context_turns = EXCLUDED.context_turns,
    raw_completion = EXCLUDED.raw_completion,
    tool_calls_json = EXCLUDED.tool_calls_json,
    tool_call_count = EXCLUDED.tool_call_count,
    reasoning_trace = EXCLUDED.reasoning_trace,
    grounding_result = EXCLUDED.grounding_result,
    pii_detections = EXCLUDED.pii_detections,
    injection_flags = EXCLUDED.injection_flags,
    final_response = EXCLUDED.final_response,
    prediction_log_id = EXCLUDED.prediction_log_id,
    engine_used = EXCLUDED.engine_used,
    routing_reason = EXCLUDED.routing_reason,
    created_at = EXCLUDED.created_at;

INSERT INTO vciso_prediction_models (
    id, model_type, version, model_artifact_path, model_framework, backtest_accuracy,
    backtest_precision, backtest_recall, backtest_f1, backtest_mape, feature_count,
    training_samples, training_duration_seconds, status, active, last_drift_check,
    drift_score, created_at, activated_at, deprecated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-prediction-model-' || gs),
    CASE ((gs - 1) % 6) + 1
        WHEN 1 THEN 'alert_volume_forecast'
        WHEN 2 THEN 'asset_risk_prediction'
        WHEN 3 THEN 'vulnerability_exploit_prediction'
        WHEN 4 THEN 'attack_technique_trend'
        WHEN 5 THEN 'insider_threat_trajectory'
        ELSE 'campaign_detection'
    END,
    CASE WHEN gs <= 6 THEN 'v1.0' ELSE 'v2.0' END,
    format('/models/vciso/%s/%s.bin', ((gs - 1) % 6) + 1, CASE WHEN gs <= 6 THEN 'v1' ELSE 'v2' END),
    CASE WHEN gs % 2 = 0 THEN 'xgboost' ELSE 'prophet' END,
    0.7800 + ((gs % 12)::numeric / 1000),
    0.7600 + ((gs % 10)::numeric / 1000),
    0.7400 + ((gs % 10)::numeric / 1000),
    0.7500 + ((gs % 10)::numeric / 1000),
    8.5000 + ((gs % 20)::numeric / 10),
    14 + (gs % 8),
    18000 + (gs * 2000),
    1200 + (gs * 180),
    CASE WHEN gs <= 6 THEN 'deprecated' ELSE 'validating' END,
    false,
    now() - make_interval(days => gs),
    0.0200 + ((gs % 8)::numeric / 1000),
    now() - make_interval(days => 25 - gs),
    NULL,
    CASE WHEN gs <= 6 THEN now() - make_interval(days => 3) ELSE NULL END
FROM generate_series(1, 12) gs
ON CONFLICT (model_type, version) DO UPDATE SET
    model_artifact_path = EXCLUDED.model_artifact_path,
    model_framework = EXCLUDED.model_framework,
    backtest_accuracy = EXCLUDED.backtest_accuracy,
    backtest_precision = EXCLUDED.backtest_precision,
    backtest_recall = EXCLUDED.backtest_recall,
    backtest_f1 = EXCLUDED.backtest_f1,
    backtest_mape = EXCLUDED.backtest_mape,
    feature_count = EXCLUDED.feature_count,
    training_samples = EXCLUDED.training_samples,
    training_duration_seconds = EXCLUDED.training_duration_seconds,
    status = EXCLUDED.status,
    active = EXCLUDED.active,
    last_drift_check = EXCLUDED.last_drift_check,
    drift_score = EXCLUDED.drift_score,
    deprecated_at = EXCLUDED.deprecated_at;

INSERT INTO vciso_feature_snapshots (
    id, tenant_id, feature_set, entity_type, entity_id, vector_json, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-feature-snapshot-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'risk_forecast'
        WHEN 1 THEN 'alert_volume'
        WHEN 2 THEN 'insider_trajectory'
        ELSE 'campaign_trend'
    END,
    CASE (gs - 1) % 3
        WHEN 0 THEN 'asset'
        WHEN 1 THEN 'user'
        ELSE 'threat'
    END,
    format('feature-entity-%s', ((gs - 1) % GREATEST({{ .Scale.AssetCount }}, 1)) + 1),
    jsonb_build_object(
        'feature_1', round((10 + (gs % 50))::numeric, 2),
        'feature_2', round((20 + (gs % 40))::numeric, 2),
        'feature_3', round((30 + (gs % 30))::numeric, 2)
    ),
    now() - make_interval(hours => gs % 240)
FROM generate_series(1, {{ .Scale.VCISOFeatureSnapshotCount }}) gs
ON CONFLICT (id) DO NOTHING;

INSERT INTO vciso_predictions (
    id, tenant_id, prediction_type, model_version, prediction_json, confidence_score,
    confidence_interval, top_features, explanation_text, target_entity_type, target_entity_id,
    forecast_start, forecast_end, outcome_observed, outcome_value, accuracy_score, prediction_log_id,
    created_at, evaluated_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-prediction-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 6
        WHEN 0 THEN 'alert_volume_forecast'
        WHEN 1 THEN 'asset_risk_prediction'
        WHEN 2 THEN 'vulnerability_exploit_prediction'
        WHEN 3 THEN 'attack_technique_trend'
        WHEN 4 THEN 'insider_threat_trajectory'
        ELSE 'campaign_detection'
    END,
    'v2.0',
    jsonb_build_object('forecast_value', 20 + (gs % 120), 'trend', CASE WHEN gs % 2 = 0 THEN 'up' ELSE 'stable' END),
    round((0.72 + ((gs % 20)::numeric / 100)), 4),
    jsonb_build_object('lower', 12 + (gs % 40), 'upper', 30 + (gs % 60)),
    jsonb_build_array(
        jsonb_build_object('name', 'open_alerts', 'importance', 0.38),
        jsonb_build_object('name', 'internet_assets', 'importance', 0.24)
    ),
    'Seeded predictive explanation for forecasting and executive planning.',
    CASE WHEN gs % 2 = 0 THEN 'asset' WHEN gs % 3 = 0 THEN 'user' ELSE 'threat' END,
    format('prediction-entity-%s', ((gs - 1) % GREATEST({{ .Scale.AssetCount }}, 1)) + 1),
    now(),
    now() + make_interval(days => 7 + (gs % 14)),
    gs % 5 = 0,
    CASE WHEN gs % 5 = 0 THEN jsonb_build_object('actual', 18 + (gs % 25)) ELSE NULL END,
    CASE WHEN gs % 5 = 0 THEN round((0.78 + ((gs % 10)::numeric / 100)), 4) ELSE NULL END,
    NULL,
    now() - make_interval(hours => gs % 240),
    CASE WHEN gs % 5 = 0 THEN now() - make_interval(hours => gs % 48) ELSE NULL END
FROM generate_series(1, {{ .Scale.VCISOPredictionCount }}) gs
ON CONFLICT (id) DO UPDATE SET
    prediction_type = EXCLUDED.prediction_type,
    model_version = EXCLUDED.model_version,
    prediction_json = EXCLUDED.prediction_json,
    confidence_score = EXCLUDED.confidence_score,
    confidence_interval = EXCLUDED.confidence_interval,
    top_features = EXCLUDED.top_features,
    explanation_text = EXCLUDED.explanation_text,
    target_entity_type = EXCLUDED.target_entity_type,
    target_entity_id = EXCLUDED.target_entity_id,
    forecast_start = EXCLUDED.forecast_start,
    forecast_end = EXCLUDED.forecast_end,
    outcome_observed = EXCLUDED.outcome_observed,
    outcome_value = EXCLUDED.outcome_value,
    accuracy_score = EXCLUDED.accuracy_score,
    evaluated_at = EXCLUDED.evaluated_at;

INSERT INTO vciso_briefings (
    id, tenant_id, type, period_start, period_end, content, risk_score_at_time, generated_by, created_at
)
SELECT
    uuid_generate_v5('{{ .SeedNamespace }}'::uuid, 'vciso-briefing-' || gs),
    '{{ .MainTenantID }}'::uuid,
    CASE (gs - 1) % 4
        WHEN 0 THEN 'executive'
        WHEN 1 THEN 'technical'
        WHEN 2 THEN 'compliance'
        ELSE 'custom'
    END,
    (current_date - make_interval(days => (gs * 30)))::date,
    (current_date - make_interval(days => ((gs - 1) * 30 + 1)))::date,
    jsonb_build_object('summary', format('Seeded vCISO briefing %s', gs), 'highlights', jsonb_build_array('risk trend', 'top remediations')),
    58.0 + (gs % 24),
    CASE WHEN gs % 2 = 0 THEN '{{ .ExecutiveUserID }}'::uuid ELSE '{{ .SecurityManagerUserID }}'::uuid END,
    now() - make_interval(days => gs * 7)
FROM generate_series(1, 12) gs
ON CONFLICT (id) DO NOTHING;
