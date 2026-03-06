"use client";

import { useCallback, useMemo } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useDebounce } from "./use-debounce";
import type { FetchParams, DataTableControlledProps } from "@/types/table";
import type { PaginatedResponse } from "@/types/api";

interface UseDataTableOptions<TData> {
  fetchFn: (params: FetchParams) => Promise<PaginatedResponse<TData>>;
  queryKey: string; // unique key for React Query caching
  defaultPageSize?: number;
  defaultSort?: { column: string; direction: "asc" | "desc" };
}

interface UseDataTableReturn<TData> {
  data: TData[];
  totalRows: number;
  isLoading: boolean;
  error: string | null;
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  sortColumn: string | undefined;
  sortDirection: "asc" | "desc";
  setSort: (column: string, direction: "asc" | "desc") => void;
  searchValue: string;
  setSearch: (value: string) => void;
  activeFilters: Record<string, string | string[]>;
  setFilter: (key: string, value: string | string[] | undefined) => void;
  clearFilters: () => void;
  refetch: () => void;
  tableProps: DataTableControlledProps<TData>;
}

export function useDataTable<TData>(
  options: UseDataTableOptions<TData>
): UseDataTableReturn<TData> {
  const { fetchFn, queryKey, defaultPageSize = 25, defaultSort } = options;
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // Read state from URL
  const page = parseInt(searchParams.get("page") ?? "1", 10);
  const pageSize = parseInt(
    searchParams.get("per_page") ?? String(defaultPageSize),
    10
  );
  const sortColumn = searchParams.get("sort") ?? defaultSort?.column;
  const sortDirection = (searchParams.get("order") ??
    defaultSort?.direction ??
    "desc") as "asc" | "desc";
  const rawSearch = searchParams.get("search") ?? "";
  const debouncedSearch = useDebounce(rawSearch, 300);

  // Collect all other params as filters
  const activeFilters = useMemo(() => {
    const reserved = new Set(["page", "per_page", "sort", "order", "search"]);
    const filters: Record<string, string | string[]> = {};
    searchParams.forEach((value, key) => {
      if (!reserved.has(key)) {
        const existing = filters[key];
        if (existing) {
          filters[key] = Array.isArray(existing)
            ? [...existing, value]
            : [existing, value];
        } else {
          filters[key] = value;
        }
      }
    });
    return filters;
  }, [searchParams]);

  const updateParams = useCallback(
    (updates: Record<string, string | string[] | undefined>) => {
      const params = new URLSearchParams(searchParams.toString());
      for (const [key, value] of Object.entries(updates)) {
        if (
          value === undefined ||
          value === "" ||
          (Array.isArray(value) && value.length === 0)
        ) {
          params.delete(key);
        } else if (Array.isArray(value)) {
          params.delete(key);
          value.forEach((v) => params.append(key, v));
        } else {
          params.set(key, value);
        }
      }
      router.push(`${pathname}?${params.toString()}`);
    },
    [router, pathname, searchParams]
  );

  const setPage = useCallback(
    (p: number) => updateParams({ page: String(p) }),
    [updateParams]
  );
  const setPageSize = useCallback(
    (s: number) => updateParams({ per_page: String(s), page: "1" }),
    [updateParams]
  );
  const setSort = useCallback(
    (column: string, direction: "asc" | "desc") =>
      updateParams({ sort: column, order: direction, page: "1" }),
    [updateParams]
  );
  const setSearch = useCallback(
    (value: string) => updateParams({ search: value || undefined, page: "1" }),
    [updateParams]
  );
  const setFilter = useCallback(
    (key: string, value: string | string[] | undefined) =>
      updateParams({ [key]: value, page: "1" }),
    [updateParams]
  );
  const clearFilters = useCallback(() => {
    const reserved = ["page", "per_page", "sort", "order", "search"];
    const params = new URLSearchParams();
    reserved.forEach((k) => {
      const v = searchParams.get(k);
      if (v) params.set(k, v);
    });
    params.set("page", "1");
    router.push(`${pathname}?${params.toString()}`);
  }, [router, pathname, searchParams]);

  const fetchParams: FetchParams = useMemo(
    () => ({
      page,
      per_page: pageSize,
      sort: sortColumn ?? undefined,
      order: sortDirection,
      search: debouncedSearch || undefined,
      filters:
        Object.keys(activeFilters).length > 0 ? activeFilters : undefined,
    }),
    [page, pageSize, sortColumn, sortDirection, debouncedSearch, activeFilters]
  );

  const {
    data: queryData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: [queryKey, fetchParams],
    queryFn: () => fetchFn(fetchParams),
    placeholderData: (prev) => prev, // keepPreviousData equivalent in RQ v5
    staleTime: 30_000,
  });

  const errorMessage = error
    ? error instanceof Error
      ? error.message
      : "An error occurred"
    : null;

  const tableProps: DataTableControlledProps<TData> = {
    data: queryData?.data ?? [],
    totalRows: queryData?.meta.total ?? 0,
    page,
    pageSize,
    onPageChange: setPage,
    onPageSizeChange: setPageSize,
    sortColumn: sortColumn ?? undefined,
    sortDirection,
    onSortChange: setSort,
    searchValue: rawSearch,
    onSearchChange: setSearch,
    activeFilters,
    onFilterChange: setFilter,
    onClearFilters: clearFilters,
    isLoading,
    error: errorMessage,
    onRetry: () => refetch(),
  };

  return {
    data: queryData?.data ?? [],
    totalRows: queryData?.meta.total ?? 0,
    isLoading,
    error: errorMessage,
    page,
    pageSize,
    setPage,
    setPageSize,
    sortColumn: sortColumn ?? undefined,
    sortDirection,
    setSort,
    searchValue: rawSearch,
    setSearch,
    activeFilters,
    setFilter,
    clearFilters,
    refetch,
    tableProps,
  };
}
