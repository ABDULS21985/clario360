'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, Loader2, Rocket } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { titleCase } from '@/lib/format';
import { formatStepType } from '@/lib/workflow-utils';
import { useWorkflowTemplate } from '@/hooks/use-workflow-templates';
import { useCreateDefinitionFromTemplate } from '@/hooks/use-workflow-templates';
import type { WorkflowTemplate } from '@/types/models';

export function TemplateDetailClient() {
  const params = useParams();
  const router = useRouter();
  const templateId = (params?.templateId as string | undefined) ?? '';
  const [showUseDialog, setShowUseDialog] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDescription, setNewDescription] = useState('');

  const { data: template, isLoading, isError, refetch } =
    useWorkflowTemplate(templateId);
  const createFromTemplate = useCreateDefinitionFromTemplate();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton variant="card" count={3} />
      </div>
    );
  }

  if (isError || !template) {
    return (
      <ErrorState
        message="Failed to load template"
        onRetry={() => refetch()}
      />
    );
  }

  function handleUse() {
    setNewName(`${template!.name} (Copy)`);
    setNewDescription(template!.description);
    setShowUseDialog(true);
  }

  function handleCreate() {
    if (!newName.trim()) return;
    createFromTemplate.mutate(
      {
        template_id: templateId,
        name: newName,
        description: newDescription || undefined,
      },
      {
        onSuccess: (def) => {
          setShowUseDialog(false);
          router.push(`/admin/workflows/definitions/${def.id}/designer`);
        },
      },
    );
  }

  return (
    <div className="space-y-6">
      <button
        onClick={() => router.push('/admin/workflows/templates')}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        type="button"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Templates
      </button>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">{template.name}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-2">
            <Badge variant="secondary">{titleCase(template.category)}</Badge>
            {(template.tags ?? []).map((tag) => (
              <Badge key={tag} variant="outline" className="text-xs">
                {tag}
              </Badge>
            ))}
          </div>
          {template.description && (
            <p className="mt-2 text-sm text-muted-foreground max-w-xl">
              {template.description}
            </p>
          )}
        </div>
        <Button size="sm" onClick={handleUse}>
          <Rocket className="mr-1 h-3.5 w-3.5" />
          Use This Template
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Steps */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Steps ({template.steps?.length ?? 0})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-1.5">
              {(template.steps ?? []).map((step, idx) => (
                <div
                  key={step.id}
                  className="flex items-center gap-2 text-sm"
                >
                  <span className="text-xs text-muted-foreground w-4">
                    {idx + 1}.
                  </span>
                  <span className="font-medium">{step.name}</span>
                  <Badge variant="outline" className="text-[10px]">
                    {formatStepType(step.type)}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Variables */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">
              Variables ({template.variables?.length ?? 0})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {(template.variables?.length ?? 0) === 0 ? (
              <p className="text-xs text-muted-foreground">
                No variables defined.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="text-xs">Name</TableHead>
                    <TableHead className="text-xs">Type</TableHead>
                    <TableHead className="text-xs">Required</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {template.variables!.map((v) => (
                    <TableRow key={v.name}>
                      <TableCell className="font-mono text-xs py-1.5">
                        {v.name}
                      </TableCell>
                      <TableCell className="text-xs py-1.5">
                        <Badge variant="outline" className="text-[10px]">
                          {v.type}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs py-1.5">
                        {v.required ? 'Yes' : 'No'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Use template dialog */}
      <Dialog open={showUseDialog} onOpenChange={setShowUseDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Use Template</DialogTitle>
            <DialogDescription>
              Create a new workflow definition from this template.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="tpl-name" className="text-xs">
                Name <span className="text-red-500">*</span>
              </Label>
              <Input
                id="tpl-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="h-8 text-sm"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="tpl-desc" className="text-xs">
                Description
              </Label>
              <Input
                id="tpl-desc"
                value={newDescription}
                onChange={(e) => setNewDescription(e.target.value)}
                className="h-8 text-sm"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowUseDialog(false)}
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
