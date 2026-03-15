"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { parseApiError } from "@/lib/format";
import type { AxiosError } from "axios";
import type { ApiError, PaginatedResponse } from "@/types/api";
import type {
  ApiKey,
  CreateApiKeyRequest,
  CreateApiKeyResponse,
} from "@/types/api-key";
import type { FetchParams } from "@/types/table";

export function useApiKeys(params?: FetchParams) {
  return useQuery<PaginatedResponse<ApiKey>, AxiosError<ApiError>>({
    queryKey: ["api-keys-admin", params],
    queryFn: async () => {
      const { data } = await api.get<PaginatedResponse<ApiKey>>("/api/v1/api-keys", {
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

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  return useMutation<CreateApiKeyResponse, AxiosError<ApiError>, CreateApiKeyRequest>({
    mutationFn: async (payload) => {
      const { data } = await api.post<CreateApiKeyResponse>("/api/v1/api-keys", payload);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys-admin"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useRevokeApiKey() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (keyId) => {
      await api.delete(`/api/v1/api-keys/${keyId}`);
    },
    onSuccess: () => {
      toast.success("API key revoked");
      queryClient.invalidateQueries({ queryKey: ["api-keys-admin"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useRotateApiKey() {
  const queryClient = useQueryClient();
  return useMutation<CreateApiKeyResponse, AxiosError<ApiError>, string>({
    mutationFn: async (keyId) => {
      const { data } = await api.post<CreateApiKeyResponse>(`/api/v1/api-keys/${keyId}/rotate`);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys-admin"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}
