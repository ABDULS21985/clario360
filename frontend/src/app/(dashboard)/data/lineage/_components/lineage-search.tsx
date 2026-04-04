'use client';

import { useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search, Loader2 } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useDebounce } from '@/hooks/use-debounce';
import { dataSuiteApi, type LineageNode } from '@/lib/data-suite';

interface LineageSearchProps {
  value: string;
  onChange: (value: string) => void;
  onSelectResult?: (node: LineageNode) => void;
}

export function LineageSearch({
  value,
  onChange,
  onSelectResult,
}: LineageSearchProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const debouncedValue = useDebounce(value, 300);

  const searchQuery = useQuery({
    queryKey: ['lineage-search', debouncedValue],
    queryFn: () => dataSuiteApi.searchLineage(debouncedValue),
    enabled: debouncedValue.length >= 2,
  });

  const results = searchQuery.data ?? [];
  const showDropdown = open && debouncedValue.length >= 2;

  return (
    <div
      ref={containerRef}
      className="relative"
      onBlur={(e) => {
        if (!containerRef.current?.contains(e.relatedTarget as Node)) {
          setOpen(false);
        }
      }}
    >
      <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
      {searchQuery.isFetching && (
        <Loader2 className="absolute right-3 top-3 h-4 w-4 animate-spin text-muted-foreground" />
      )}
      <Input
        className="w-[260px] pl-9"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onFocus={() => setOpen(true)}
        placeholder="Search lineage..."
      />

      {showDropdown && (
        <div className="absolute left-0 top-full z-50 mt-1 w-[340px] rounded-md border bg-popover shadow-md">
          {results.length === 0 && !searchQuery.isFetching && (
            <div className="px-3 py-2 text-sm text-muted-foreground">No results found.</div>
          )}
          {results.length === 0 && searchQuery.isFetching && (
            <div className="px-3 py-2 text-sm text-muted-foreground">Searching...</div>
          )}
          {results.map((result) => (
            <button
              key={result.node.id}
              type="button"
              className="flex w-full items-start gap-2 px-3 py-2 text-left text-sm hover:bg-accent focus:bg-accent focus:outline-none"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                onSelectResult?.(result.node);
                setOpen(false);
              }}
            >
              <div className="min-w-0 flex-1">
                <div className="truncate font-medium">{result.node.name}</div>
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Badge variant="outline" className="text-[10px] px-1 py-0">
                    {result.node.type.replace(/_/g, ' ')}
                  </Badge>
                  {result.match_fields.length > 0 && (
                    <span>matched: {result.match_fields.join(', ')}</span>
                  )}
                </div>
              </div>
              <span className="shrink-0 text-xs text-muted-foreground">
                {Math.round(result.score * 100)}%
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
