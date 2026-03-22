import type { AuditLog } from "./models";

// ── Statistics types ─────────────────────────────────────────────────────────

export interface AuditGroupStat {
  key: string;
  count: number;
  percentage: number;
}

export interface AuditTimeseriesStat {
  timestamp: string;
  count: number;
}

export interface AuditUserStat {
  user_id: string;
  user_name: string;
  user_email: string;
  event_count: number;
  last_event_at: string;
}

export interface AuditResourceStat {
  resource_type: string;
  resource_id: string;
  resource_name: string;
  event_count: number;
}

export interface AuditLogStats {
  total_events: number;
  events_today: number;
  events_this_week: number;
  events_this_month: number;
  unique_users: number;
  unique_services: number;
  by_service: AuditGroupStat[];
  by_action: AuditGroupStat[];
  by_severity: AuditGroupStat[];
  by_hour: AuditTimeseriesStat[];
  by_day: AuditTimeseriesStat[];
  top_users: AuditUserStat[];
  top_resources: AuditResourceStat[];
}

export interface AuditStatsParams {
  group_by?: "service" | "action" | "user" | "tenant";
  date_from?: string;
  date_to?: string;
}

// ── Detail types ─────────────────────────────────────────────────────────────

export interface GeoLocation {
  country: string;
  city: string;
  latitude: number;
  longitude: number;
}

export interface AuditChange {
  field: string;
  old_value: unknown;
  new_value: unknown;
}

export interface AuditLogDetail extends Omit<AuditLog, "correlation_id"> {
  request_body: Record<string, unknown> | null;
  response_status: number | null;
  response_body: Record<string, unknown> | null;
  geo_location: GeoLocation | null;
  session_id: string | null;
  correlation_id: string | null;
  duration_ms: number | null;
  changes: AuditChange[];
}

// ── Timeline types ───────────────────────────────────────────────────────────

export interface AuditTimelineEvent {
  id: string;
  action: string;
  user_name: string;
  timestamp: string;
  changes: AuditChange[];
  summary: string;
}

export interface AuditTimeline {
  resource_id: string;
  resource_type: string;
  resource_name: string;
  events: AuditTimelineEvent[];
}

export interface AuditTimelineParams {
  action?: string;
  user_id?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  per_page?: number;
}

// ── Export types ──────────────────────────────────────────────────────────────

export interface AuditExportParams {
  format: "csv" | "ndjson";
  date_from: string;
  date_to?: string;
  service?: string;
  action?: string;
  user_id?: string;
  resource_type?: string;
  severity?: string;
  columns?: string[];
}

export interface AuditExportJobStatus {
  job_id: string;
  status: "queued" | "processing" | "completed" | "failed";
  poll_url?: string;
  download_url?: string;
  record_count?: number;
  error?: string;
}

// ── Verification types ───────────────────────────────────────────────────────

export interface AuditVerificationRequest {
  date_from?: string;
  date_to?: string;
}

export interface AuditVerificationResult {
  verified: boolean;
  total_records: number;
  verified_records: number;
  broken_chain_at: string | null;
  first_record: string;
  last_record: string;
  verification_hash: string;
  verified_at: string;
}

// ── Partition types ──────────────────────────────────────────────────────────

export type AuditPartitionStatus = "active" | "archived" | "pending";

export interface AuditPartition {
  id: string;
  name: string;
  date_range_start: string;
  date_range_end: string;
  record_count: number;
  size_bytes: number;
  status: AuditPartitionStatus;
  created_at: string;
}

export interface CreatePartitionRequest {
  name: string;
  date_range_start: string;
  date_range_end: string;
}
