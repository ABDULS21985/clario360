// DataTable-specific types used across the platform
import type { LucideIcon } from "lucide-react";

export interface FilterConfig {
  key: string;
  label: string;
  type: "text" | "select" | "multi-select" | "date-range" | "boolean" | "range";
  options?: Array<{ label: string; value: string }>;
  placeholder?: string;
  min?: number;
  max?: number;
  step?: number;
  valueSuffix?: string;
}

export interface BulkAction {
  label: string;
  icon?: LucideIcon;
  variant?: "default" | "destructive";
  onClick: (selectedIds: string[]) => Promise<void>;
  confirmMessage?: string;
}

export interface RowAction<TData> {
  label: string;
  icon?: LucideIcon;
  variant?: "default" | "destructive";
  onClick: (row: TData) => void;
  hidden?: (row: TData) => boolean;
  disabled?: (row: TData) => boolean;
}

export interface EmptyStateConfig {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: { label: string; onClick: () => void; icon?: LucideIcon };
}

export interface FetchParams {
  page: number;
  per_page: number;
  sort?: string;
  order?: "asc" | "desc";
  search?: string;
  filters?: Record<string, string | string[]>;
}

export interface DataTableControlledProps<TData> {
  data: TData[];
  totalRows: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  sortColumn?: string;
  sortDirection?: "asc" | "desc";
  onSortChange: (column: string, direction: "asc" | "desc") => void;
  searchValue?: string;
  onSearchChange?: (value: string) => void;
  activeFilters?: Record<string, string | string[]>;
  onFilterChange?: (key: string, value: string | string[] | undefined) => void;
  onClearFilters?: () => void;
  isLoading?: boolean;
  error?: string | null;
  onRetry?: () => void;
}
