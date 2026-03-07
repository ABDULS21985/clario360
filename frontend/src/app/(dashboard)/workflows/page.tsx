import type { Metadata } from 'next';
import { WorkflowsPageClient } from './workflows-page-client';

export const metadata: Metadata = {
  title: 'Workflows',
};

export default function WorkflowsPage() {
  return <WorkflowsPageClient />;
}
