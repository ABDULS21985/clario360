import type { Metadata } from 'next';
import { InstancesList } from './components/instances-list';

export const metadata: Metadata = {
  title: 'Workflow Instances',
};

export default function AdminWorkflowInstancesPage() {
  return <InstancesList />;
}
