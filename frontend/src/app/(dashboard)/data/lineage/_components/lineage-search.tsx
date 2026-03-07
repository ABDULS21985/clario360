'use client';

import { Search } from 'lucide-react';
import { Input } from '@/components/ui/input';

interface LineageSearchProps {
  value: string;
  onChange: (value: string) => void;
}

export function LineageSearch({
  value,
  onChange,
}: LineageSearchProps) {
  return (
    <div className="relative">
      <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
      <Input className="w-[260px] pl-9" value={value} onChange={(event) => onChange(event.target.value)} placeholder="Search lineage..." />
    </div>
  );
}
