import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../api/client';
import type { HealthResponse } from '../api/schema';

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => apiClient.get<HealthResponse>('/api/health'),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}
