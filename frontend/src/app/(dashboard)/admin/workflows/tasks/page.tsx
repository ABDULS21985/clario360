import type { Metadata } from 'next';
import { AdminTaskList } from './components/admin-task-list';

export const metadata: Metadata = {
  title: 'Workflow Tasks',
};

export default function AdminWorkflowTasksPage() {
  return <AdminTaskList />;
}
