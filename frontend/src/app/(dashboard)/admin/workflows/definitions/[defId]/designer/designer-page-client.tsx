'use client';

import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import {
  useWorkflowDefinition,
  useUpdateWorkflowDefinition,
  usePublishWorkflowDefinition,
} from '@/hooks/use-workflow-definitions';
import { WorkflowCanvas } from './components/workflow-canvas';
import type { WorkflowStep } from '@/types/models';

export function DesignerPageClient() {
  const params = useParams();
  const router = useRouter();
  const defId = (params?.defId as string | undefined) ?? '';

  const { data: definition, isLoading, isError, refetch } = useWorkflowDefinition(defId);
  const updateMutation = useUpdateWorkflowDefinition();
  const publishMutation = usePublishWorkflowDefinition();

  if (isLoading) {
    return (
      <div className="h-[calc(100vh-4rem)]">
        <LoadingSkeleton variant="card" count={1} />
      </div>
    );
  }

  if (isError || !definition) {
    return (
      <ErrorState
        message="Failed to load workflow definition"
        onRetry={() => refetch()}
      />
    );
  }

  const readOnly = definition.status !== 'draft';

  function handleSave(steps: WorkflowStep[]) {
    updateMutation.mutate({
      defId: definition!.id,
      data: { steps },
    });
  }

  function handlePublish() {
    publishMutation.mutate(definition!.id);
  }

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)]">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-2 border-b bg-background">
        <button
          onClick={() => router.push(`/admin/workflows/definitions/${defId}`)}
          className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
          type="button"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>
        <div className="min-w-0">
          <h1 className="text-sm font-semibold truncate">{definition.name}</h1>
          <p className="text-xs text-muted-foreground">
            v{definition.version} &middot;{' '}
            {readOnly ? 'Read-only' : 'Editing'}
          </p>
        </div>
      </div>

      {/* Canvas */}
      <div className="flex-1 min-h-0">
        <WorkflowCanvas
          definition={definition}
          readOnly={readOnly}
          isSaving={updateMutation.isPending}
          isPublishing={publishMutation.isPending}
          onSave={handleSave}
          onPublish={handlePublish}
        />
      </div>
    </div>
  );
}
