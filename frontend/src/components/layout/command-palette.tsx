'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Search, X } from 'lucide-react';
import { useCommandPaletteStore } from '@/stores/command-palette-store';
import { useCommandPalette } from '@/hooks/use-command-palette';
import { useAuth } from '@/hooks/use-auth';
import { navigation } from '@/config/navigation';
import { cn } from '@/lib/utils';
import { enterpriseApi } from '@/lib/enterprise';

interface NavEntry {
  id: string;
  label: string;
  href: string;
  section: string;
  icon: React.ElementType;
}

function buildNavEntries(hasPermission: (p: string) => boolean): NavEntry[] {
  const entries: NavEntry[] = [];
  for (const section of navigation) {
    if (section.permission !== '*:read' && !hasPermission(section.permission)) continue;
    for (const item of section.items) {
      if (item.permission && item.permission !== '*:read' && !hasPermission(item.permission))
        continue;
      entries.push({
        id: item.id,
        label: item.label,
        href: item.href,
        section: section.label || 'Navigation',
        icon: item.icon,
      });
    }
  }
  return entries;
}

export function CommandPalette() {
  useCommandPalette(); // register global Cmd+K listener
  const { open, query, setQuery, close } = useCommandPaletteStore();
  const { hasPermission } = useAuth();
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);
  const [activeIdx, setActiveIdx] = useState(0);

  const navEntries = buildNavEntries(hasPermission);
  const normalizedQuery = query.trim().toLowerCase();
  const recordSearchEnabled = normalizedQuery.length >= 2;
  const meetingsQuery = useQuery({
    queryKey: ['command-palette', 'meetings', normalizedQuery],
    queryFn: async () => {
      if (!hasPermission('acta:read')) return [];
      const response = await enterpriseApi.acta.listMeetings({ page: 1, per_page: 5, search: normalizedQuery, order: 'desc' });
      return response.data.map((meeting) => ({
        id: `meeting-${meeting.id}`,
        label: meeting.title,
        href: `/acta/meetings/${meeting.id}`,
        section: 'Meetings',
        icon: Search,
      }));
    },
    enabled: recordSearchEnabled,
  });
  const committeesQuery = useQuery({
    queryKey: ['command-palette', 'committees', normalizedQuery],
    queryFn: async () => {
      if (!hasPermission('acta:read')) return [];
      const response = await enterpriseApi.acta.listCommittees({ page: 1, per_page: 5, search: normalizedQuery, order: 'desc' });
      return response.data.map((committee) => ({
        id: `committee-${committee.id}`,
        label: committee.name,
        href: `/acta/committees/${committee.id}`,
        section: 'Committees',
        icon: Search,
      }));
    },
    enabled: recordSearchEnabled,
  });
  const contractsQuery = useQuery({
    queryKey: ['command-palette', 'contracts', normalizedQuery],
    queryFn: async () => {
      if (!hasPermission('lex:read')) return [];
      const response = await enterpriseApi.lex.searchContracts(normalizedQuery, { page: 1, per_page: 5, order: 'desc' });
      return response.data.map((contract) => ({
        id: `contract-${contract.id}`,
        label: contract.title,
        href: `/lex/contracts/${contract.id}`,
        section: 'Contracts',
        icon: Search,
      }));
    },
    enabled: recordSearchEnabled,
  });
  const reportsQuery = useQuery({
    queryKey: ['command-palette', 'reports', normalizedQuery],
    queryFn: async () => {
      if (!hasPermission('visus:read')) return [];
      const response = await enterpriseApi.visus.listReports({ page: 1, per_page: 5, search: normalizedQuery, order: 'desc' });
      return response.data.map((report) => ({
        id: `report-${report.id}`,
        label: report.name,
        href: `/visus/reports`,
        section: 'Reports',
        icon: Search,
      }));
    },
    enabled: recordSearchEnabled,
  });

  const filtered = normalizedQuery
    ? [
        ...navEntries.filter((e) => e.label.toLowerCase().includes(normalizedQuery)),
        ...(meetingsQuery.data ?? []),
        ...(committeesQuery.data ?? []),
        ...(contractsQuery.data ?? []),
        ...(reportsQuery.data ?? []),
      ].slice(0, 15)
    : navEntries.slice(0, 10);

  const navigate = useCallback(
    (href: string) => {
      close();
      router.push(href);
    },
    [close, router],
  );

  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50);
      setActiveIdx(0);
    }
  }, [open]);

  useEffect(() => {
    setActiveIdx(0);
  }, [query]);

  if (!open) return null;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIdx((i) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIdx((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter' && filtered[activeIdx]) {
      navigate(filtered[activeIdx].href);
    } else if (e.key === 'Escape') {
      close();
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] bg-black/50 backdrop-blur-sm"
      onClick={close}
    >
      <div
        className="w-full max-w-lg rounded-xl border bg-popover shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
        aria-modal="true"
      >
        {/* Input */}
        <div className="flex items-center gap-3 border-b px-4 py-3">
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search pages and actions..."
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            aria-label="Search"
          />
          {query && (
            <button
              onClick={() => setQuery('')}
              className="text-muted-foreground hover:text-foreground"
              aria-label="Clear search"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>

        {/* Results */}
        <div className="max-h-80 overflow-y-auto py-2">
          {filtered.length === 0 ? (
            <p className="px-4 py-8 text-center text-sm text-muted-foreground">
              No results found for &ldquo;{query}&rdquo;
            </p>
          ) : (
            filtered.map((entry, idx) => {
              const Icon = entry.icon;
              return (
                <button
                  key={entry.id}
                  onClick={() => navigate(entry.href)}
                  className={cn(
                    'flex w-full items-center gap-3 px-4 py-2.5 text-left text-sm transition-colors',
                    idx === activeIdx ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50',
                  )}
                >
                  <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span className="flex-1">{entry.label}</span>
                  {entry.section && (
                    <span className="text-xs text-muted-foreground">{entry.section}</span>
                  )}
                </button>
              );
            })
          )}
        </div>

        {/* Footer hint */}
        <div className="border-t px-4 py-2 flex items-center gap-3 text-xs text-muted-foreground">
          <span>↑↓ navigate</span>
          <span>↵ open</span>
          <span>esc close</span>
        </div>
      </div>
    </div>
  );
}
