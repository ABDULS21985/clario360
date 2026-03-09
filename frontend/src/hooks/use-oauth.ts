"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { parseApiError } from "@/lib/format";
import type { AxiosError } from "axios";
import type { ApiError } from "@/types/api";
import type { OAuthProvider, OAuthConnection } from "@/types/oauth";

export function useOAuthProviders() {
  return useQuery<OAuthProvider[], AxiosError<ApiError>>({
    queryKey: ["oauth-providers"],
    queryFn: async () => {
      const { data } = await api.get<OAuthProvider[]>("/api/v1/auth/oauth/providers");
      return data;
    },
  });
}

export function useOAuthConnections() {
  return useQuery<OAuthConnection[], AxiosError<ApiError>>({
    queryKey: ["oauth-connections"],
    queryFn: async () => {
      const { data } = await api.get<OAuthConnection[]>("/api/v1/auth/oauth/connections");
      return data;
    },
  });
}

export function useLinkOAuth() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, { provider: string; code: string; state: string }>({
    mutationFn: async (payload) => {
      await api.post("/api/v1/auth/oauth/link", payload);
    },
    onSuccess: () => {
      toast.success("Account linked successfully");
      queryClient.invalidateQueries({ queryKey: ["oauth-connections"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useUnlinkOAuth() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (provider) => {
      await api.delete(`/api/v1/auth/oauth/link/${provider}`);
    },
    onSuccess: () => {
      toast.success("Account unlinked");
      queryClient.invalidateQueries({ queryKey: ["oauth-connections"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function getOAuthAuthorizeUrl(provider: string): string {
  const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  return `${baseUrl}/api/v1/auth/oauth/${provider}/authorize`;
}
