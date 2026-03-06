'use client';

import { useMemo } from 'react';
import { cn } from '@/lib/utils';

interface JsonDiffViewerProps {
  oldValue: Record<string, unknown> | null | undefined;
  newValue: Record<string, unknown> | null | undefined;
}

type DiffKind = 'added' | 'removed' | 'changed' | 'unchanged';

interface DiffLine {
  key: string;
  kind: DiffKind;
  oldVal?: string;
  newVal?: string;
  indent: number;
}

function stringify(val: unknown): string {
  if (val === null) return 'null';
  if (typeof val === 'string') return `"${val}"`;
  if (typeof val === 'object') return JSON.stringify(val, null, 2);
  return String(val);
}

function computeDiff(
  oldObj: Record<string, unknown> | null,
  newObj: Record<string, unknown> | null,
  indent = 0
): DiffLine[] {
  if (!oldObj && !newObj) return [];

  if (!oldObj) {
    return Object.entries(newObj!).map(([key, val]) => ({
      key,
      kind: 'added',
      newVal: stringify(val),
      indent,
    }));
  }

  if (!newObj) {
    return Object.entries(oldObj).map(([key, val]) => ({
      key,
      kind: 'removed',
      oldVal: stringify(val),
      indent,
    }));
  }

  const allKeys = new Set([...Object.keys(oldObj), ...Object.keys(newObj)]);
  const lines: DiffLine[] = [];

  allKeys.forEach((key) => {
    const hasOld = key in oldObj;
    const hasNew = key in newObj;

    if (!hasOld) {
      lines.push({ key, kind: 'added', newVal: stringify(newObj[key]), indent });
    } else if (!hasNew) {
      lines.push({ key, kind: 'removed', oldVal: stringify(oldObj[key]), indent });
    } else {
      const oldStr = stringify(oldObj[key]);
      const newStr = stringify(newObj[key]);
      if (oldStr === newStr) {
        lines.push({ key, kind: 'unchanged', oldVal: oldStr, indent });
      } else {
        lines.push({ key, kind: 'changed', oldVal: oldStr, newVal: newStr, indent });
      }
    }
  });

  return lines;
}

const kindStyles: Record<DiffKind, string> = {
  added: 'bg-green-50 dark:bg-green-950/20 text-green-700 dark:text-green-400',
  removed: 'bg-red-50 dark:bg-red-950/20 text-red-700 dark:text-red-400',
  changed: 'bg-yellow-50 dark:bg-yellow-950/20 text-yellow-700 dark:text-yellow-400',
  unchanged: 'text-muted-foreground',
};

const kindPrefix: Record<DiffKind, string> = {
  added: '+',
  removed: '-',
  changed: '~',
  unchanged: ' ',
};

export function JsonDiffViewer({ oldValue, newValue }: JsonDiffViewerProps) {
  const lines = useMemo(
    () => computeDiff(oldValue ?? null, newValue ?? null),
    [oldValue, newValue]
  );

  if (!oldValue && !newValue) {
    return (
      <p className="text-sm text-muted-foreground">No change data available.</p>
    );
  }

  if (lines.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No differences detected.</p>
    );
  }

  return (
    <div className="rounded-md border bg-muted/20 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-xs font-mono">
          <tbody>
            {lines.map((line, i) => (
              <tr
                key={i}
                className={cn('border-b last:border-0', kindStyles[line.kind])}
              >
                <td className="px-2 py-0.5 select-none text-muted-foreground w-4 text-center">
                  {kindPrefix[line.kind]}
                </td>
                <td
                  className="px-2 py-0.5 font-semibold whitespace-nowrap"
                  style={{ paddingLeft: `${8 + line.indent * 16}px` }}
                >
                  {line.key}:
                </td>
                {line.kind === 'changed' ? (
                  <td className="px-2 py-0.5">
                    <span className="line-through opacity-60 mr-2">{line.oldVal}</span>
                    <span>→ {line.newVal}</span>
                  </td>
                ) : line.kind === 'removed' ? (
                  <td className="px-2 py-0.5">{line.oldVal}</td>
                ) : (
                  <td className="px-2 py-0.5">{line.newVal ?? line.oldVal}</td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
