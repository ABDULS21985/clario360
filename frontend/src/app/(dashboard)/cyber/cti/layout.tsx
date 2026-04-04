import type { ReactNode } from 'react';
import { CTISubnav } from '@/components/cyber/cti/cti-subnav';

interface CTILayoutProps {
  children: ReactNode;
}

export default function CTILayout({ children }: CTILayoutProps) {
  return (
    <div className="space-y-6">
      <CTISubnav />
      {children}
    </div>
  );
}
