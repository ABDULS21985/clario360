import type { Metadata } from 'next';
import { InstanceDetailClient } from './components/instance-detail';

export const metadata: Metadata = {
  title: 'Workflow Instance',
};

export default function InstanceDetailPage() {
  return <InstanceDetailClient />;
}
