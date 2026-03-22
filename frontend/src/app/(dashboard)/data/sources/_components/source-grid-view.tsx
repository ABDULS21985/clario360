'use client';

import { type ConnectionTestResult, type DataSource } from '@/lib/data-suite';
import { SourceCard } from '@/app/(dashboard)/data/sources/_components/source-card';

interface TestState {
  loading: boolean;
  result?: ConnectionTestResult | null;
  error?: string | null;
}

interface SourceGridViewProps {
  sources: DataSource[];
  testStates: Record<string, TestState>;
  onTest: (source: DataSource) => void;
  onSync: (source: DataSource) => void;
  onEdit: (source: DataSource) => void;
  onDelete: (source: DataSource) => void;
  onToggleStatus?: (source: DataSource) => void;
}

export function SourceGridView({
  sources,
  testStates,
  onTest,
  onSync,
  onEdit,
  onDelete,
  onToggleStatus,
}: SourceGridViewProps) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3 md:grid-cols-2">
      {sources.map((source) => (
        <SourceCard
          key={source.id}
          source={source}
          testing={testStates[source.id]?.loading ?? false}
          testResult={testStates[source.id]?.result}
          testError={testStates[source.id]?.error}
          onTest={onTest}
          onSync={onSync}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggleStatus={onToggleStatus}
        />
      ))}
    </div>
  );
}
