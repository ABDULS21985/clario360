import type { Metadata } from 'next';
import { AdminTaskDetailClient } from './admin-task-detail-client';

export const metadata: Metadata = {
  title: 'Task Detail',
};

export default function AdminTaskDetailPage() {
  return <AdminTaskDetailClient />;
}
