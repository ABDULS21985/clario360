import type { Metadata } from 'next';
import { PageHeader } from '@/components/common/page-header';
import { WorkflowKpiCards } from './_components/workflow-kpi-cards';
import { InstanceStatusChart } from './_components/instance-status-chart';
import { TaskWorkloadTable } from './_components/task-workload-table';

export const metadata: Metadata = {
  title: 'Workflow Analytics',
};

export default function WorkflowAnalyticsPage() {
  return (
    <div className="space-y-8">
      <PageHeader
        title="Workflow Analytics"
        description="Monitor workflow execution health, task workload, and definition usage."
      />

      <WorkflowKpiCards />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <InstanceStatusChart />
        <TaskWorkloadTable />
      </div>
    </div>
  );
}
