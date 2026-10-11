import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../api/client';
import type { MaterializedViewsResponse } from '../api/schema';

export function useMaterializedViews() {
  return useQuery({
    queryKey: ['materialized-views'],
    queryFn: () => apiClient.get<MaterializedViewsResponse>('/api/materialized-views'),
    staleTime: 10_000,
  });
}
