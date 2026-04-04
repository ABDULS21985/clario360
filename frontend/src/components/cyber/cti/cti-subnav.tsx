'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  BarChart3,
  Globe,
  Radar,
  Shield,
  Target,
  Users,
  Zap,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { ROUTES } from '@/lib/constants';

const ITEMS = [
  { href: ROUTES.CYBER_CTI, label: 'Dashboard', icon: Radar },
  { href: ROUTES.CYBER_CTI_EVENTS, label: 'Events', icon: Zap },
  { href: ROUTES.CYBER_CTI_CAMPAIGNS, label: 'Campaigns', icon: Target },
  { href: ROUTES.CYBER_CTI_ACTORS, label: 'Actors', icon: Users },
  { href: ROUTES.CYBER_CTI_BRAND_ABUSE, label: 'Brand Abuse', icon: Shield },
  { href: ROUTES.CYBER_CTI_SECTORS, label: 'Sectors', icon: BarChart3 },
  { href: ROUTES.CYBER_CTI_GEO, label: 'Geo', icon: Globe },
];

export function CTISubnav() {
  const pathname = usePathname() ?? '';

  return (
    <nav
      aria-label="CTI sections"
      className="overflow-x-auto rounded-[22px] border border-[color:var(--card-border)] bg-[rgba(255,255,255,0.74)] p-2 shadow-[var(--card-shadow)] backdrop-blur-md"
    >
      <div className="flex min-w-max items-center gap-2">
        {ITEMS.map((item) => {
          const Icon = item.icon;
          const isActive = item.href === ROUTES.CYBER_CTI
            ? pathname === item.href
            : pathname === item.href || pathname.startsWith(`${item.href}/`);

          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={isActive ? 'page' : undefined}
              className={cn(
                'inline-flex items-center gap-2 rounded-2xl px-4 py-2.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-emerald-50 text-emerald-950 ring-1 ring-emerald-200'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
              )}
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}