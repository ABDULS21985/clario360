import type { Metadata } from 'next';
import { WorkflowTasksPageClient } from './tasks-page-client';

export const metadata: Metadata = {
  title: 'My Tasks',
};

export default function WorkflowTasksPage() {
  return <WorkflowTasksPageClient />;
}
