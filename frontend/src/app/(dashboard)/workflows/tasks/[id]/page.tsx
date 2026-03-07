import type { Metadata } from 'next';
import { getTaskPageMetadata } from '@/lib/server-metadata';
import { TaskDetailPageClient } from './task-detail-page-client';

interface TaskDetailPageProps {
  params: {
    id: string;
  };
}

export async function generateMetadata(
  { params }: TaskDetailPageProps,
): Promise<Metadata> {
  return getTaskPageMetadata(params.id);
}

export default function TaskDetailPage() {
  return <TaskDetailPageClient />;
}
