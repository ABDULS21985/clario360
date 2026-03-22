import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAsset } from '@/types/cyber';

/**
 * Resolves asset UUIDs to display names.
 * Fetches from `GET /api/v1/cyber/assets` with per-page large enough to cover
 * the IDs, then builds a name map. Results are cached by React Query.
 */
export function useAssetNames(assetIds: string[]): Record<string, string> {
  const uniqueIds = [...new Set(assetIds.filter(Boolean))];

  const { data } = useQuery({
    queryKey: ['cyber-asset-names', uniqueIds.sort().join(',')],
    queryFn: async () => {
      if (uniqueIds.length === 0) return {};
      const nameMap: Record<string, string> = {};
      // Fetch individually (no batch endpoint) — React Query deduplicates and caches
      const results = await Promise.allSettled(
        uniqueIds.map((id) =>
          apiGet<{ data: CyberAsset }>(`${API_ENDPOINTS.CYBER_ASSETS}/${id}`),
        ),
      );
      results.forEach((result, idx) => {
        if (result.status === 'fulfilled' && result.value?.data) {
          nameMap[uniqueIds[idx]] = result.value.data.name;
        }
      });
      return nameMap;
    },
    enabled: uniqueIds.length > 0,
    staleTime: 5 * 60 * 1000, // cache for 5 minutes
  });

  return data ?? {};
}
