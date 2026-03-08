import { apiGet } from '@/lib/api';
import type { PaginatedResponse, PaginationMeta } from '@/types/api';
import type { FetchParams } from '@/types/table';

export interface SuiteEnvelope<T> {
  data: T;
}

interface SuitePaginatedEnvelope<T> {
  data: T[];
  meta: PaginationMeta;
}

export function buildSuiteQueryParams(
  params: FetchParams,
  extra?: Record<string, unknown>,
): Record<string, unknown> {
  const query: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
    ...extra,
  };

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    if (value === undefined || value === '' || (Array.isArray(value) && value.length === 0)) {
      continue;
    }
    query[key] = Array.isArray(value) ? value.join(',') : value;
  }

  return query;
}

export async function fetchSuitePaginated<T>(
  url: string,
  params: FetchParams,
  extra?: Record<string, unknown>,
): Promise<PaginatedResponse<T>> {
  const response = await apiGet<SuitePaginatedEnvelope<T>>(url, buildSuiteQueryParams(params, extra));
  return {
    data: response.data,
    meta: response.meta,
  };
}

export async function fetchSuiteData<T>(
  url: string,
  params?: Record<string, unknown>,
): Promise<T> {
  const response = await apiGet<SuiteEnvelope<T>>(url, params);
  return response.data;
}
