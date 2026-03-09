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
      { id: 'cyber-threats', label: 'Threat Hunting', href: '/cyber/threats', icon: Search },
      { id: 'cyber-ctem', label: 'CTEM Assessments', href: '/cyber/ctem', icon: Target },
      {
        id: 'cyber-remediation',
        label: 'Remediation',
        href: '/cyber/remediation',
        icon: Wrench,
        permission: 'remediation:read',
      },
      { id: 'cyber-dspm', label: 'DSPM', href: '/cyber/dspm', icon: Database },
      { id: 'cyber-ueba', label: 'UEBA', href: '/cyber/ueba', icon: Fingerprint },
      { id: 'cyber-vciso', label: 'Virtual CISO', href: '/cyber/vciso', icon: Bot },
      { id: 'cyber-rules', label: 'Detection Rules', href: '/cyber/rules', icon: ShieldCheck, permission: 'cyber:read' },
      { id: 'cyber-mitre', label: 'MITRE Coverage', href: '/cyber/mitre', icon: Grid3X3, permission: 'cyber:read' },
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
          { id: 'admin-workflow-tasks', label: 'Task Queue', href: '/workflows/tasks', icon: ClipboardList },
          { id: 'admin-workflow-instances', label: 'Instances', href: '/workflows', icon: Workflow },
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
