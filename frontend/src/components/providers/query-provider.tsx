'use client';

import React, { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 60 * 1000, // 5 min — prevents refetch storms on navigation
            gcTime: 10 * 60 * 1000, // 10 min garbage collection
            refetchOnWindowFocus: false, // disable global refetch-on-focus
            retry: (failureCount, error) => {
              // Don't retry 401/403/404 errors
              if (
                error &&
                typeof error === 'object' &&
                'status' in error &&
                [401, 403, 404].includes(Number((error as { status: number }).status))
              ) {
                return false;
              }
              return failureCount < 2;
            },
          },
        },
      }),
  );

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
