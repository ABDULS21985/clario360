"use client";

import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from "@tanstack/react-query";
import api from "@/lib/api";
import type { AxiosError } from "axios";
import type { ApiError } from "@/types/api";

export function useApiQuery<T>(
  key: string[],
  url: string,
  options?: Omit<UseQueryOptions<T, AxiosError<ApiError>>, "queryKey" | "queryFn">
) {
  return useQuery<T, AxiosError<ApiError>>({
    queryKey: key,
    queryFn: async () => {
      const { data } = await api.get<T>(url);
      return data;
    },
    ...options,
  });
}

export function useApiMutation<TData, TVariables>(
  url: string,
  method: "post" | "put" | "patch" | "delete" = "post",
  options?: UseMutationOptions<TData, AxiosError<ApiError>, TVariables>
) {
  return useMutation<TData, AxiosError<ApiError>, TVariables>({
    mutationFn: async (variables) => {
      const { data } = await api[method]<TData>(url, variables);
      return data;
    },
    ...options,
  });
}
