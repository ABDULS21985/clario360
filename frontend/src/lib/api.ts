import axios, {
  AxiosError,
  InternalAxiosRequestConfig,
  AxiosResponse,
} from 'axios';
import { getAccessToken, setAccessToken, clearAccessToken } from '@/lib/auth';
import type { ApiError } from '@/types/api';

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

const api = axios.create({
  baseURL: API_URL,
  timeout: 30000,
});

// ── Request interceptor ──────────────────────────────────────────────────────

api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Attach access token (in-memory only — never localStorage)
    const token = getAccessToken();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    // Generate a unique request ID for tracing
    if (config.headers) {
      const reqId =
        typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
          ? crypto.randomUUID()
          : Math.random().toString(36).slice(2);
      config.headers['X-Request-ID'] = reqId;
    }

    // Set Content-Type only when body is present and not FormData
    if (
      config.data !== undefined &&
      config.data !== null &&
      !(config.data instanceof FormData) &&
      config.headers
    ) {
      config.headers['Content-Type'] = 'application/json';
    }

    return config;
  },
  (error: unknown) => Promise.reject(error),
);

// ── Token refresh queue (promise-based mutex) ────────────────────────────────

let isRefreshing = false;
let refreshSubscribers: Array<(token: string) => void> = [];
let refreshRejecters: Array<(err: unknown) => void> = [];

function subscribeTokenRefresh(onRefreshed: (token: string) => void, onFailed: (err: unknown) => void): void {
  refreshSubscribers.push(onRefreshed);
  refreshRejecters.push(onFailed);
}

function onRefreshSuccess(newToken: string): void {
  refreshSubscribers.forEach((cb) => cb(newToken));
  refreshSubscribers = [];
  refreshRejecters = [];
}

function onRefreshFailure(err: unknown): void {
  refreshRejecters.forEach((cb) => cb(err));
  refreshSubscribers = [];
  refreshRejecters = [];
}

// ── Response interceptor ─────────────────────────────────────────────────────

api.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // ── 401 handling ──────────────────────────────────────────────────────
    if (error.response?.status === 401) {
      const url = originalRequest?.url ?? '';

      // If the refresh route itself failed → clear session, redirect, stop
      if (url.includes('/api/auth/refresh') || url.includes('/api/v1/auth/refresh')) {
        clearAccessToken();
        if (typeof window !== 'undefined') {
          window.location.href = '/login';
        }
        return Promise.reject(buildApiError(error));
      }

      // Login failures are intentional 401s — don't retry
      if (url.includes('/api/v1/auth/login')) {
        return Promise.reject(buildApiError(error));
      }

      // Already retried — don't retry again
      if (originalRequest._retry) {
        return Promise.reject(buildApiError(error));
      }

      if (isRefreshing) {
        // Queue this request until the ongoing refresh completes
        return new Promise<AxiosResponse>((resolve, reject) => {
          subscribeTokenRefresh(
            (newToken) => {
              if (originalRequest.headers) {
                originalRequest.headers.Authorization = `Bearer ${newToken}`;
              }
              resolve(api(originalRequest));
            },
            (err) => reject(err),
          );
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // Call BFF refresh route — refresh token travels cookie-to-cookie,
        // frontend JS never sees it
        const { data } = await axios.post<{ access_token: string }>(
          '/api/auth/refresh',
          {},
          { withCredentials: true },
        );

        setAccessToken(data.access_token);
        onRefreshSuccess(data.access_token);

        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
        }
        return api(originalRequest);
      } catch (refreshError) {
        onRefreshFailure(refreshError);
        clearAccessToken();

        // Notify auth store that session expired (picked up by SessionExpiredDialog)
        if (typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('clario360:session-expired'));
        }
        return Promise.reject(buildApiError(error));
      } finally {
        isRefreshing = false;
      }
    }

    // ── Error transformation ──────────────────────────────────────────────
    return Promise.reject(buildApiError(error));
  },
);

function buildApiError(error: AxiosError): ApiError {
  if (!error.response) {
    if (error.code === 'ECONNABORTED') {
      return {
        status: 408,
        code: 'TIMEOUT',
        message: 'Request timed out. Please try again.',
      };
    }
    return {
      status: 0,
      code: 'NETWORK_ERROR',
      message: 'Unable to connect to server. Please check your connection.',
    };
  }

  const body = error.response.data as Record<string, unknown> | null;
  const status = error.response.status;

  return {
    status,
    code: (body?.['code'] as string | undefined) ?? `HTTP_${status}`,
    message:
      (body?.['message'] as string | undefined) ??
      (body?.['error'] as string | undefined) ??
      `Request failed with status ${status}`,
    details: (body?.['details'] as Record<string, string[]> | undefined) ?? undefined,
    request_id: (body?.['request_id'] as string | undefined) ?? undefined,
  };
}

// ── Typed helpers ─────────────────────────────────────────────────────────────

export async function apiGet<T>(url: string, params?: Record<string, unknown> | object): Promise<T> {
  const response = await api.get<T>(url, { params });
  return response.data;
}

export async function apiPost<T>(url: string, data?: unknown): Promise<T> {
  const response = await api.post<T>(url, data);
  return response.data;
}

export async function apiPut<T>(url: string, data?: unknown): Promise<T> {
  const response = await api.put<T>(url, data);
  return response.data;
}

export async function apiPatch<T>(url: string, data?: unknown): Promise<T> {
  const response = await api.patch<T>(url, data);
  return response.data;
}

export async function apiDelete<T>(url: string): Promise<T> {
  const response = await api.delete<T>(url);
  return response.data;
}

export async function apiUpload<T>(
  url: string,
  file: File,
  fields?: Record<string, string>,
  onUploadProgress?: (percent: number) => void,
): Promise<T> {
  const form = new FormData();
  form.append('file', file);
  if (fields) {
    Object.entries(fields).forEach(([k, v]) => form.append(k, v));
  }
  const response = await api.post<T>(url, form, {
    onUploadProgress: onUploadProgress
      ? (e) => {
          if (e.total) {
            onUploadProgress(Math.round((e.loaded * 100) / e.total));
          }
        }
      : undefined,
  });
  return response.data;
}

export default api;
