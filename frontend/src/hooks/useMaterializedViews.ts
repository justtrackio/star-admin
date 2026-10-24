import { isMaterializedViewsResponse } from '../api/materializedViews';
import { useSnapshotStream } from './useSnapshotStream';

export function useMaterializedViews() {
  return useSnapshotStream('/api/materialized-views', isMaterializedViewsResponse);
}

export function useMaterializedView(database: string, id: string) {
  return useSnapshotStream(`/api/materialized-views/${encodeURIComponent(database)}/${encodeURIComponent(id)}`, isMaterializedViewsResponse);
}
