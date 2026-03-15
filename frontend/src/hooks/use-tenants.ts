"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { parseApiError } from "@/lib/format";
import type { AxiosError } from "axios";
import type { ApiError, PaginatedResponse } from "@/types/api";
import type {
  Tenant,
  TenantSettings,
  TenantUsage,
  ProvisionTenantRequest,
} from "@/types/tenant";
import type { FetchParams } from "@/types/table";

export function useTenants(params?: FetchParams) {
  return useQuery<PaginatedResponse<Tenant>, AxiosError<ApiError>>({
    queryKey: ["tenants", params],
    queryFn: async () => {
      const { data } = await api.get<PaginatedResponse<Tenant>>("/api/v1/tenants", {
        params: {
          page: params?.page,
          per_page: params?.per_page,
          sort: params?.sort,
          order: params?.order,
          search: params?.search || undefined,
          status: params?.filters?.status,
          subscription_tier: params?.filters?.subscription_tier,
        },
      });
      return data;
    },
  });
}

export function useTenant(tenantId: string, polling = false) {
  return useQuery<Tenant, AxiosError<ApiError>>({
    queryKey: ["tenants", tenantId],
    queryFn: async () => {
      const { data } = await api.get<Tenant>(`/api/v1/tenants/${tenantId}`);
      return data;
    },
    enabled: !!tenantId,
    refetchInterval: polling ? 3000 : false,
  });
}

/**
 * Provision a new tenant via POST /api/v1/tenants (CreateTenantRequest).
 * owner_email/owner_name are embedded in the settings JSON.
 */
export function useProvisionTenant() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, ProvisionTenantRequest>({
    mutationFn: async (payload) => {
      const { data } = await api.post<Tenant>("/api/v1/tenants", payload);
      return data;
    },
    onSuccess: () => {
      toast.success("Tenant created successfully");
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useUpdateTenant() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, { tenantId: string; data: Record<string, unknown> }>({
    mutationFn: async ({ tenantId, data: payload }) => {
      const { data } = await api.put<Tenant>(`/api/v1/tenants/${tenantId}`, payload);
      return data;
    },
    onSuccess: (_, variables) => {
      toast.success("Tenant updated");
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
      queryClient.invalidateQueries({ queryKey: ["tenants", variables.tenantId] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

/**
 * Deprovision a tenant by setting status to "deprovisioned" via PUT /api/v1/tenants/{id}.
 * The backend UpdateTenantRequest accepts status changes.
 */
export function useDeprovisionTenant() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, string>({
    mutationFn: async (tenantId) => {
      const { data } = await api.put<Tenant>(`/api/v1/tenants/${tenantId}`, {
        status: "deprovisioned",
      });
      return data;
    },
    onSuccess: () => {
      toast.success("Tenant deprovisioned");
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

/**
 * Usage endpoint — may not exist yet. Gracefully returns undefined on 404.
 */
export function useTenantUsage(tenantId: string) {
  return useQuery<TenantUsage | null, AxiosError<ApiError>>({
    queryKey: ["tenants", tenantId, "usage"],
    queryFn: async () => {
      try {
        const { data } = await api.get<TenantUsage>(`/api/v1/tenants/${tenantId}/usage`);
        return data;
      } catch {
        return null;
      }
    },
    enabled: !!tenantId,
  });
}

/**
 * Update tenant settings by writing to the tenant's settings JSONB field
 * via PUT /api/v1/tenants/{id} with { settings: {...} }.
 */
export function useUpdateTenantSettings() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, { tenantId: string; settings: Partial<TenantSettings> }>({
    mutationFn: async ({ tenantId, settings }) => {
      const { data } = await api.put<Tenant>(`/api/v1/tenants/${tenantId}`, {
        settings,
      });
      return data;
    },
    onSuccess: (_, variables) => {
      toast.success("Tenant settings updated");
      queryClient.invalidateQueries({ queryKey: ["tenants", variables.tenantId] });
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}
