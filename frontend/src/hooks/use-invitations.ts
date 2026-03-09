"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { parseApiError } from "@/lib/format";
import type { AxiosError } from "axios";
import type { ApiError, PaginatedResponse } from "@/types/api";
import type {
  Invitation,
  CreateInvitationRequest,
  InvitationStats,
} from "@/types/invitation";
import type { FetchParams } from "@/types/table";

export function useInvitations(params?: FetchParams) {
  return useQuery<PaginatedResponse<Invitation>, AxiosError<ApiError>>({
    queryKey: ["invitations", params],
    queryFn: async () => {
      const { data } = await api.get<PaginatedResponse<Invitation>>("/api/v1/invitations", {
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

export function useInvitationStats() {
  return useQuery<InvitationStats, AxiosError<ApiError>>({
    queryKey: ["invitations", "stats"],
    queryFn: async () => {
      const { data } = await api.get<InvitationStats>("/api/v1/invitations/stats");
      return data;
    },
  });
}

export function useCreateInvitation() {
  const queryClient = useQueryClient();
  return useMutation<Invitation, AxiosError<ApiError>, CreateInvitationRequest>({
    mutationFn: async (payload) => {
      const { data } = await api.post<Invitation>("/api/v1/invitations", payload);
      return data;
    },
    onSuccess: () => {
      toast.success("Invitation sent");
      queryClient.invalidateQueries({ queryKey: ["invitations"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useResendInvitation() {
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (invitationId) => {
      await api.put(`/api/v1/invitations/${invitationId}/resend`);
    },
    onSuccess: () => {
      toast.success("Invitation resent");
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}

export function useDeleteInvitation() {
  const queryClient = useQueryClient();
  return useMutation<void, AxiosError<ApiError>, string>({
    mutationFn: async (invitationId) => {
      await api.delete(`/api/v1/invitations/${invitationId}`);
    },
    onSuccess: () => {
      toast.success("Invitation cancelled");
      queryClient.invalidateQueries({ queryKey: ["invitations"] });
    },
    onError: (error) => {
      toast.error(parseApiError(error));
    },
  });
}
