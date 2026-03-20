import type { AuditStatsParams } from "@/types/audit";
import type { AuditLog } from "@/types/models";
import type { FetchParams } from "@/types/table";

export const AUDIT_DEFAULT_RANGE_DAYS = 30;

const AUDIT_DEFAULT_SORT = "created_at";
const AUDIT_DEFAULT_ORDER = "desc";
const DAY_IN_MS = 24 * 60 * 60 * 1000;

const AUDIT_SEVERITIES = ["critical", "high", "warning", "info"] as const;

export type AuditSeverity = (typeof AUDIT_SEVERITIES)[number];

export interface AuditDateRange {
  date_from: string;
  date_to: string;
}

function readFilterValue(
  value: string | string[] | undefined
): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

function isAuditSeverity(
  value: string | null | undefined
): value is AuditSeverity {
  return value !== undefined && AUDIT_SEVERITIES.includes(value as AuditSeverity);
}

export function getDefaultAuditDateRange(now = new Date()): AuditDateRange {
  return {
    date_from: new Date(
      now.getTime() - AUDIT_DEFAULT_RANGE_DAYS * DAY_IN_MS
    ).toISOString(),
    date_to: now.toISOString(),
  };
}

export function normalizeAuditStatsParams(
  params?: AuditStatsParams,
  fallbackDateRange = getDefaultAuditDateRange()
): AuditStatsParams {
  return {
    ...params,
    date_from: params?.date_from ?? fallbackDateRange.date_from,
    date_to: params?.date_to ?? fallbackDateRange.date_to,
  };
}

export function buildAuditLogQueryParams(
  params: FetchParams,
  fallbackDateRange = getDefaultAuditDateRange()
) {
  const dateFrom = readFilterValue(params.filters?.date_from);
  const dateTo = readFilterValue(params.filters?.date_to);
  const service = readFilterValue(params.filters?.service);
  const severity = readFilterValue(params.filters?.severity);
  const userId = readFilterValue(params.filters?.user_id);

  return {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort ?? AUDIT_DEFAULT_SORT,
    order: params.order ?? AUDIT_DEFAULT_ORDER,
    search: params.search || undefined,
    date_from: dateFrom ?? fallbackDateRange.date_from,
    date_to: dateTo ?? fallbackDateRange.date_to,
    service,
    severity,
    user_id: userId,
  };
}

export function resolveAuditSeverity(
  action: string,
  severity?: AuditLog["severity"] | null
): AuditSeverity {
  if (isAuditSeverity(severity)) {
    return severity;
  }

  const normalizedAction = action.toLowerCase();

  if (
    normalizedAction.includes("delete") ||
    normalizedAction.includes("suspend")
  ) {
    return "high";
  }

  if (
    normalizedAction.includes("login.failed") ||
    normalizedAction.includes("unauthorized") ||
    normalizedAction.includes("denied")
  ) {
    return "warning";
  }

  return "info";
}

export const AUDIT_SEVERITY_FILTER_OPTIONS = [
  { label: "Critical", value: "critical" },
  { label: "High", value: "high" },
  { label: "Warning", value: "warning" },
  { label: "Info", value: "info" },
] satisfies Array<{ label: string; value: AuditSeverity }>;
