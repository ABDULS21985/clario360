'use client';

import { useState, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import {
  Play,
  Search,
  Calendar,
  Globe,
  MousePointerClick,
  Webhook,
  GitBranch,
  Layers,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { StatusBadge } from '@/components/shared/status-badge';
import { StartWorkflowDialog } from '@/app/(dashboard)/admin/workflows/instances/components/start-workflow-dialog';
import { useWorkflowDefinitions } from '@/hooks/use-workflow-definitions';
import { workflowDefinitionStatusConfig } from '@/lib/status-configs';
import { titleCase } from '@/lib/format';
import type { WorkflowDefinition } from '@/types/models';

const TRIGGER_ICONS: Record<string, React.ElementType> = {
  manual: MousePointerClick,
  event: Globe,
  schedule: Calendar,
  webhook: Webhook,
};

const CATEGORIES = [
  'all',
  'approval',
  'onboarding',
  'review',
  'escalation',
  'notification',
  'data_pipeline',
  'compliance',
  'custom',
] as const;

export function DefinitionsBrowserClient() {
  const router = useRouter();
  const [search, setSearch] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('all');
  const [startOpen, setStartOpen] = useState(false);

  const { data, isLoading, isError, refetch } = useWorkflowDefinitions({
    status: 'active',
    per_page: 100,
  });

  const definitions = data?.data ?? [];

  const filtered = useMemo(() => {
    return definitions.filter((def) => {
      const matchesSearch =
        !search ||
        def.name.toLowerCase().includes(search.toLowerCase()) ||
        (def.description ?? '').toLowerCase().includes(search.toLowerCase());
      const matchesCategory =
        categoryFilter === 'all' || def.category === categoryFilter;
      return matchesSearch && matchesCategory;
    });
  }, [definitions, search, categoryFilter]);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Browse Workflows"
          description="Explore and start available workflow processes."
        />
        <LoadingSkeleton variant="card" count={6} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Browse Workflows"
          description="Explore and start available workflow processes."
        />
        <ErrorState
          message="Failed to load workflow definitions"
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Browse Workflows"
        description="Explore available workflow processes and start new instances."
        actions={
          <Button size="sm" onClick={() => setStartOpen(true)}>
            <Play className="mr-1.5 h-3.5 w-3.5" />
            Start Workflow
          </Button>
        }
      />

      {/* Filters */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search workflows..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
        <Select value={categoryFilter} onValueChange={setCategoryFilter}>
          <SelectTrigger className="w-full sm:w-44">
            <SelectValue placeholder="All categories" />
          </SelectTrigger>
          <SelectContent>
            {CATEGORIES.map((cat) => (
              <SelectItem key={cat} value={cat}>
                {cat === 'all' ? 'All Categories' : titleCase(cat)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Results count */}
      <p className="text-sm text-muted-foreground">
        {filtered.length} workflow{filtered.length !== 1 ? 's' : ''} available
      </p>

      {/* Grid */}
      {filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed p-12 text-center">
          <Layers className="mx-auto h-8 w-8 text-muted-foreground" />
          <p className="mt-3 text-sm text-muted-foreground">
            No workflows match your search.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {filtered.map((def) => (
            <DefinitionCard
              key={def.id}
              definition={def}
              onView={() => router.push(`/admin/workflows/definitions/${def.id}`)}
              onStart={() => setStartOpen(true)}
            />
          ))}
        </div>
      )}

      <StartWorkflowDialog open={startOpen} onOpenChange={setStartOpen} />
    </div>
  );
}

function DefinitionCard({
  definition,
  onView,
  onStart,
}: {
  definition: WorkflowDefinition;
  onView: () => void;
  onStart: () => void;
}) {
  const TriggerIcon =
    TRIGGER_ICONS[definition.trigger_config?.type ?? 'manual'] ?? Globe;
  const stepCount = definition.step_count ?? definition.steps?.length ?? 0;
  const instanceCount = definition.instance_count ?? 0;

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <CardTitle className="truncate text-base">{definition.name}</CardTitle>
            <div className="mt-1 flex flex-wrap items-center gap-1.5">
              <StatusBadge
                status={definition.status}
                config={workflowDefinitionStatusConfig}
              />
              {definition.category && (
                <Badge variant="secondary" className="text-xs">
                  {titleCase(definition.category)}
                </Badge>
              )}
            </div>
          </div>
          <TriggerIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
        </div>
      </CardHeader>

      <CardContent className="flex flex-1 flex-col gap-3">
        {definition.description ? (
          <p className="line-clamp-2 text-sm text-muted-foreground">
            {definition.description}
          </p>
        ) : (
          <p className="text-sm italic text-muted-foreground">No description.</p>
        )}

        <div className="flex items-center gap-4 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            {stepCount} step{stepCount !== 1 ? 's' : ''}
          </span>
          <span>v{definition.version}</span>
          <span>{instanceCount} run{instanceCount !== 1 ? 's' : ''}</span>
        </div>

        <div className="mt-auto flex gap-2 pt-1">
          <Button variant="outline" size="sm" className="flex-1" onClick={onView}>
            View Details
          </Button>
          <Button size="sm" className="flex-1" onClick={onStart}>
            <Play className="mr-1 h-3 w-3" />
            Start
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
