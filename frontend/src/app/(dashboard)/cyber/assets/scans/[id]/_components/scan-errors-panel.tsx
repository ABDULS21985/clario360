'use client';

import { AlertTriangle } from 'lucide-react';

interface Props {
  error: string;
}

export function ScanErrorsPanel({ error }: Props) {
  return (
    <div className="rounded-xl border border-destructive/30 bg-destructive/5 p-5">
      <div className="mb-3 flex items-center gap-2">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-3.5 w-3.5 text-destructive" aria-hidden />
        </span>
        <h3 className="text-sm font-semibold text-destructive">Scan Error</h3>
      </div>
      <pre className="overflow-x-auto whitespace-pre-wrap rounded-md bg-destructive/10 px-3 py-2 font-mono text-xs text-destructive">
        {error}
      </pre>
    </div>
  );
}
