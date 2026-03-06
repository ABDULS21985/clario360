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
