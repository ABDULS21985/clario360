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
  File,
  Building2,
  Gavel,
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
      { id: 'cyber-vciso', label: 'Virtual CISO', href: '/cyber/vciso', icon: Bot },
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
    id: 'governance',
    label: 'GOVERNANCE',
    permission: 'acta:read',
    items: [
      { id: 'acta-overview', label: 'Board (Acta)', href: '/acta', icon: Building2 },
      { id: 'acta-committees', label: 'Committees', href: '/acta/committees', icon: Users },
      { id: 'acta-meetings', label: 'Meetings', href: '/acta/meetings', icon: BookOpen },
      { id: 'acta-actions', label: 'Action Items', href: '/acta/action-items', icon: ClipboardList },
    ],
  },
  {
    id: 'legal',
    label: 'LEGAL',
    permission: 'lex:read',
    items: [
      { id: 'lex-overview', label: 'Overview', href: '/lex', icon: Gavel },
      { id: 'lex-contracts', label: 'Contracts', href: '/lex/contracts', icon: FileText },
      { id: 'lex-documents', label: 'Documents', href: '/lex/documents', icon: File },
      { id: 'lex-compliance', label: 'Compliance', href: '/lex/compliance', icon: Scale },
    ],
  },
  {
    id: 'executive',
    label: 'EXECUTIVE',
    permission: 'visus:read',
    items: [
      { id: 'visus-dashboard', label: 'Visus360', href: '/visus', icon: Eye },
      { id: 'visus-reports', label: 'Reports', href: '/visus/reports', icon: FileBarChart },
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
        id: 'admin-audit',
        label: 'Audit Logs',
        href: '/admin/audit',
        icon: ClipboardList,
        permission: '*:read',
      },
      {
        id: 'admin-settings',
        label: 'Settings',
        href: '/admin/settings',
        icon: Settings,
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
      },
      { id: 'admin-files', label: 'Files', href: '/files', icon: File },
      { id: 'admin-notifications', label: 'Notifications', href: '/notifications', icon: Bell },
    ],
  },
];
