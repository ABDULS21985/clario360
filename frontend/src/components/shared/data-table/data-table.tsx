"use client";

import { useState, useMemo } from "react";
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type RowSelectionState,
  type VisibilityState,
} from "@tanstack/react-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DataTablePagination } from "./data-table-pagination";
import { DataTableToolbar } from "./data-table-toolbar";
import { DataTableSkeleton } from "./data-table-skeleton";
import { DataTableEmpty } from "./data-table-empty";
import { DataTableError } from "./data-table-error";
import { DataTableColumnHeader } from "./data-table-column-header";
import { actionsColumn } from "./columns/common-columns";
import { SearchInput } from "@/components/shared/forms/search-input";
import { cn } from "@/lib/utils";
import type { FilterConfig, BulkAction, RowAction, EmptyStateConfig } from "@/types/table";

interface DataTableProps<TData, TValue = unknown> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  totalRows: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  pageSizeOptions?: number[];
  sortColumn?: string;
  sortDirection?: "asc" | "desc";
  onSortChange: (column: string, direction: "asc" | "desc") => void;
  searchValue?: string;
  onSearchChange?: (value: string) => void;
  searchPlaceholder?: string;
  filters?: FilterConfig[];
  activeFilters?: Record<string, string | string[]>;
  onFilterChange?: (key: string, value: string | string[] | undefined) => void;
  onClearFilters?: () => void;
  enableSelection?: boolean;
  onSelectionChange?: (selectedIds: string[]) => void;
  getRowId?: (row: TData) => string;
  bulkActions?: BulkAction[];
  rowActions?: RowAction<TData>[] | ((row: TData) => RowAction<TData>[]);
  onRowClick?: (row: TData) => void;
  enableColumnToggle?: boolean;
  defaultHiddenColumns?: string[];
  enableExport?: boolean;
  onExport?: (format: "csv" | "json") => void;
  isLoading?: boolean;
  error?: string | null;
  onRetry?: () => void;
  emptyState?: EmptyStateConfig;
  stickyHeader?: boolean;
  compact?: boolean;
  striped?: boolean;
  className?: string;
  searchSlot?: React.ReactNode;
}

export function DataTable<TData, TValue = unknown>({
  columns,
  data,
  totalRows,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions,
  sortColumn,
  sortDirection,
  onSortChange,
  searchValue,
  onSearchChange,
  searchPlaceholder = "Search...",
  filters,
  activeFilters = {},
  onFilterChange,
  onClearFilters,
  enableSelection = false,
  onSelectionChange,
  getRowId,
  bulkActions,
  rowActions,
  onRowClick,
  enableColumnToggle = true,
  defaultHiddenColumns = [],
  enableExport = false,
  onExport,
  isLoading = false,
  error = null,
  onRetry,
  emptyState,
  stickyHeader = true,
  compact = false,
  striped = false,
  className,
  searchSlot,
}: DataTableProps<TData, TValue>) {
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(
    Object.fromEntries(defaultHiddenColumns.map((col) => [col, false]))
  );

  const resolvedColumns = useMemo<ColumnDef<TData, TValue>[]>(() => {
    if (!rowActions) return columns;
    return [...columns, actionsColumn(rowActions) as ColumnDef<TData, TValue>];
  }, [columns, rowActions]);

  const defaultGetRowId = (row: TData): string =>
    (row as TData & { id?: string }).id ?? String(Math.random());

  const table = useReactTable({
    data,
    columns: resolvedColumns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: getRowId ?? defaultGetRowId,
    state: { rowSelection, columnVisibility },
    onRowSelectionChange: (updater) => {
      const next =
        typeof updater === "function" ? updater(rowSelection) : updater;
      setRowSelection(next);
      if (onSelectionChange) {
        onSelectionChange(Object.keys(next).filter((k) => next[k]));
      }
    },
    onColumnVisibilityChange: setColumnVisibility,
    enableRowSelection: enableSelection,
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
    pageCount: Math.ceil(totalRows / pageSize),
  });

  const selectedIds = useMemo(
    () => Object.keys(rowSelection).filter((k) => rowSelection[k]),
    [rowSelection]
  );

  const hasActiveFilters = Object.values(activeFilters).some(
    (v) =>
      v !== undefined && v !== "" && !(Array.isArray(v) && v.length === 0)
  );
  const resolvedSearchSlot =
    searchSlot ?? (onSearchChange ? (
      <SearchInput
        value={searchValue ?? ""}
        onChange={onSearchChange}
        placeholder={searchPlaceholder}
        loading={isLoading}
      />
    ) : undefined);

  const cellPadding = compact ? "px-3 py-1.5" : "px-4 py-3";

  return (
    <div className={cn("w-full space-y-3", className)}>
      <DataTableToolbar
        searchSlot={resolvedSearchSlot}
        filters={filters}
        activeFilters={activeFilters}
        onFilterChange={onFilterChange}
        onClearFilters={onClearFilters}
        columns={table.getAllColumns()}
        columnVisibility={Object.fromEntries(
          table.getAllColumns().map((col) => [col.id, col.getIsVisible()])
        )}
        onColumnVisibilityChange={(vis) => setColumnVisibility(vis)}
        enableColumnToggle={enableColumnToggle}
        enableExport={enableExport}
        onExport={onExport}
        selectedCount={selectedIds.length}
        bulkActions={bulkActions}
        getSelectedIds={() => selectedIds}
      />

      <div className="rounded-md border border-border overflow-hidden">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader
              className={cn(stickyHeader && "sticky top-0 z-10 bg-background")}
            >
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow
                  key={headerGroup.id}
                  className="hover:bg-transparent"
                >
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={cn("whitespace-nowrap", cellPadding)}
                      style={{
                        width:
                          header.getSize() !== 150
                            ? header.getSize()
                            : undefined,
                      }}
                    >
                      {header.isPlaceholder ? null : header.column.getCanSort() ? (
                        <DataTableColumnHeader
                          column={header.column}
                          title={String(
                            header.column.columnDef.header ?? header.id
                          )}
                          onSortChange={onSortChange}
                          sortColumn={sortColumn}
                          sortDirection={sortDirection}
                        />
                      ) : (
                        flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )
                      )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody aria-busy={isLoading}>
              {isLoading ? (
                <TableRow>
                  <TableCell
                    colSpan={resolvedColumns.length}
                    className="p-0"
                  >
                    <DataTableSkeleton
                      columns={columns.length}
                      rows={pageSize > 10 ? 10 : pageSize}
                      hasCheckbox={enableSelection}
                      hasActions={!!rowActions}
                    />
                  </TableCell>
                </TableRow>
              ) : error ? (
                <TableRow>
                  <TableCell colSpan={resolvedColumns.length}>
                    <DataTableError error={error} onRetry={onRetry} />
                  </TableCell>
                </TableRow>
              ) : table.getRowModel().rows.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={resolvedColumns.length}>
                    {emptyState ? (
                      <DataTableEmpty
                        icon={emptyState.icon}
                        title={emptyState.title}
                        description={emptyState.description}
                        action={emptyState.action}
                        hasActiveFilters={hasActiveFilters}
                      />
                    ) : (
                      <DataTableEmpty hasActiveFilters={hasActiveFilters} />
                    )}
                  </TableCell>
                </TableRow>
              ) : (
                table.getRowModel().rows.map((row, rowIndex) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() ? "selected" : undefined}
                    aria-selected={row.getIsSelected()}
                    onClick={
                      onRowClick ? () => onRowClick(row.original) : undefined
                    }
                    className={cn(
                      onRowClick && "cursor-pointer",
                      striped && rowIndex % 2 === 1 && "bg-muted/20",
                      "hover:bg-muted/40 transition-colors"
                    )}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cn("whitespace-nowrap", cellPadding)}
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <DataTablePagination
        page={page}
        pageSize={pageSize}
        totalRows={totalRows}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        pageSizeOptions={pageSizeOptions}
      />
    </div>
  );
}
