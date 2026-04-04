// API endpoint paths
export const API_ENDPOINTS = {
  // Auth
  AUTH_LOGIN: '/api/v1/auth/login',
  AUTH_LOGOUT: '/api/v1/auth/logout',
  AUTH_REFRESH: '/api/v1/auth/refresh',
  AUTH_REGISTER: '/api/v1/auth/register',
  AUTH_FORGOT_PASSWORD: '/api/v1/auth/forgot-password',
  AUTH_RESET_PASSWORD: '/api/v1/auth/reset-password',
  AUTH_VERIFY_MFA: '/api/v1/auth/verify-mfa',
  AUTH_CHECK_EMAIL: '/api/v1/auth/check-email',

  // User
  USERS_ME: '/api/v1/users/me',
  USERS_ME_MFA_ENABLE: '/api/v1/users/me/mfa/enable',
  USERS_ME_MFA_VERIFY_SETUP: '/api/v1/users/me/mfa/verify-setup',
  USERS_ME_MFA_DISABLE: '/api/v1/users/me/mfa/disable',
  USERS_ME_PASSWORD: '/api/v1/users/me/password',
  USER_DETAIL: (id: string) => `/api/v1/users/${id}`,
  ROLES: '/api/v1/roles',

  // BFF routes (Next.js API routes)
  BFF_SESSION: '/api/auth/session',
  BFF_REFRESH: '/api/auth/refresh',

  // Onboarding
  ONBOARDING_REGISTER: '/api/v1/onboarding/register',
  ONBOARDING_VERIFY_EMAIL: '/api/v1/onboarding/verify-email',
  ONBOARDING_RESEND_OTP: '/api/v1/onboarding/resend-otp',
  ONBOARDING_WIZARD: '/api/v1/onboarding/wizard',
  ONBOARDING_ORGANIZATION: '/api/v1/onboarding/wizard/organization',
  ONBOARDING_BRANDING: '/api/v1/onboarding/wizard/branding',
  ONBOARDING_TEAM: '/api/v1/onboarding/wizard/team',
  ONBOARDING_SUITES: '/api/v1/onboarding/wizard/suites',
  ONBOARDING_COMPLETE: '/api/v1/onboarding/wizard/complete',
  ONBOARDING_STATUS: '/api/v1/onboarding/status',
  ONBOARDING_INVITATIONS_VALIDATE: '/api/v1/invitations/validate',
  ONBOARDING_INVITATIONS_ACCEPT: '/api/v1/invitations/accept',

  // Invitations
  INVITATIONS: '/api/v1/invitations',

  // Cyber — Dashboard
  CYBER_DASHBOARD: '/api/v1/cyber/dashboard',
  CYBER_DASHBOARD_KPIS: '/api/v1/cyber/dashboard/kpis',
  CYBER_DASHBOARD_METRICS: '/api/v1/cyber/dashboard/metrics',
  CYBER_DASHBOARD_ALERTS_TIMELINE: '/api/v1/cyber/dashboard/alerts-timeline',
  CYBER_DASHBOARD_SEVERITY_DISTRIBUTION: '/api/v1/cyber/dashboard/severity-distribution',
  CYBER_DASHBOARD_MITRE_HEATMAP: '/api/v1/cyber/dashboard/mitre-heatmap',
  CYBER_DASHBOARD_TOP_ATTACKED_ASSETS: '/api/v1/cyber/dashboard/top-attacked-assets',
  CYBER_DASHBOARD_ANALYST_WORKLOAD: '/api/v1/cyber/dashboard/analyst-workload',

  // Cyber — Assets
  CYBER_ASSETS: '/api/v1/cyber/assets',
  CYBER_ASSETS_STATS: '/api/v1/cyber/assets/stats',
  CYBER_ASSETS_SCAN: '/api/v1/cyber/assets/scan',
  CYBER_ASSETS_SCANS: '/api/v1/cyber/assets/scans',
  CYBER_ASSETS_BULK: '/api/v1/cyber/assets/bulk',
  CYBER_ASSETS_BULK_TAGS: '/api/v1/cyber/assets/bulk/tags',

  // Cyber — Alerts
  CYBER_ALERTS_COUNT: '/api/v1/cyber/alerts/count',
  CYBER_ALERTS_STATS: '/api/v1/cyber/alerts/stats',
  CYBER_ALERTS: '/api/v1/cyber/alerts',
  CYBER_ALERT_DETAIL: (id: string) => `/api/v1/cyber/alerts/${id}`,
  CYBER_ALERT_COMMENTS: (id: string) => `/api/v1/cyber/alerts/${id}/comments`,
  CYBER_ALERT_TIMELINE: (id: string) => `/api/v1/cyber/alerts/${id}/timeline`,
  CYBER_ALERT_RELATED: (id: string) => `/api/v1/cyber/alerts/${id}/related`,
  CYBER_ALERT_STATUS: (id: string) => `/api/v1/cyber/alerts/${id}/status`,
  CYBER_ALERT_ASSIGN: (id: string) => `/api/v1/cyber/alerts/${id}/assign`,
  CYBER_ALERT_ESCALATE: (id: string) => `/api/v1/cyber/alerts/${id}/escalate`,
  CYBER_ALERT_FALSE_POSITIVE: (id: string) => `/api/v1/cyber/alerts/${id}/false-positive`,
  CYBER_ALERT_MERGE: (id: string) => `/api/v1/cyber/alerts/${id}/merge`,
  CYBER_ALERT_BULK_STATUS: '/api/v1/cyber/alerts/bulk/status',
  CYBER_ALERT_BULK_ASSIGN: '/api/v1/cyber/alerts/bulk/assign',
  CYBER_ALERT_BULK_FALSE_POSITIVE: '/api/v1/cyber/alerts/bulk/false-positive',

  // Cyber — Vulnerabilities
  CYBER_VULNERABILITIES_AGING: '/api/v1/cyber/vulnerabilities/aging',

  // Cyber — Threats
  CYBER_THREATS: '/api/v1/cyber/threats',
  CYBER_THREAT_STATS: '/api/v1/cyber/threats/stats',
  CYBER_THREAT_STATS_TREND: '/api/v1/cyber/threats/stats/trend',
  CYBER_THREAT_DETAIL: (id: string) => `/api/v1/cyber/threats/${id}`,
  CYBER_THREAT_STATUS: (id: string) => `/api/v1/cyber/threats/${id}/status`,
  CYBER_THREAT_INDICATORS: (id: string) => `/api/v1/cyber/threats/${id}/indicators`,
  CYBER_THREAT_ALERTS: (id: string) => `/api/v1/cyber/threats/${id}/alerts`,
  CYBER_THREAT_TIMELINE: (id: string) => `/api/v1/cyber/threats/${id}/timeline`,
  CYBER_INDICATORS: '/api/v1/cyber/indicators',
  CYBER_INDICATORS_STATS: '/api/v1/cyber/indicators/stats',
  CYBER_INDICATORS_BULK: '/api/v1/cyber/indicators/bulk',
  CYBER_INDICATORS_BATCH: '/api/v1/cyber/indicators/batch',
  CYBER_INDICATOR_DETAIL: (id: string) => `/api/v1/cyber/indicators/${id}`,
  CYBER_INDICATOR_ENRICHMENT: (id: string) => `/api/v1/cyber/indicators/${id}/enrichment`,
  CYBER_INDICATOR_MATCHES: (id: string) => `/api/v1/cyber/indicators/${id}/matches`,
  CYBER_INDICATOR_STATUS: (id: string) => `/api/v1/cyber/indicators/${id}/status`,
  CYBER_INDICATORS_CHECK: '/api/v1/cyber/indicators/check',
  CYBER_THREAT_FEEDS: '/api/v1/cyber/threat-feeds',
  CYBER_THREAT_FEED_DETAIL: (id: string) => `/api/v1/cyber/threat-feeds/${id}`,
  CYBER_THREAT_FEED_SYNC: (id: string) => `/api/v1/cyber/threat-feeds/${id}/sync`,
  CYBER_THREAT_FEED_HISTORY: (id: string) => `/api/v1/cyber/threat-feeds/${id}/history`,

  // Cyber — Detection Rules
  CYBER_RULES: '/api/v1/cyber/rules',
  CYBER_RULE_STATS: '/api/v1/cyber/rules/stats',
  CYBER_RULE_TEMPLATES: '/api/v1/cyber/rules/templates',
  CYBER_RULE_DETAIL: (id: string) => `/api/v1/cyber/rules/${id}`,
  CYBER_RULE_TOGGLE: (id: string) => `/api/v1/cyber/rules/${id}/toggle`,
  CYBER_RULE_TEST: (id: string) => `/api/v1/cyber/rules/${id}/test`,
  CYBER_RULE_PERFORMANCE: (id: string) => `/api/v1/cyber/rules/${id}/performance`,
  CYBER_RULE_FEEDBACK: (ruleId: string) => `/api/v1/cyber/rules/${ruleId}/feedback`,

  // Cyber — CTEM
  CYBER_CTEM_ASSESSMENTS: '/api/v1/cyber/ctem/assessments',
  CYBER_CTEM_ASSESSMENT_DETAIL: (id: string) => `/api/v1/cyber/ctem/assessments/${id}`,
  CYBER_CTEM_ASSESSMENT_START: (id: string) => `/api/v1/cyber/ctem/assessments/${id}/start`,
  CYBER_CTEM_ASSESSMENT_CANCEL: (id: string) => `/api/v1/cyber/ctem/assessments/${id}/cancel`,
  CYBER_CTEM_ASSESSMENT_FINDINGS: (id: string) => `/api/v1/cyber/ctem/assessments/${id}/findings`,
  CYBER_CTEM_ASSESSMENT_REMEDIATION_GROUPS: (id: string) => `/api/v1/cyber/ctem/assessments/${id}/remediation-groups`,
  CYBER_CTEM_FINDING_STATUS: (findingId: string) => `/api/v1/cyber/ctem/findings/${findingId}/status`,
  CYBER_CTEM_REMEDIATION_GROUP: (groupId: string) => `/api/v1/cyber/ctem/remediation-groups/${groupId}`,
  CYBER_CTEM_REMEDIATION_GROUP_STATUS: (groupId: string) => `/api/v1/cyber/ctem/remediation-groups/${groupId}/status`,
  CYBER_CTEM_REMEDIATION_GROUP_EXECUTE: (groupId: string) => `/api/v1/cyber/ctem/remediation-groups/${groupId}/execute`,
  CYBER_CTEM_ASSESSMENT_REPORT_EXPORT: (id: string) => `/api/v1/cyber/ctem/assessments/${id}/report/export`,
  CYBER_CTEM_ASSESSMENT_COMPARE: (id: string, otherId: string) => `/api/v1/cyber/ctem/assessments/${id}/compare/${otherId}`,
  CYBER_CTEM_EXPOSURE_SCORE: '/api/v1/cyber/ctem/exposure-score',
  CYBER_CTEM_EXPOSURE_HISTORY: '/api/v1/cyber/ctem/exposure-score/history',
  CYBER_CTEM_DASHBOARD: '/api/v1/cyber/ctem/dashboard',

  // Cyber — Security Events
  CYBER_EVENTS: '/api/v1/cyber/events',
  CYBER_EVENTS_EXPORT: '/api/v1/cyber/events/export',
  CYBER_EVENT_DETAIL: (id: string) => `/api/v1/cyber/events/${id}`,
  CYBER_EVENT_STATS: '/api/v1/cyber/events/stats',

  // Cyber — Analytics
  CYBER_ANALYTICS_THREAT_FORECAST: '/api/v1/cyber/analytics/threat-forecast',
  CYBER_ANALYTICS_ALERT_FORECAST: '/api/v1/cyber/analytics/alert-forecast',
  CYBER_ANALYTICS_TECHNIQUE_TRENDS: '/api/v1/cyber/analytics/technique-trends',
  CYBER_ANALYTICS_CAMPAIGNS: '/api/v1/cyber/analytics/campaigns',
  CYBER_ANALYTICS_LANDSCAPE: '/api/v1/cyber/analytics/landscape',

  // Cyber — Remediation
  CYBER_REMEDIATION: '/api/v1/cyber/remediation',
  CYBER_REMEDIATION_STATS: '/api/v1/cyber/remediation/stats',

  // Cyber — DSPM
  CYBER_DSPM_DASHBOARD: '/api/v1/cyber/dspm/dashboard',
  CYBER_DSPM_DATA_ASSETS: '/api/v1/cyber/dspm/data-assets',
  CYBER_DSPM_SCAN: '/api/v1/cyber/dspm/scan',
  CYBER_DSPM_SHADOW_COPIES: '/api/v1/cyber/dspm/shadow-copies',

  // Cyber — DSPM Access Intelligence
  CYBER_DSPM_ACCESS_DASHBOARD: '/api/v1/cyber/dspm/access/dashboard',
  CYBER_DSPM_ACCESS_IDENTITIES: '/api/v1/cyber/dspm/access/identities',
  CYBER_DSPM_ACCESS_MAPPINGS: '/api/v1/cyber/dspm/access/mappings',
  CYBER_DSPM_ACCESS_OVERPRIVILEGED: '/api/v1/cyber/dspm/access/mappings/overprivileged',
  CYBER_DSPM_ACCESS_STALE: '/api/v1/cyber/dspm/access/mappings/stale',
  CYBER_DSPM_ACCESS_RISK_RANKING: '/api/v1/cyber/dspm/access/analysis/risk-ranking',
  CYBER_DSPM_ACCESS_BLAST_RANKING: '/api/v1/cyber/dspm/access/analysis/blast-radius-ranking',
  CYBER_DSPM_ACCESS_ESCALATION: '/api/v1/cyber/dspm/access/analysis/escalation-paths',
  CYBER_DSPM_ACCESS_CROSS_ASSET: '/api/v1/cyber/dspm/access/analysis/cross-asset',
  CYBER_DSPM_ACCESS_POLICIES: '/api/v1/cyber/dspm/access/policies',
  CYBER_DSPM_ACCESS_VIOLATIONS: '/api/v1/cyber/dspm/access/policies/violations',
  CYBER_DSPM_ACCESS_COLLECT: '/api/v1/cyber/dspm/access/collect',

  // Cyber — DSPM Remediation Engine
  CYBER_DSPM_REMEDIATIONS: '/api/v1/cyber/dspm/remediations',
  CYBER_DSPM_REMEDIATION_STATS: '/api/v1/cyber/dspm/remediations/stats',
  CYBER_DSPM_REMEDIATION_DASHBOARD: '/api/v1/cyber/dspm/remediations/dashboard',
  CYBER_DSPM_DATA_POLICIES: '/api/v1/cyber/dspm/policies',
  CYBER_DSPM_POLICY_VIOLATIONS: '/api/v1/cyber/dspm/policies/violations',
  CYBER_DSPM_EXCEPTIONS: '/api/v1/cyber/dspm/exceptions',

  // Cyber — DSPM Advanced Intelligence
  CYBER_DSPM_CLASSIFICATION_ENHANCED: '/api/v1/cyber/dspm/classification/enhanced',
  CYBER_DSPM_CLASSIFICATION_CUSTOM_RULES: '/api/v1/cyber/dspm/classification/custom-rules',
  CYBER_DSPM_CLASSIFICATION_HISTORY: (assetId: string) => `/api/v1/cyber/dspm/classification/history/${assetId}`,
  CYBER_DSPM_LINEAGE_GRAPH: '/api/v1/cyber/dspm/lineage/graph',
  CYBER_DSPM_LINEAGE_UPSTREAM: (assetId: string) => `/api/v1/cyber/dspm/lineage/upstream/${assetId}`,
  CYBER_DSPM_LINEAGE_DOWNSTREAM: (assetId: string) => `/api/v1/cyber/dspm/lineage/downstream/${assetId}`,
  CYBER_DSPM_LINEAGE_IMPACT: (assetId: string) => `/api/v1/cyber/dspm/lineage/impact/${assetId}`,
  CYBER_DSPM_LINEAGE_PII_FLOW: '/api/v1/cyber/dspm/lineage/pii-flow',
  CYBER_DSPM_AI_USAGE: '/api/v1/cyber/dspm/ai/usage',
  CYBER_DSPM_AI_USAGE_ASSET: (assetId: string) => `/api/v1/cyber/dspm/ai/usage/${assetId}`,
  CYBER_DSPM_AI_MODEL_DATA: (modelSlug: string) => `/api/v1/cyber/dspm/ai/models/${modelSlug}/data`,
  CYBER_DSPM_AI_RISK_RANKING: '/api/v1/cyber/dspm/ai/risk-ranking',
  CYBER_DSPM_AI_DASHBOARD: '/api/v1/cyber/dspm/ai/dashboard',
  CYBER_DSPM_FINANCIAL_PORTFOLIO: '/api/v1/cyber/dspm/financial/impact',
  CYBER_DSPM_FINANCIAL_ASSET: (assetId: string) => `/api/v1/cyber/dspm/financial/impact/${assetId}`,
  CYBER_DSPM_FINANCIAL_TOP_RISKS: '/api/v1/cyber/dspm/financial/top-risks',
  CYBER_DSPM_COMPLIANCE_POSTURE: '/api/v1/cyber/dspm/compliance/posture',
  CYBER_DSPM_COMPLIANCE_FRAMEWORK: (framework: string) => `/api/v1/cyber/dspm/compliance/posture/${framework}`,
  CYBER_DSPM_COMPLIANCE_GAPS: '/api/v1/cyber/dspm/compliance/gaps',
  CYBER_DSPM_COMPLIANCE_RESIDENCY: '/api/v1/cyber/dspm/compliance/residency',
  CYBER_DSPM_COMPLIANCE_AUDIT: (framework: string) => `/api/v1/cyber/dspm/compliance/audit-report/${framework}`,
  CYBER_DSPM_PROLIFERATION_OVERVIEW: '/api/v1/cyber/dspm/proliferation/overview',
  CYBER_DSPM_PROLIFERATION_ASSET: (assetId: string) => `/api/v1/cyber/dspm/proliferation/${assetId}`,

  // Root Cause Analysis
  RCA_ANALYZE: '/api/v1/rca/analyze',

  // Cyber — vCISO
  CYBER_VCISO_BRIEFING: '/api/v1/cyber/vciso/briefing',
  CYBER_VCISO_REPORT: '/api/v1/cyber/vciso/report',
  CYBER_VCISO_CHAT: '/api/v1/cyber/vciso/chat',
  CYBER_VCISO_CONVERSATIONS: '/api/v1/cyber/vciso/conversations',
  CYBER_VCISO_SUGGESTIONS: '/api/v1/cyber/vciso/suggestions',
  CYBER_VCISO_LLM_USAGE: '/api/v1/cyber/vciso/llm/usage',
  CYBER_VCISO_LLM_HEALTH: '/api/v1/cyber/vciso/llm/health',
  CYBER_VCISO_LLM_CONFIG: '/api/v1/cyber/vciso/llm/config',
  CYBER_VCISO_LLM_PROMPTS: '/api/v1/cyber/vciso/llm/prompts',
  CYBER_VCISO_LLM_AUDIT: '/api/v1/cyber/vciso/llm/audit',

  // Cyber — vCISO Governance
  CYBER_VCISO_RISKS: '/api/v1/cyber/vciso/risks',
  CYBER_VCISO_RISKS_STATS: '/api/v1/cyber/vciso/risks/stats',
  CYBER_VCISO_POLICIES: '/api/v1/cyber/vciso/policies',
  CYBER_VCISO_POLICY_EXCEPTIONS: '/api/v1/cyber/vciso/policy-exceptions',
  CYBER_VCISO_POLICY_GENERATE: '/api/v1/cyber/vciso/policies/generate',
  CYBER_VCISO_VENDORS: '/api/v1/cyber/vciso/vendors',
  CYBER_VCISO_QUESTIONNAIRES: '/api/v1/cyber/vciso/questionnaires',
  CYBER_VCISO_EVIDENCE: '/api/v1/cyber/vciso/evidence',
  CYBER_VCISO_EVIDENCE_STATS: '/api/v1/cyber/vciso/evidence/stats',
  CYBER_VCISO_MATURITY: '/api/v1/cyber/vciso/maturity',
  CYBER_VCISO_BENCHMARKS: '/api/v1/cyber/vciso/benchmarks',
  CYBER_VCISO_BUDGET: '/api/v1/cyber/vciso/budget',
  CYBER_VCISO_BUDGET_SUMMARY: '/api/v1/cyber/vciso/budget/summary',
  CYBER_VCISO_AWARENESS: '/api/v1/cyber/vciso/awareness',
  CYBER_VCISO_IAM_FINDINGS: '/api/v1/cyber/vciso/iam-findings',
  CYBER_VCISO_IAM_SUMMARY: '/api/v1/cyber/vciso/iam-findings/summary',
  CYBER_VCISO_ESCALATION_RULES: '/api/v1/cyber/vciso/escalation-rules',
  CYBER_VCISO_PLAYBOOKS: '/api/v1/cyber/vciso/playbooks',
  CYBER_VCISO_OBLIGATIONS: '/api/v1/cyber/vciso/obligations',
  CYBER_VCISO_CONTROL_TESTS: '/api/v1/cyber/vciso/control-tests',
  CYBER_VCISO_CONTROL_DEPENDENCIES: '/api/v1/cyber/vciso/control-dependencies',
  CYBER_VCISO_INTEGRATIONS: '/api/v1/cyber/vciso/integrations',
  CYBER_VCISO_CONTROL_OWNERSHIP: '/api/v1/cyber/vciso/control-ownership',
  CYBER_VCISO_APPROVALS: '/api/v1/cyber/vciso/approvals',

  // Cyber — MITRE
  CYBER_MITRE_COVERAGE: '/api/v1/cyber/mitre/coverage',
  CYBER_MITRE_TACTICS: '/api/v1/cyber/mitre/tactics',
  CYBER_MITRE_TECHNIQUES: '/api/v1/cyber/mitre/techniques',
  CYBER_MITRE_TECHNIQUE_DETAIL: (id: string) => `/api/v1/cyber/mitre/techniques/${id}`,
  CYBER_MITRE_FRAMEWORK_META: '/api/v1/cyber/mitre/framework-meta',

  // Cyber — Risk
  CYBER_RISK_SCORE: '/api/v1/cyber/risk/score',
  CYBER_RISK_HEATMAP: '/api/v1/cyber/risk/heatmap',
  CYBER_UEBA_DASHBOARD: '/api/v1/cyber/ueba/dashboard',
  CYBER_UEBA_RISK_RANKING: '/api/v1/cyber/ueba/risk-ranking',
  CYBER_UEBA_PROFILES: '/api/v1/cyber/ueba/profiles',
  CYBER_UEBA_ALERTS: '/api/v1/cyber/ueba/alerts',
  CYBER_UEBA_ALERTS_BULK_STATUS: '/api/v1/cyber/ueba/alerts/bulk/status',
  CYBER_UEBA_CONFIG: '/api/v1/cyber/ueba/config',

  // Cyber — CTI (Cyber Threat Intelligence)
  CTI_SEVERITY_LEVELS: '/api/v1/cyber/cti/severity-levels',
  CTI_CATEGORIES: '/api/v1/cyber/cti/categories',
  CTI_REGIONS: '/api/v1/cyber/cti/regions',
  CTI_SECTORS: '/api/v1/cyber/cti/sectors',
  CTI_DATA_SOURCES: '/api/v1/cyber/cti/data-sources',
  CTI_EVENTS: '/api/v1/cyber/cti/events',
  CTI_EVENT_DETAIL: (id: string) => `/api/v1/cyber/cti/events/${id}`,
  CTI_EVENT_TAGS: (id: string) => `/api/v1/cyber/cti/events/${id}/tags`,
  CTI_EVENT_FALSE_POSITIVE: (id: string) => `/api/v1/cyber/cti/events/${id}/false-positive`,
  CTI_EVENT_RESOLVE: (id: string) => `/api/v1/cyber/cti/events/${id}/resolve`,
  CTI_ACTORS: '/api/v1/cyber/cti/actors',
  CTI_ACTOR_DETAIL: (id: string) => `/api/v1/cyber/cti/actors/${id}`,
  CTI_CAMPAIGNS: '/api/v1/cyber/cti/campaigns',
  CTI_CAMPAIGN_DETAIL: (id: string) => `/api/v1/cyber/cti/campaigns/${id}`,
  CTI_CAMPAIGN_STATUS: (id: string) => `/api/v1/cyber/cti/campaigns/${id}/status`,
  CTI_CAMPAIGN_EVENTS: (id: string) => `/api/v1/cyber/cti/campaigns/${id}/events`,
  CTI_CAMPAIGN_IOCS: (id: string) => `/api/v1/cyber/cti/campaigns/${id}/iocs`,
  CTI_BRANDS: '/api/v1/cyber/cti/brands',
  CTI_BRAND_ABUSE: '/api/v1/cyber/cti/brand-abuse',
  CTI_BRAND_ABUSE_DETAIL: (id: string) => `/api/v1/cyber/cti/brand-abuse/${id}`,
  CTI_BRAND_ABUSE_TAKEDOWN: (id: string) => `/api/v1/cyber/cti/brand-abuse/${id}/takedown-status`,
  CTI_DASHBOARD_THREAT_MAP: '/api/v1/cyber/cti/dashboard/threat-map',
  CTI_DASHBOARD_SECTORS: '/api/v1/cyber/cti/dashboard/sectors',
  CTI_DASHBOARD_EXECUTIVE: '/api/v1/cyber/cti/dashboard/executive',
  CTI_ADMIN_REFRESH: '/api/v1/cyber/cti/admin/refresh-aggregations',

  // Jobs
  JOBS: '/api/v1/jobs',

  // Data
  DATA_SOURCES: '/api/v1/data/sources',
  DATA_SOURCES_STATS: '/api/v1/data/sources/stats',
  DATA_PIPELINES: '/api/v1/data/pipelines',
  DATA_PIPELINES_COUNT: '/api/v1/data/pipelines/count',
  DATA_PIPELINES_STATS: '/api/v1/data/pipelines/stats',
  DATA_MODELS: '/api/v1/data/models',
  DATA_DATASETS: '/api/v1/data/models',
  DATA_QUALITY: '/api/v1/data/quality/dashboard',
  DATA_QUALITY_DASHBOARD: '/api/v1/data/quality/dashboard',
  DATA_QUALITY_SCORE: '/api/v1/data/quality/score',
  DATA_QUALITY_TREND: '/api/v1/data/quality/score/trend',
  DATA_QUALITY_RULES: '/api/v1/data/quality/rules',
  DATA_QUALITY_RESULTS: '/api/v1/data/quality/results',
  DATA_CONTRADICTIONS: '/api/v1/data/contradictions',
  DATA_CONTRADICTIONS_STATS: '/api/v1/data/contradictions/stats',
  DATA_LINEAGE_GRAPH: '/api/v1/data/lineage/graph',
  DATA_LINEAGE_STATS: '/api/v1/data/lineage/stats',
  DATA_DARK_DATA: '/api/v1/data/dark-data',
  DATA_DARK_DATA_STATS: '/api/v1/data/dark-data/stats',
  DATA_ANALYTICS_QUERY: '/api/v1/data/analytics/query',
  DATA_ANALYTICS_SAVED: '/api/v1/data/analytics/saved',
  DATA_ANALYTICS_AUDIT: '/api/v1/data/analytics/audit',
  DATA_DASHBOARD: '/api/v1/data/dashboard',

  // Acta
  ACTA_COMMITTEES: '/api/v1/acta/committees',
  ACTA_MEETINGS: '/api/v1/acta/meetings',
  ACTA_ACTION_ITEMS: '/api/v1/acta/action-items',
  ACTA_DOCUMENTS: '/api/v1/acta/documents',
  ACTA_TEMPLATES: '/api/v1/acta/templates',

  // Lex
  LEX_CONTRACTS: '/api/v1/lex/contracts',
  LEX_DOCUMENTS: '/api/v1/lex/documents',
  LEX_COMPLIANCE_RULES: '/api/v1/lex/compliance/rules',
  LEX_COMPLIANCE_ALERTS: '/api/v1/lex/compliance/alerts',
  LEX_COMPLIANCE_DASHBOARD: '/api/v1/lex/compliance/dashboard',
  LEX_COMPLIANCE_RUN: '/api/v1/lex/compliance/run',

  // Visus
  VISUS_DASHBOARDS: '/api/v1/visus/dashboards',
  VISUS_REPORTS: '/api/v1/visus/reports',
  VISUS_WIDGETS: '/api/v1/visus/widgets',

  // Workflows
  WORKFLOWS_TASKS_COUNT: '/api/v1/workflows/tasks/count',
  WORKFLOWS_TASKS: '/api/v1/workflows/tasks',

  // Notifications
  NOTIFICATIONS: '/api/v1/notifications',
  NOTIFICATIONS_COUNTS: '/api/v1/notifications/counts',
  NOTIFICATIONS_UNREAD_COUNT: '/api/v1/notifications/unread-count',
  NOTIFICATIONS_READ_ALL: '/api/v1/notifications/read-all',
  NOTIFICATIONS_BULK_DELETE: '/api/v1/notifications/bulk',
  NOTIFICATIONS_PREFERENCES: '/api/v1/notifications/preferences',
  NOTIFICATIONS_WEBHOOKS: '/api/v1/notifications/webhooks',
  NOTIFICATIONS_DELIVERY_STATS: '/api/v1/notifications/delivery-stats',
  NOTIFICATIONS_TEST: '/api/v1/notifications/test',
  NOTIFICATIONS_RETRY_FAILED: '/api/v1/notifications/retry-failed',

  // Audit
  AUDIT_LOGS: '/api/v1/audit/logs',

  // Files
  FILES: '/api/v1/files',

  // Workflows
  WORKFLOWS_INSTANCES: '/api/v1/workflows/instances',
  WORKFLOWS_DEFINITIONS: '/api/v1/workflows/definitions',
  WORKFLOWS_TEMPLATES: '/api/v1/workflows/templates',
  WORKFLOWS_DEFINITIONS_FROM_TEMPLATE: '/api/v1/workflows/definitions/from-template',
} as const;

// Route paths
export const ROUTES = {
  HOME: '/',
  DASHBOARD: '/dashboard',
  LOGIN: '/login',
  REGISTER: '/register',
  VERIFY_EMAIL: '/verify',
  INVITE: '/invite',
  SETUP: '/setup',
  FORGOT_PASSWORD: '/forgot-password',
  RESET_PASSWORD: '/reset-password',

  // Dashboard routes
  CYBER: '/cyber',
  CYBER_VCISO: '/cyber/vciso',
  CYBER_UEBA: '/cyber/ueba',
  CYBER_ALERTS: '/cyber/alerts',
  CYBER_ASSETS: '/cyber/assets',
  CYBER_THREATS: '/cyber/threats',
  CYBER_CTI: '/cyber/cti',
  CYBER_CTI_EVENTS: '/cyber/cti/events',
  CYBER_CTI_CAMPAIGNS: '/cyber/cti/campaigns',
  CYBER_CTI_ACTORS: '/cyber/cti/actors',
  CYBER_CTI_BRAND_ABUSE: '/cyber/cti/brand-abuse',
  CYBER_CTI_SECTORS: '/cyber/cti/sectors',
  CYBER_CTI_GEO: '/cyber/cti/geo',
  CYBER_INDICATORS: '/cyber/indicators',
  CYBER_THREAT_FEEDS: '/cyber/threat-feeds',
  DATA: '/data',
  DATA_PIPELINES: '/data/pipelines',
  ACTA: '/acta',
  LEX: '/lex',
  VISUS: '/visus',
  ADMIN_USERS: '/admin/users',
  ADMIN_ROLES: '/admin/roles',
  ADMIN_AUDIT: '/admin/audit',
  WORKFLOWS: '/workflows',
  WORKFLOWS_TASKS: '/workflows/tasks',
  WORKFLOWS_DEFINITIONS: '/admin/workflows/definitions',
  WORKFLOWS_TEMPLATES: '/admin/workflows/templates',
  NOTIFICATIONS: '/notifications',
  ADMIN_NOTIFICATIONS: '/admin/notifications',
  ADMIN_NOTIFICATIONS_WEBHOOKS: '/admin/notifications/webhooks',
  SETTINGS: '/settings',
  SETTINGS_NOTIFICATIONS: '/settings/notifications',
} as const;

// Cookie names
export const COOKIES = {
  ACCESS: 'clario360_access',
  REFRESH: 'clario360_refresh',
} as const;

// Error codes from backend
export const ERROR_CODES = {
  INVALID_CREDENTIALS: 'INVALID_CREDENTIALS',
  ACCOUNT_LOCKED: 'ACCOUNT_LOCKED',
  ACCOUNT_SUSPENDED: 'ACCOUNT_SUSPENDED',
  MFA_REQUIRED: 'MFA_REQUIRED',
  MFA_INVALID: 'MFA_INVALID',
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  RATE_LIMITED: 'RATE_LIMITED',
  EMAIL_TAKEN: 'EMAIL_TAKEN',
  NETWORK_ERROR: 'NETWORK_ERROR',
  TIMEOUT: 'TIMEOUT',
} as const;

// Session config
export const SESSION = {
  ACCESS_TOKEN_MAX_AGE: parseInt(process.env.AUTH_ACCESS_TOKEN_MAX_AGE ?? '900', 10),
  REFRESH_TOKEN_MAX_AGE: parseInt(process.env.AUTH_REFRESH_TOKEN_MAX_AGE ?? '604800', 10),
  COOKIE_SECURE: process.env.AUTH_COOKIE_SECURE !== 'false',
  COOKIE_DOMAIN: process.env.AUTH_COOKIE_DOMAIN ?? 'localhost',
  COOKIE_SAMESITE: (process.env.AUTH_COOKIE_SAMESITE ?? 'strict') as 'strict' | 'lax' | 'none',
} as const;
