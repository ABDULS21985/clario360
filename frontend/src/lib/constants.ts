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

  // Cyber
  CYBER_ALERTS_COUNT: '/api/v1/cyber/alerts/count',
  CYBER_ALERTS: '/api/v1/cyber/alerts',

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
