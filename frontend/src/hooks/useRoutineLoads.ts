import { isRoutineLoadsResponse } from '../api/snapshots';
import { useSnapshotStream } from './useSnapshotStream';

export function useRoutineLoads() {
  return useSnapshotStream('/api/routine-loads', isRoutineLoadsResponse);
}
