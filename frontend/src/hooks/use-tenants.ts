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

export function useProvisionTenant() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, ProvisionTenantRequest>({
    mutationFn: async (payload) => {
      const { data } = await api.post<Tenant>("/api/v1/admin/tenants/provision", payload);
      return data;
    },
    onSuccess: () => {
      toast.success("Tenant provisioning started");
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useUpdateTenant() {
  const queryClient = useQueryClient();
  return useMutation<Tenant, AxiosError<ApiError>, { tenantId: string; data: Partial<Tenant> }>({
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

export function useDeprovisionTenant() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (tenantId) => {
      await api.post("/api/v1/admin/tenants/deprovision", { tenant_id: tenantId });
    },
    onSuccess: () => {
      toast.success("Tenant deprovisioning started");
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useTenantUsage(tenantId: string) {
  return useQuery<TenantUsage, AxiosError<ApiError>>({
    queryKey: ["tenants", tenantId, "usage"],
    queryFn: async () => {
      const { data } = await api.get<TenantUsage>(`/api/v1/tenants/${tenantId}/usage`);
      return data;
    },
    enabled: !!tenantId,
  });
}

export function useTenantSettings(tenantId: string) {
  return useQuery<TenantSettings, AxiosError<ApiError>>({
    queryKey: ["tenants", tenantId, "settings"],
    queryFn: async () => {
      const { data } = await api.get<TenantSettings>(`/api/v1/tenants/${tenantId}/settings`);
      return data;
    },
    enabled: !!tenantId,
  });
}

export function useUpdateTenantSettings() {
  const queryClient = useQueryClient();
  return useMutation<TenantSettings, AxiosError<ApiError>, { tenantId: string; settings: Partial<TenantSettings> }>({
    mutationFn: async ({ tenantId, settings }) => {
      const { data } = await api.put<TenantSettings>(`/api/v1/tenants/${tenantId}/settings`, settings);
      return data;
    },
    onSuccess: (_, variables) => {
      toast.success("Tenant settings updated");
      queryClient.invalidateQueries({ queryKey: ["tenants", variables.tenantId, "settings"] });
      queryClient.invalidateQueries({ queryKey: ["tenants", variables.tenantId] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}
