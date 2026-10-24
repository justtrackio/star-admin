import { isDictionariesResponse } from '../api/snapshots';
import { useSnapshotStream } from './useSnapshotStream';

export function useDictionaries() {
  return useSnapshotStream('/api/dictionaries', isDictionariesResponse);
}
