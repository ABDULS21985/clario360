"use client";
import type { ColumnDef } from "@tanstack/react-table";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableRowActions } from "@/components/shared/data-table/data-table-row-actions";
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
        // Dynamically require RelativeTime to avoid breaking if not yet created
        try {
          // eslint-disable-next-line @typescript-eslint/no-require-imports
          const { RelativeTime } = require("@/components/shared/relative-time");
          return <RelativeTime date={value} />;
        } catch {
          // Fall through to static date rendering if component doesn't exist
        }
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
