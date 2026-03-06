"use client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import type { AxiosError } from "axios";
import type { ApiError } from "@/types/api";
import { parseApiError } from "@/lib/format";

interface MutationOptions<TData, TVariables> {
  onSuccess?: (data: TData) => void;
  onError?: (error: AxiosError<ApiError>) => void;
  invalidateKeys?: string[];
  successMessage?: string;
  errorMessage?: string;
}

export function useApiMutation<TData = unknown, TVariables = unknown>(
  method: "post" | "put" | "patch" | "delete",
  url: string | ((variables: TVariables) => string),
  options: MutationOptions<TData, TVariables> = {}
) {
  const queryClient = useQueryClient();
  const { onSuccess, onError, invalidateKeys = [], successMessage, errorMessage } = options;

  return useMutation<TData, AxiosError<ApiError>, TVariables>({
    mutationFn: async (variables) => {
      const resolvedUrl = typeof url === "function" ? url(variables) : url;
      const { data } = method === "delete"
        ? await api.delete<TData>(resolvedUrl)
        : await api[method]<TData>(resolvedUrl, variables);
      return data;
    },
    onSuccess: (data) => {
      if (successMessage) toast.success(successMessage);
      if (invalidateKeys.length > 0) {
        invalidateKeys.forEach((key) => queryClient.invalidateQueries({ queryKey: [key] }));
      }
      onSuccess?.(data);
    },
    onError: (error) => {
      const message = errorMessage ?? parseApiError(error);
      toast.error(message);
      onError?.(error);
    },
  });
}
