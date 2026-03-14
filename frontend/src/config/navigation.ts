import {
  LayoutDashboard,
  Shield,
  Monitor,
  AlertTriangle,
  Search,
  Target,
  Wrench,
  Database,
  Bot,
  ShieldCheck,
  Fingerprint,
  Grid3X3,
  Rss,
  BarChart3,
  FolderOpen,
  Boxes,
  GitBranch,
  CheckCircle,
  Zap,
  Package,
  TrendingUp,
  Users,
  BookOpen,
  Scale,
  FileText,
  Eye,
  FileBarChart,
  Settings,
  KeyRound,
  ClipboardList,
  Workflow,
  Bell,
  BellRing,
  File,
  Building2,
  Gavel,
  Plug,
  Key,
  Mail,
  Layout,
  Server,
  BookMarked,
  FileCheck2,
  Handshake,
  Archive,
  Gauge,
  GraduationCap,
  Siren,
  CircuitBoard,
  ListChecks,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

export interface BadgeConfig {
  endpoint: string;
  key: string;
  variant: 'default' | 'destructive' | 'warning';
  pollIntervalMs: number;
}

export interface NavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  permission?: string;
  badge?: BadgeConfig;
  children?: NavItem[];
}

export interface NavSection {
  id: string;
  label: string;
  permission: string;
  items: NavItem[];
}

export const navigation: NavSection[] = [
  {
    id: 'main',
    label: '',
    permission: '*:read',
    items: [
      {
        id: 'dashboard',
        label: 'Dashboard',
        href: '/dashboard',
        icon: LayoutDashboard,
      },
      {
        id: 'notebooks',
        label: 'Notebook Workspace',
        href: '/notebooks',
        icon: Bot,
      },
    ],
  },
  {
    id: 'cybersecurity',
    label: 'CYBERSECURITY',
    permission: 'cyber:read',
    items: [
      { id: 'cyber-overview', label: 'Overview', href: '/cyber', icon: Shield },
      { id: 'cyber-assets', label: 'Assets', href: '/cyber/assets', icon: Monitor },
      {
        id: 'cyber-alerts',
        label: 'Alerts',
        href: '/cyber/alerts',
        icon: AlertTriangle,
        badge: {
          endpoint: '/api/v1/cyber/alerts/count?status=new,acknowledged',
          key: 'count',
          variant: 'destructive',
          pollIntervalMs: 30000,
        },
      },
      { id: 'cyber-indicators', label: 'IOC Management', href: '/cyber/indicators', icon: Fingerprint, permission: 'cyber:read' },
      { id: 'cyber-feeds', label: 'Threat Feeds', href: '/cyber/threat-feeds', icon: Rss, permission: 'cyber:manage' },
      { id: 'cyber-threats', label: 'Threat Hunting', href: '/cyber/threats', icon: Search },
      { id: 'cyber-ctem', label: 'CTEM Assessments', href: '/cyber/ctem', icon: Target },
      {
        id: 'cyber-remediation',
        label: 'Remediation',
        href: '/cyber/remediation',
        icon: Wrench,
        permission: 'remediation:read',
      },
      {
        id: 'cyber-dspm',
        label: 'DSPM',
        href: '/cyber/dspm',
        icon: Database,
        children: [
          { id: 'cyber-dspm-overview', label: 'Overview', href: '/cyber/dspm', icon: Database },
          { id: 'cyber-dspm-assets', label: 'Data Assets', href: '/cyber/dspm/assets', icon: FolderOpen },
          { id: 'cyber-dspm-remediations', label: 'Remediations', href: '/cyber/dspm/remediations', icon: Wrench },
          { id: 'cyber-dspm-policies', label: 'Policies', href: '/cyber/dspm/policies', icon: FileText },
          { id: 'cyber-dspm-exceptions', label: 'Exceptions', href: '/cyber/dspm/exceptions', icon: AlertTriangle },
          { id: 'cyber-dspm-compliance', label: 'Compliance', href: '/cyber/dspm/compliance', icon: Scale },
        ],
      },
      { id: 'cyber-ueba', label: 'UEBA', href: '/cyber/ueba', icon: Fingerprint },
      {
        id: 'cyber-vciso',
        label: 'Virtual CISO',
        href: '/cyber/vciso',
        icon: Bot,
        children: [
          { id: 'cyber-vciso-briefing', label: 'Briefing', href: '/cyber/vciso', icon: Bot },
          { id: 'cyber-vciso-risk-register', label: 'Risk Register', href: '/cyber/vciso/risk-register', icon: BookMarked },
          { id: 'cyber-vciso-policies', label: 'Policies', href: '/cyber/vciso/policies', icon: FileText },
          { id: 'cyber-vciso-compliance', label: 'Compliance', href: '/cyber/vciso/compliance', icon: FileCheck2 },
          { id: 'cyber-vciso-third-party', label: 'Third-Party Risk', href: '/cyber/vciso/third-party', icon: Handshake },
          { id: 'cyber-vciso-evidence', label: 'Evidence', href: '/cyber/vciso/evidence', icon: Archive },
          { id: 'cyber-vciso-maturity', label: 'Maturity & Budget', href: '/cyber/vciso/maturity', icon: Gauge },
          { id: 'cyber-vciso-awareness', label: 'Awareness & IAM', href: '/cyber/vciso/awareness', icon: GraduationCap },
          { id: 'cyber-vciso-incident-readiness', label: 'Incident Readiness', href: '/cyber/vciso/incident-readiness', icon: Siren },
          { id: 'cyber-vciso-integrations', label: 'Integrations', href: '/cyber/vciso/integrations', icon: CircuitBoard },
          { id: 'cyber-vciso-workflows', label: 'Workflows', href: '/cyber/vciso/workflows', icon: ListChecks },
        ],
      },
      { id: 'cyber-detection', label: 'Detection Rules', href: '/cyber/detection-rules', icon: ShieldCheck, permission: 'cyber:read' },
      { id: 'cyber-mitre', label: 'MITRE ATT&CK', href: '/cyber/mitre-attack', icon: Grid3X3, permission: 'cyber:read' },
    ],
  },
  {
    id: 'data-intelligence',
    label: 'DATA INTELLIGENCE',
    permission: 'data:read',
    items: [
      { id: 'data-overview', label: 'Overview', href: '/data', icon: BarChart3 },
      { id: 'data-sources', label: 'Data Sources', href: '/data/sources', icon: FolderOpen },
      { id: 'data-models', label: 'Data Models', href: '/data/models', icon: Boxes },
      {
        id: 'data-pipelines',
        label: 'Pipelines',
        href: '/data/pipelines',
        icon: GitBranch,
        badge: {
          endpoint: '/api/v1/data/pipelines/count?status=failed',
          key: 'count',
          variant: 'warning',
          pollIntervalMs: 60000,
        },
      },
      { id: 'data-quality', label: 'Data Quality', href: '/data/quality', icon: CheckCircle },
      { id: 'data-contradictions', label: 'Contradictions', href: '/data/contradictions', icon: Zap },
      { id: 'data-dark', label: 'Dark Data', href: '/data/dark-data', icon: Package },
      { id: 'data-analytics', label: 'Analytics', href: '/data/analytics', icon: TrendingUp },
    ],
  },
  {
    id: 'enterprise',
    label: 'ENTERPRISE',
    permission: '*:read',
    items: [
      { id: 'acta-overview', label: 'Acta Overview', href: '/acta', icon: Building2, permission: 'acta:read' },
      { id: 'acta-committees', label: 'Acta Committees', href: '/acta/committees', icon: Users, permission: 'acta:read' },
      { id: 'acta-meetings', label: 'Acta Meetings', href: '/acta/meetings', icon: BookOpen, permission: 'acta:read' },
      { id: 'acta-actions', label: 'Acta Action Items', href: '/acta/action-items', icon: ClipboardList, permission: 'acta:read' },
      { id: 'lex-overview', label: 'Lex Overview', href: '/lex', icon: Gavel, permission: 'lex:read' },
      { id: 'lex-contracts', label: 'Lex Contracts', href: '/lex/contracts', icon: FileText, permission: 'lex:read' },
      { id: 'lex-documents', label: 'Lex Documents', href: '/lex/documents', icon: File, permission: 'lex:read' },
      { id: 'lex-compliance', label: 'Lex Compliance', href: '/lex/compliance', icon: Scale, permission: 'lex:read' },
      { id: 'visus-dashboard', label: 'Visus Dashboard', href: '/visus', icon: Eye, permission: 'visus:read' },
      { id: 'visus-kpis', label: 'Visus KPIs', href: '/visus/kpis', icon: TrendingUp, permission: 'visus:read' },
      { id: 'visus-reports', label: 'Visus Reports', href: '/visus/reports', icon: FileBarChart, permission: 'visus:read' },
      { id: 'visus-alerts', label: 'Visus Alerts', href: '/visus/alerts', icon: Bell, permission: 'visus:read' },
    ],
  },
  {
    id: 'platform',
    label: 'PLATFORM',
    permission: 'users:read',
    items: [
      {
        id: 'admin-users',
        label: 'Users',
        href: '/admin/users',
        icon: Users,
        permission: 'users:read',
      },
      {
        id: 'admin-roles',
        label: 'Roles',
        href: '/admin/roles',
        icon: KeyRound,
        permission: 'roles:read',
      },
      {
        id: 'admin-tenants',
        label: 'Tenants',
        href: '/admin/tenants',
        icon: Building2,
        permission: 'admin:tenants',
      },
      {
        id: 'admin-api-keys',
        label: 'API Keys',
        href: '/admin/api-keys',
        icon: Key,
        permission: 'users:read',
      },
      {
        id: 'admin-invitations',
        label: 'Invitations',
        href: '/admin/invitations',
        icon: Mail,
        permission: 'users:write',
      },
      {
        id: 'admin-audit',
        label: 'Audit Logs',
        href: '/admin/audit',
        icon: ClipboardList,
        permission: '*:read',
      },
      {
        id: 'admin-ai-governance',
        label: 'AI Governance',
        href: '/admin/ai-governance',
        icon: Bot,
        permission: 'users:read',
        children: [
          { id: 'admin-ai-governance-models', label: 'Model Registry', href: '/admin/ai-governance', icon: Bot, permission: 'users:read' },
          { id: 'admin-ai-governance-compute', label: 'Compute', href: '/admin/ai-governance/compute', icon: Server, permission: 'users:read' },
          { id: 'admin-ai-governance-benchmarks', label: 'Benchmarks', href: '/admin/ai-governance/benchmarks', icon: BarChart3, permission: 'users:read' },
        ],
      },
      {
        id: 'admin-settings',
        label: 'Settings',
        href: '/admin/settings',
        icon: Settings,
        permission: 'tenant:write',
      },
      {
        id: 'admin-integrations',
        label: 'Integrations',
        href: '/admin/integrations',
        icon: Plug,
        permission: 'tenant:write',
      },
      {
        id: 'admin-workflows',
        label: 'Workflows',
        href: '/workflows',
        icon: Workflow,
        badge: {
          endpoint: '/api/v1/workflows/tasks/count',
          key: 'pending',
          variant: 'default',
          pollIntervalMs: 30000,
        },
        children: [
          { id: 'admin-workflow-tasks', label: 'Task Queue', href: '/admin/workflows/tasks', icon: ClipboardList },
          { id: 'admin-workflow-instances', label: 'Instances', href: '/admin/workflows/instances', icon: Workflow },
          { id: 'admin-workflow-definitions', label: 'Definitions', href: '/admin/workflows/definitions', icon: GitBranch },
          { id: 'admin-workflow-templates', label: 'Templates', href: '/admin/workflows/templates', icon: Layout },
        ],
      },
      { id: 'admin-files', label: 'Files', href: '/files', icon: File },
      { id: 'admin-notifications', label: 'Notifications', href: '/notifications', icon: Bell },
      {
        id: 'admin-notification-management',
        label: 'Notification Mgmt',
        href: '/admin/notifications',
        icon: BellRing,
        permission: 'tenant:write',
      },
    ],
  },
];
