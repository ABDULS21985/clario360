export const breadcrumbMap: Record<string, string> = {
  dashboard: 'Dashboard',
  cyber: 'Cybersecurity',
  assets: 'Assets',
  alerts: 'Alerts',
  threats: 'Threat Hunting',
  ctem: 'CTEM Assessments',
  remediation: 'Remediation',
  dspm: 'DSPM',
  vciso: 'Virtual CISO',
  'risk-register': 'Risk Register',
  policies: 'Policies',
  'third-party': 'Third-Party Risk',
  evidence: 'Evidence',
  maturity: 'Maturity & Budget',
  awareness: 'Awareness & IAM',
  'incident-readiness': 'Incident Readiness',
  integrations: 'Integrations',
  data: 'Data Intelligence',
  sources: 'Data Sources',
  models: 'Data Models',
  pipelines: 'Pipelines',
  quality: 'Data Quality',
  contradictions: 'Contradictions',
  'dark-data': 'Dark Data',
  analytics: 'Analytics',
  acta: 'Governance',
  committees: 'Committees',
  meetings: 'Meetings',
  'action-items': 'Action Items',
  lex: 'Legal',
  contracts: 'Contracts',
  documents: 'Documents',
  compliance: 'Compliance',
  visus: 'Executive Intelligence',
  dashboards: 'Dashboards',
  reports: 'Reports',
  admin: 'Platform',
  users: 'Users',
  roles: 'Roles',
  audit: 'Audit Logs',
  settings: 'Settings',
  workflows: 'Workflows',
  tasks: 'Tasks',
  notifications: 'Notifications',
  files: 'Files',
  notifications_prefs: 'Notification Preferences',
  tenants: 'Tenants',
  'api-keys': 'API Keys',
  invitations: 'Invitations',
  new: 'New',
  callback: 'Callback',
};

export const UUID_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function isUUID(segment: string): boolean {
  return UUID_REGEX.test(segment);
}

export function segmentToLabel(segment: string): string {
  if (breadcrumbMap[segment]) return breadcrumbMap[segment];
  // Title-case hyphenated slugs
  return segment
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
