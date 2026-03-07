'use client';

import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { useApiMutation } from '@/hooks/use-api-mutation';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { EmptyState } from '@/components/common/empty-state';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { API_ENDPOINTS } from '@/lib/constants';
import { Search, ShieldCheck } from 'lucide-react';
import type { RuleTemplate } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';

interface RuleTemplatePickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onActivated: () => void;
}

interface ActivatePayload {
  template_id: string;
}

const CATEGORY_ALL = 'All';

export function RuleTemplatePicker({
  open,
  onOpenChange,
  onActivated,
}: RuleTemplatePickerProps) {
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState<string>(CATEGORY_ALL);
  const [activatingId, setActivatingId] = useState<string | null>(null);

  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-rule-templates'],
    queryFn: () =>
      apiGet<PaginatedResponse<RuleTemplate>>(API_ENDPOINTS.CYBER_RULE_TEMPLATES, {
        per_page: 200,
      }),
    enabled: open,
    staleTime: 60_000,
  });

  const templates = envelope?.data ?? [];

  const categories = useMemo(() => {
    const unique = Array.from(new Set(templates.map((t) => t.category))).sort();
    return [CATEGORY_ALL, ...unique];
  }, [templates]);

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    return templates.filter((t) => {
      const matchesCategory =
        activeCategory === CATEGORY_ALL || t.category === activeCategory;
      const matchesSearch =
        !query ||
        t.name.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query) ||
        t.category.toLowerCase().includes(query);
      return matchesCategory && matchesSearch;
    });
  }, [templates, search, activeCategory]);

  const { mutate: activateTemplate } = useApiMutation<unknown, ActivatePayload>(
    'post',
    `${API_ENDPOINTS.CYBER_RULES}/activate`,
    {
      successMessage: 'Rule activated from template',
      invalidateKeys: ['cyber-rules'],
      onSuccess: () => {
        setActivatingId(null);
        onActivated();
      },
      onError: () => {
        setActivatingId(null);
      },
    },
  );

  function handleActivate(template: RuleTemplate) {
    setActivatingId(template.id);
    activateTemplate({ template_id: template.id });
  }

  function handleOpenChange(value: boolean) {
    if (!value) {
      setSearch('');
      setActiveCategory(CATEGORY_ALL);
      setActivatingId(null);
    }
    onOpenChange(value);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-3xl">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-primary" />
            Rule Template Library
          </DialogTitle>
        </DialogHeader>

        {/* Search */}
        <div className="border-b px-6 py-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search templates by name, description or category…"
              className="pl-9"
            />
          </div>
        </div>

        {/* Category chips */}
        {!isLoading && !error && categories.length > 1 && (
          <div className="flex flex-wrap gap-1.5 border-b px-6 py-3">
            {categories.map((cat) => (
              <button
                key={cat}
                onClick={() => setActiveCategory(cat)}
                className={`rounded-full px-3 py-0.5 text-xs font-medium transition-colors ${
                  activeCategory === cat
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <LoadingSkeleton key={i} variant="card" />
              ))}
            </div>
          ) : error ? (
            <ErrorState
              message="Failed to load rule templates"
              onRetry={() => refetch()}
            />
          ) : filtered.length === 0 ? (
            <EmptyState
              icon={ShieldCheck}
              title="No templates found"
              description={
                search
                  ? 'Try adjusting your search or category filter.'
                  : 'No rule templates are available.'
              }
            />
          ) : (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {filtered.map((template) => {
                const isActivating = activatingId === template.id;
                const mitreIds = template.mitre_technique_ids ?? [];
                const visibleMitre = mitreIds.slice(0, 3);
                const extraMitre = mitreIds.length - visibleMitre.length;

                return (
                  <div
                    key={template.id}
                    className="flex flex-col justify-between rounded-xl border bg-card p-4 transition-shadow hover:shadow-sm"
                  >
                    {/* Card header */}
                    <div className="space-y-2">
                      <div className="flex items-start justify-between gap-2">
                        <p className="font-semibold leading-snug">{template.name}</p>
                        <Badge variant="outline" className="shrink-0 text-xs capitalize">
                          {template.category}
                        </Badge>
                      </div>

                      <SeverityIndicator severity={template.severity} size="sm" showLabel />

                      <p className="line-clamp-2 text-xs text-muted-foreground">
                        {template.description}
                      </p>

                      {visibleMitre.length > 0 && (
                        <div className="flex flex-wrap gap-1">
                          {visibleMitre.map((tid) => (
                            <Badge
                              key={tid}
                              variant="outline"
                              className="font-mono text-xs"
                            >
                              {tid}
                            </Badge>
                          ))}
                          {extraMitre > 0 && (
                            <span className="self-center text-xs text-muted-foreground">
                              +{extraMitre} more
                            </span>
                          )}
                        </div>
                      )}
                    </div>

                    {/* Activate button */}
                    <div className="mt-4">
                      <Button
                        size="sm"
                        className="w-full"
                        disabled={isActivating || activatingId !== null}
                        onClick={() => handleActivate(template)}
                      >
                        {isActivating ? 'Activating…' : 'Activate'}
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
