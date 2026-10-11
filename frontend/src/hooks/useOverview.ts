import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../api/client';
import type { OverviewResponse } from '../api/schema';

export function useOverview() {
  return useQuery({
    queryKey: ['overview'],
    queryFn: () => apiClient.get<OverviewResponse>('/api/overview'),
    refetchInterval: 15_000,
    staleTime: 10_000,
  });
}
