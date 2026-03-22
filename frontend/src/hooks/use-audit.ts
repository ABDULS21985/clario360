"use client";

import { useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { normalizeAuditStatsParams } from "@/lib/audit";
import { downloadBlob, parseApiError } from "@/lib/format";
import type { AxiosError } from "axios";
import type { ApiError } from "@/types/api";
import type {
  AuditLogStats,
  AuditStatsParams,
  AuditLogDetail,
  AuditTimeline,
  AuditTimelineParams,
  AuditExportParams,
  AuditVerificationRequest,
  AuditVerificationResult,
  AuditPartition,
} from "@/types/audit";

// ── Statistics ───────────────────────────────────────────────────────────────

export function useAuditStats(params?: AuditStatsParams) {
  const normalizedParams = useMemo(
    () => normalizeAuditStatsParams(params),
    [params?.date_from, params?.date_to, params?.group_by]
  );

  return useQuery<AuditLogStats, AxiosError<ApiError>>({
    queryKey: ["audit-stats", normalizedParams],
    queryFn: async () => {
      const { data } = await api.get<AuditLogStats>(
        "/api/v1/audit/logs/stats",
        { params: normalizedParams }
      );
      return data;
    },
    staleTime: 60_000,
  });
}

// ── Detail ───────────────────────────────────────────────────────────────────

export function useAuditLogDetail(logId: string) {
  return useQuery<AuditLogDetail, AxiosError<ApiError>>({
    queryKey: ["audit-log-detail", logId],
    queryFn: async () => {
      const { data } = await api.get<AuditLogDetail>(
        `/api/v1/audit/logs/${logId}`
      );
      return data;
    },
    enabled: !!logId,
  });
}

// ── Timeline ─────────────────────────────────────────────────────────────────

export function useAuditTimeline(
  resourceId: string,
  params?: AuditTimelineParams
) {
  return useQuery<AuditTimeline, AxiosError<ApiError>>({
    queryKey: ["audit-timeline", resourceId, params],
    queryFn: async () => {
      const { data } = await api.get<AuditTimeline>(
        `/api/v1/audit/logs/timeline/${resourceId}`,
        { params }
      );
      return data;
    },
    enabled: !!resourceId,
  });
}

// ── Export ────────────────────────────────────────────────────────────────────

export function useAuditExport() {
  return useMutation<void, AxiosError<ApiError>, AuditExportParams>({
    mutationFn: async (params) => {
      // Backend expects columns as a single comma-separated string, not an array
      const queryParams: Record<string, string | undefined> = {
        format: params.format,
        date_from: params.date_from,
        date_to: params.date_to,
        service: params.service,
        action: params.action,
        user_id: params.user_id,
        resource_type: params.resource_type,
        severity: params.severity,
        columns: params.columns?.length ? params.columns.join(",") : undefined,
      };
      const response = await api.get("/api/v1/audit/logs/export", {
        params: queryParams,
        responseType: "blob",
      });
      const ext = params.format === "csv" ? "csv" : "ndjson";
      const filename = `audit-logs-export.${ext}`;
      downloadBlob(response.data as Blob, filename);
    },
    onSuccess: () => {
      toast.success("Audit logs exported successfully");
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

// ── Verification ─────────────────────────────────────────────────────────────

export function useAuditVerify() {
  return useMutation<
    AuditVerificationResult,
    AxiosError<ApiError>,
    AuditVerificationRequest
  >({
    mutationFn: async (body) => {
      const { data } = await api.post<AuditVerificationResult>(
        "/api/v1/audit/verify",
        body
      );
      return data;
    },
  });
}

// ── Partitions ───────────────────────────────────────────────────────────────

export function useAuditPartitions() {
  return useQuery<AuditPartition[], AxiosError<ApiError>>({
    queryKey: ["audit-partitions"],
    queryFn: async () => {
      const { data } = await api.get<AuditPartition[]>(
        "/api/v1/audit/partitions"
      );
      return data;
    },
  });
}

export function useCreateAuditPartition() {
  const queryClient = useQueryClient();
  return useMutation<AuditPartition[], AxiosError<ApiError>, void>({
    mutationFn: async () => {
      const { data } = await api.post<AuditPartition[]>(
        "/api/v1/audit/partitions"
      );
      return data;
    },
    onSuccess: (data) => {
      toast.success("Partition maintenance completed");
      queryClient.setQueryData(["audit-partitions"], data);
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useArchiveAuditPartition() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (partitionId) => {
      await api.post(`/api/v1/audit/partitions/${partitionId}/archive`);
    },
    onSuccess: () => {
      toast.success("Partition archived successfully");
      queryClient.invalidateQueries({ queryKey: ["audit-partitions"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useDeleteAuditPartition() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (partitionId) => {
      await api.delete(`/api/v1/audit/partitions/${partitionId}`);
    },
    onSuccess: () => {
      toast.success("Partition deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["audit-partitions"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}
