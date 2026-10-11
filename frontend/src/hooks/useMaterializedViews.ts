import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../api/client';
import type { MaterializedViewsResponse } from '../api/schema';
import { isMaterializedViewsResponse } from '../api/materializedViews';

export function useMaterializedViews() {
  const queryClient = useQueryClient();
  const [streamError, setStreamError] = useState<Error | null>(null);
  const query = useQuery({
    queryKey: ['materialized-views'],
    queryFn: () => apiClient.get<MaterializedViewsResponse>('/api/materialized-views'),
    staleTime: 10_000,
    enabled: false,
  });

  useEffect(() => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
    let source: EventSource | undefined;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let disposed = false;

    function connect() {
      if (disposed) return;
      const connection = new EventSource(`${baseUrl}/api/materialized-views/watch`);
      source = connection;
      connection.addEventListener('message', (event: MessageEvent<string>) => {
        if (disposed) return;
        try {
          const snapshot: unknown = JSON.parse(event.data);
          if (!isMaterializedViewsResponse(snapshot)) {
            throw new Error('Invalid materialized views update');
          }
          queryClient.setQueryData<MaterializedViewsResponse>(['materialized-views'], snapshot);
          setStreamError(null);
        } catch (error) {
          setStreamError(error instanceof Error ? error : new Error('Could not read live update'));
        }
      });
      connection.addEventListener('error', () => {
        if (disposed) return;
        setStreamError(new Error('Live updates interrupted. Reconnecting automatically…'));
        // EventSource retries dropped streams itself, but HTTP errors can close it permanently.
        if (connection.readyState === EventSource.CLOSED) {
          clearTimeout(reconnectTimer);
          reconnectTimer = setTimeout(connect, 5000);
        }
      });
      connection.addEventListener('refresh-error', (event: MessageEvent<string>) => {
        if (disposed) return;
        setStreamError(new Error(event.data || 'Could not refresh materialized views'));
      });
    }
    connect();

    return () => {
      disposed = true;
      clearTimeout(reconnectTimer);
      source?.close();
    };
  }, [queryClient]);

  return {
    ...query,
    error: streamError ?? query.error,
    isPending: query.isPending && !streamError,
  };
}
