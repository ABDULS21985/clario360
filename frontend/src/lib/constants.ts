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

  // Cyber — Vulnerabilities
  CYBER_VULNERABILITIES_AGING: '/api/v1/cyber/vulnerabilities/aging',

  // Cyber — Threats
  CYBER_THREATS: '/api/v1/cyber/threats',
  CYBER_INDICATORS_CHECK: '/api/v1/cyber/indicators/check',

  // Cyber — Detection Rules
  CYBER_RULES: '/api/v1/cyber/rules',
  CYBER_RULE_TEMPLATES: '/api/v1/cyber/rules/templates',

  // Cyber — CTEM
  CYBER_CTEM_ASSESSMENTS: '/api/v1/cyber/ctem/assessments',
  CYBER_CTEM_EXPOSURE_SCORE: '/api/v1/cyber/ctem/exposure-score',
  CYBER_CTEM_EXPOSURE_HISTORY: '/api/v1/cyber/ctem/exposure-score/history',

  // Cyber — Remediation
  CYBER_REMEDIATION: '/api/v1/cyber/remediation',
  CYBER_REMEDIATION_STATS: '/api/v1/cyber/remediation/stats',

  // Cyber — DSPM
  CYBER_DSPM_DASHBOARD: '/api/v1/cyber/dspm/dashboard',
  CYBER_DSPM_DATA_ASSETS: '/api/v1/cyber/dspm/data-assets',
  CYBER_DSPM_SCAN: '/api/v1/cyber/dspm/scan',
  CYBER_DSPM_SHADOW_COPIES: '/api/v1/cyber/dspm/shadow-copies',

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

  // Cyber — MITRE
  CYBER_MITRE_COVERAGE: '/api/v1/cyber/mitre/coverage',
  CYBER_MITRE_TACTICS: '/api/v1/cyber/mitre/tactics',
  CYBER_MITRE_TECHNIQUES: '/api/v1/cyber/mitre/techniques',

  // Cyber — Risk
  CYBER_RISK_SCORE: '/api/v1/cyber/risk/score',
  CYBER_RISK_HEATMAP: '/api/v1/cyber/risk/heatmap',
  CYBER_UEBA_DASHBOARD: '/api/v1/cyber/ueba/dashboard',
  CYBER_UEBA_RISK_RANKING: '/api/v1/cyber/ueba/risk-ranking',
  CYBER_UEBA_PROFILES: '/api/v1/cyber/ueba/profiles',
  CYBER_UEBA_ALERTS: '/api/v1/cyber/ueba/alerts',
  CYBER_UEBA_CONFIG: '/api/v1/cyber/ueba/config',

  // Jobs
  JOBS: '/api/v1/jobs',

  // Data
  DATA_SOURCES: '/api/v1/data/sources',
  DATA_SOURCES_STATS: '/api/v1/data/sources/stats',
  DATA_PIPELINES: '/api/v1/data/pipelines',
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
