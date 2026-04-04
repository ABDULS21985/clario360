'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { SearchInput } from '@/components/shared/forms/search-input';
import { useDebounce } from '@/hooks/use-debounce';

interface EntityLinkDialogProps<TItem> {
  title: string;
  description?: string;
  searchPlaceholder: string;
  searchFn: (query: string) => Promise<TItem[]>;
  renderItem: (item: TItem) => React.ReactNode;
  onSelect: (item: TItem) => Promise<void>;
  isOpen: boolean;
  onClose: () => void;
  getKey: (item: TItem) => string;
}

export function EntityLinkDialog<TItem>({
  title,
  description,
  searchPlaceholder,
  searchFn,
  renderItem,
  onSelect,
  isOpen,
  onClose,
  getKey,
}: EntityLinkDialogProps<TItem>) {
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 250);

  const resultsQuery = useQuery({
    queryKey: [title, debouncedSearch],
    queryFn: () => searchFn(debouncedSearch),
    enabled: isOpen,
  });

  const results = resultsQuery.data ?? [];

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {description ?? 'Search linked CTI entities and attach the most relevant record.'}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <SearchInput
            value={search}
            onChange={setSearch}
            placeholder={searchPlaceholder}
            loading={resultsQuery.isFetching}
          />
          <div className="max-h-[420px] overflow-y-auto rounded-[22px] border border-[color:var(--card-border)] bg-[var(--card-bg)]">
            {results.length > 0 ? (
              <div className="divide-y">
                {results.map((item) => (
                  <button
                    key={getKey(item)}
                    type="button"
                    className="block w-full px-4 py-3 text-left transition hover:bg-muted/40"
                    onClick={() => void onSelect(item)}
                  >
                    {renderItem(item)}
                  </button>
                ))}
              </div>
            ) : (
              <div className="px-4 py-8 text-center text-sm text-muted-foreground">
                {resultsQuery.isLoading ? 'Loading results…' : 'No results match the current search.'}
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
