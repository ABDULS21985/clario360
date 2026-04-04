"use client";
import type { ColumnDef } from "@tanstack/react-table";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableRowActions } from "@/components/shared/data-table/data-table-row-actions";
import { StatusBadge } from "@/components/shared/status-badge";
import { SeverityIndicator } from "@/components/shared/severity-indicator";
import { UserAvatar } from "@/components/shared/user-avatar";
import { CopyButton } from "@/components/shared/copy-button";
import { RelativeTime } from "@/components/shared/relative-time";
import type { StatusConfig } from "@/lib/status-configs";
import type { Severity } from "@/components/shared/severity-indicator";
import type { RowAction } from "@/types/table";

export function selectColumn<TData>(): ColumnDef<TData> {
  return {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        onClick={(e) => e.stopPropagation()}
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        onClick={(e) => e.stopPropagation()}
      />
    ),
    enableSorting: false,
    enableHiding: false,
    size: 40,
  };
}

export function dateColumn<TData>(
  accessor: string,
  header: string,
  options?: { relative?: boolean }
): ColumnDef<TData> {
  return {
    id: accessor,
    accessorKey: accessor as keyof TData & string,
    header,
    cell: ({ getValue }) => {
      const value = getValue() as string | null;
      if (!value) return <span className="text-muted-foreground">&mdash;</span>;
      if (options?.relative !== false) {
        return <RelativeTime date={value} />;
      }
      return <time dateTime={value}>{new Date(value).toLocaleDateString()}</time>;
    },
    enableSorting: true,
  };
}

export function actionsColumn<TData>(
  actions: RowAction<TData>[] | ((row: TData) => RowAction<TData>[])
): ColumnDef<TData> {
  return {
    id: "actions",
    header: "",
    cell: ({ row }) => (
      <DataTableRowActions row={row.original} actions={actions} />
    ),
    enableSorting: false,
    enableHiding: false,
    size: 50,
  };
}

export function statusColumn<TData>(
  accessor: string,
  header: string,
  config: StatusConfig
): ColumnDef<TData> {
  return {
    id: accessor,
    accessorKey: accessor as keyof TData & string,
    header,
    cell: ({ getValue }) => {
      const value = getValue() as string | null;
      if (!value) return <span className="text-muted-foreground">&mdash;</span>;
      return <StatusBadge status={value} config={config} />;
    },
    enableSorting: true,
  };
}

export function severityColumn<TData>(
  accessor: string,
  header = "Severity"
): ColumnDef<TData> {
  return {
    id: accessor,
    accessorKey: accessor as keyof TData & string,
    header,
    cell: ({ getValue }) => {
      const value = getValue() as Severity | null;
      if (!value) return <span className="text-muted-foreground">&mdash;</span>;
      return <SeverityIndicator severity={value} size="sm" />;
    },
    enableSorting: true,
  };
}

export function userColumn<TData>(
  accessor: string,
  header = "User"
): ColumnDef<TData> {
  return {
    id: accessor,
    accessorKey: accessor as keyof TData & string,
    header,
    cell: ({ getValue }) => {
      const value = getValue() as { first_name: string; last_name: string; email?: string } | null;
      if (!value) return <span className="text-muted-foreground">&mdash;</span>;
      return (
        <div className="flex items-center gap-2">
          <UserAvatar user={value} size="sm" showTooltip />
          <span className="text-sm">
            {value.first_name} {value.last_name}
          </span>
        </div>
      );
    },
    enableSorting: false,
  };
}

export function idColumn<TData>(
  accessor: string,
  header = "ID"
): ColumnDef<TData> {
  return {
    id: accessor,
    accessorKey: accessor as keyof TData & string,
    header,
    cell: ({ getValue }) => {
      const value = getValue() as string | null;
      if (!value) return <span className="text-muted-foreground">&mdash;</span>;
      const short = value.slice(0, 8);
      return (
        <div className="flex items-center gap-1">
          <code className="text-xs font-mono text-muted-foreground">{short}…</code>
          <CopyButton value={value} label="Copy ID" />
        </div>
      );
    },
    enableSorting: false,
    size: 120,
  };
}
