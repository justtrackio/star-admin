import { useCallback, useEffect, useState } from 'react';
import { skipToken, useQuery, useQueryClient } from '@tanstack/react-query';

export function useSnapshotStream<T>(endpoint: string, validate: (value: unknown) => value is T) {
  const queryClient = useQueryClient();
  const [streamError, setStreamError] = useState<Error | null>(null);
  const [connectionVersion, setConnectionVersion] = useState(0);
  const [isFetching, setIsFetching] = useState(true);
  const refetch = useCallback(() => {
    setStreamError(null);
    setIsFetching(true);
    setConnectionVersion((version) => version + 1);
  }, []);
  const query = useQuery<T>({
    queryKey: [endpoint],
    queryFn: skipToken,
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
      const connection = new EventSource(`${baseUrl}${endpoint}`);
      source = connection;
      connection.addEventListener('message', (event: MessageEvent<string>) => {
        if (disposed) return;
        setIsFetching(false);
        try {
          const snapshot: unknown = JSON.parse(event.data);
          if (!validate(snapshot)) {
            throw new Error('Invalid live update');
          }
          queryClient.setQueryData<T>([endpoint], snapshot);
          setStreamError(null);
        } catch (error) {
          setStreamError(error instanceof Error ? error : new Error('Could not read live update'));
        }
      });
      connection.addEventListener('error', () => {
        if (disposed) return;
        setIsFetching(false);
        setStreamError(new Error('Live updates interrupted. Reconnecting automatically…'));
        // EventSource retries dropped streams itself, but HTTP errors can close it permanently.
        if (connection.readyState === EventSource.CLOSED) {
          clearTimeout(reconnectTimer);
          reconnectTimer = setTimeout(connect, 5000);
        }
      });
      connection.addEventListener('refresh-error', (event: MessageEvent<string>) => {
        if (disposed) return;
        setIsFetching(false);
        setStreamError(new Error(event.data || 'Could not refresh data'));
      });
    }
    connect();

    return () => {
      disposed = true;
      clearTimeout(reconnectTimer);
      source?.close();
    };
    // The version intentionally restarts the stream when the user requests a snapshot.
    // oxlint-disable-next-line react/exhaustive-effect-dependencies
  }, [queryClient, connectionVersion, endpoint, validate]);

  return {
    data: query.data,
    dataUpdatedAt: query.dataUpdatedAt,
    error: streamError,
    isPending: !query.data && !streamError,
    isFetching,
    refetch,
  };
}
