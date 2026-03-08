// Cyber suite sidebar navigation configuration.
// These items mirror the cybersecurity section in /src/config/navigation.ts
// and can be imported by any component needing cyber nav context.

import {
  Shield,
  Monitor,
  AlertTriangle,
  Search,
  Target,
  Wrench,
  Database,
  Bot,
  ShieldCheck,
  Grid3X3,
  Fingerprint,
} from 'lucide-react';
import type { NavItem } from '@/config/navigation';

export const CYBER_NAV_ITEMS: NavItem[] = [
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
  {
    id: 'cyber-rules',
    label: 'Detection Rules',
    href: '/cyber/rules',
    icon: ShieldCheck,
    permission: 'cyber:read',
  },
  {
    id: 'cyber-mitre',
    label: 'MITRE Coverage',
    href: '/cyber/mitre',
    icon: Grid3X3,
    permission: 'cyber:read',
  },
];
