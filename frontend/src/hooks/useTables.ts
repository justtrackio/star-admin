import { isTablesResponse } from '../api/snapshots';
import { useSnapshotStream } from './useSnapshotStream';

export function useTables() {
  return useSnapshotStream('/api/tables', isTablesResponse);
}
