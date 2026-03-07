'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { X, Search } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Button } from '@/components/ui/button';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { MITRECoverage } from '@/types/cyber';

interface RuleMitreSelectorProps {
  value: string[];
  onChange: (ids: string[]) => void;
}

export function RuleMitreSelector({ value, onChange }: RuleMitreSelectorProps) {
  const [search, setSearch] = useState('');
  const [open, setOpen] = useState(false);

  const { data: envelope } = useQuery({
    queryKey: ['mitre-coverage-for-selector'],
    queryFn: () => apiGet<{ data: MITRECoverage }>(API_ENDPOINTS.CYBER_MITRE_COVERAGE),
    staleTime: 300000,
  });

  const techniques = envelope?.data?.techniques ?? [];
  const filtered = techniques.filter((t) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return t.technique_id.toLowerCase().includes(q) || t.technique_name.toLowerCase().includes(q);
  });

  function toggle(id: string) {
    onChange(value.includes(id) ? value.filter((v) => v !== id) : [...value, id]);
  }

  function remove(id: string) {
    onChange(value.filter((v) => v !== id));
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5">
        {value.map((id) => {
          const t = techniques.find((t) => t.technique_id === id);
          return (
            <Badge key={id} variant="secondary" className="gap-1 pl-2 pr-1 font-mono">
              {id}
              {t && <span className="ml-0.5 max-w-[120px] truncate text-xs text-muted-foreground">— {t.technique_name}</span>}
              <button
                type="button"
                onClick={() => remove(id)}
                className="ml-0.5 rounded hover:bg-muted"
                aria-label={`Remove ${id}`}
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          );
        })}
      </div>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" type="button">
            <Search className="mr-1.5 h-3.5 w-3.5" />
            Add MITRE Technique
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-2" align="start">
          <Input
            placeholder="Search T1059, PowerShell…"
            className="mb-2 h-7 text-xs"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            autoFocus
          />
          <ScrollArea className="h-56">
            {filtered.length === 0 ? (
              <p className="py-4 text-center text-xs text-muted-foreground">No techniques found</p>
            ) : (
              <div className="space-y-0.5">
                {filtered.map((t) => (
                  <button
                    key={t.technique_id}
                    type="button"
                    className={`flex w-full items-start gap-2 rounded px-2 py-1.5 text-left text-xs transition-colors hover:bg-muted/50 ${
                      value.includes(t.technique_id) ? 'bg-primary/10 font-medium' : ''
                    }`}
                    onClick={() => toggle(t.technique_id)}
                  >
                    <span className="shrink-0 font-mono text-muted-foreground">{t.technique_id}</span>
                    <span className="truncate">{t.technique_name}</span>
                    {value.includes(t.technique_id) && <span className="ml-auto text-primary">✓</span>}
                  </button>
                ))}
              </div>
            )}
          </ScrollArea>
        </PopoverContent>
      </Popover>
    </div>
  );
}
