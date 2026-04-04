"use client";

import { cn } from "@/lib/utils";
import type { AuditChange } from "@/types/audit";

interface ChangesDiffProps {
  changes: AuditChange[];
}

function formatValue(val: unknown): string {
  if (val === null || val === undefined) return "null";
  if (typeof val === "string") return `"${val}"`;
  if (typeof val === "object") return JSON.stringify(val, null, 2);
  return String(val);
}

export function ChangesDiff({ changes }: ChangesDiffProps) {
  if (changes.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No changes recorded.</p>
    );
  }

  return (
    <div className="rounded-md border bg-muted/20 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-xs font-mono">
          <thead>
            <tr className="border-b bg-muted/30">
              <th className="px-3 py-2 text-left font-semibold text-muted-foreground">
                Field
              </th>
              <th className="px-3 py-2 text-left font-semibold text-red-600 dark:text-red-400">
                Old Value
              </th>
              <th className="px-3 py-2 text-left font-semibold text-green-600 dark:text-green-400">
                New Value
              </th>
            </tr>
          </thead>
          <tbody>
            {changes.map((change) => (
              <tr key={change.field} className="border-b last:border-0">
                <td className="px-3 py-2 font-semibold whitespace-nowrap">
                  {change.field}
                </td>
                <td
                  className={cn(
                    "px-3 py-2 whitespace-pre-wrap break-all",
                    "bg-red-50/50 dark:bg-red-950/10 text-red-700 dark:text-red-400"
                  )}
                >
                  {formatValue(change.old_value)}
                </td>
                <td
                  className={cn(
                    "px-3 py-2 whitespace-pre-wrap break-all",
                    "bg-green-50/50 dark:bg-green-950/10 text-green-700 dark:text-green-400"
                  )}
                >
                  {formatValue(change.new_value)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
