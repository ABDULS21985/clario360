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

  // BFF routes (Next.js API routes)
  BFF_SESSION: '/api/auth/session',
  BFF_REFRESH: '/api/auth/refresh',

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

  // Cyber — vCISO
  CYBER_VCISO_BRIEFING: '/api/v1/cyber/vciso/briefing',
  CYBER_VCISO_REPORT: '/api/v1/cyber/vciso/report',

  // Cyber — MITRE
  CYBER_MITRE_COVERAGE: '/api/v1/cyber/mitre/coverage',

  // Cyber — Risk
  CYBER_RISK_SCORE: '/api/v1/cyber/risk/score',

  // Data
  DATA_PIPELINES: '/api/v1/data/pipelines',
  DATA_QUALITY_SCORE: '/api/v1/data/quality/score',

  // Workflows
  WORKFLOWS_TASKS_COUNT: '/api/v1/workflows/tasks/count',
  WORKFLOWS_TASKS: '/api/v1/workflows/tasks',

  // Notifications
  NOTIFICATIONS: '/api/v1/notifications',
  NOTIFICATIONS_UNREAD_COUNT: '/api/v1/notifications/unread-count',
  NOTIFICATIONS_PREFERENCES: '/api/v1/notifications/preferences',

  // Audit
  AUDIT_LOGS: '/api/v1/audit/logs',

  // Files
  FILES: '/api/v1/files',

  // Workflows
  WORKFLOWS_INSTANCES: '/api/v1/workflows/instances',
} as const;

// Route paths
export const ROUTES = {
  HOME: '/',
  DASHBOARD: '/dashboard',
  LOGIN: '/login',
  REGISTER: '/register',
  FORGOT_PASSWORD: '/forgot-password',
  RESET_PASSWORD: '/reset-password',

  // Dashboard routes
  CYBER: '/cyber',
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
  NOTIFICATIONS: '/notifications',
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
