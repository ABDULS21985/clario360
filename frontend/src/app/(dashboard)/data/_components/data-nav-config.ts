import {
  BarChart3,
  Boxes,
  CheckCircle2,
  FileQuestion,
  FolderOpen,
  GitBranch,
  Network,
  Search,
  ShieldAlert,
} from 'lucide-react';

export const dataNavConfig = [
  { id: 'overview', label: 'Overview', href: '/data', icon: BarChart3 },
  { id: 'sources', label: 'Sources', href: '/data/sources', icon: FolderOpen },
  { id: 'models', label: 'Models', href: '/data/models', icon: Boxes },
  { id: 'pipelines', label: 'Pipelines', href: '/data/pipelines', icon: GitBranch },
  { id: 'quality', label: 'Quality', href: '/data/quality', icon: CheckCircle2 },
  { id: 'contradictions', label: 'Contradictions', href: '/data/contradictions', icon: ShieldAlert },
  { id: 'lineage', label: 'Lineage', href: '/data/lineage', icon: Network },
  { id: 'dark-data', label: 'Dark Data', href: '/data/dark-data', icon: FileQuestion },
  { id: 'analytics', label: 'Analytics', href: '/data/analytics', icon: Search },
] as const;
