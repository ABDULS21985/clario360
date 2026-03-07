import type { Metadata } from 'next';
import { getWorkflowInstancePageMetadata } from '@/lib/server-metadata';
import { WorkflowInstancePageClient } from './workflow-instance-page-client';

interface WorkflowInstancePageProps {
  params: {
    id: string;
  };
}

export async function generateMetadata(
  { params }: WorkflowInstancePageProps,
): Promise<Metadata> {
  return getWorkflowInstancePageMetadata(params.id);
}

export default function WorkflowInstancePage() {
  return <WorkflowInstancePageClient />;
}
