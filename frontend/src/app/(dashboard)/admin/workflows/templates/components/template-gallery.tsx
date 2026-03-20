'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import { SearchInput } from '@/components/shared/forms/search-input';
import { TemplateCard } from './template-card';
import { useWorkflowTemplates } from '@/hooks/use-workflow-templates';
import { useCreateDefinitionFromTemplate } from '@/hooks/use-workflow-templates';
import type { WorkflowTemplate, WorkflowCategory } from '@/types/models';

const categories: { label: string; value: WorkflowCategory | 'all' }[] = [
  { label: 'All', value: 'all' },
  { label: 'Approval', value: 'approval' },
  { label: 'Onboarding', value: 'onboarding' },
  { label: 'Review', value: 'review' },
  { label: 'Escalation', value: 'escalation' },
  { label: 'Notification', value: 'notification' },
  { label: 'Data Pipeline', value: 'data_pipeline' },
  { label: 'Compliance', value: 'compliance' },
  { label: 'Custom', value: 'custom' },
];

export function TemplateGallery() {
  const router = useRouter();
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState<string>('all');
  const [useDialogTemplate, setUseDialogTemplate] =
    useState<WorkflowTemplate | null>(null);
  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');

  const { data, isLoading, isError, refetch } = useWorkflowTemplates({
    per_page: 100,
    ...(category !== 'all' ? { category } : {}),
    ...(search ? { search } : {}),
  });

  const createFromTemplate = useCreateDefinitionFromTemplate();
  const templates = data?.data ?? [];

  function handleUseTemplate(template: WorkflowTemplate) {
    setUseDialogTemplate(template);
    setNewName(`${template.name} (Copy)`);
    setNewDescription(template.description);
  }

  function handleCreate() {
    if (!useDialogTemplate || !newName.trim()) return;
    createFromTemplate.mutate(
      {
        template_id: useDialogTemplate.id,
        name: newName,
        description: newDescription || undefined,
      },
      {
        onSuccess: (def) => {
          setUseDialogTemplate(null);
          router.push(`/admin/workflows/definitions/${def.id}/designer`);
        },
      },
    );
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Workflow Templates"
          description="Browse pre-built workflow templates"
        />
        <ErrorState
          message="Failed to load templates"
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Workflow Templates"
        description="Browse pre-built workflow templates to get started quickly."
      />

      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-3">
        <SearchInput
          value={search}
          onChange={setSearch}
          placeholder="Search templates..."
        />
        <Select value={category} onValueChange={setCategory}>
          <SelectTrigger className="w-40 h-8 text-sm">
            <SelectValue placeholder="Category" />
          </SelectTrigger>
          <SelectContent>
            {categories.map((c) => (
              <SelectItem key={c.value} value={c.value}>
                {c.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <LoadingSkeleton variant="card" count={6} />
        </div>
      ) : templates.length === 0 ? (
        <div className="text-center py-12 text-muted-foreground">
          <p className="text-sm font-medium">No templates found</p>
          <p className="text-xs mt-1">Try adjusting your search or filters.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {templates.map((template) => (
            <TemplateCard
              key={template.id}
              template={template}
              onClick={handleUseTemplate}
            />
          ))}
        </div>
      )}

      {/* Use template dialog */}
      <Dialog
        open={!!useDialogTemplate}
        onOpenChange={(open) => {
          if (!open) setUseDialogTemplate(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Use Template</DialogTitle>
            <DialogDescription>
              Create a new workflow definition from &ldquo;
              {useDialogTemplate?.name}&rdquo;.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="def-name" className="text-xs">
                Name <span className="text-red-500">*</span>
              </Label>
              <Input
                id="def-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="h-8 text-sm"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="def-desc" className="text-xs">
                Description
              </Label>
              <Input
                id="def-desc"
                value={newDescription}
                onChange={(e) => setNewDescription(e.target.value)}
                className="h-8 text-sm"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setUseDialogTemplate(null)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!newName.trim() || createFromTemplate.isPending}
            >
              {createFromTemplate.isPending && (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              )}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
